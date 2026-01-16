package micropub

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pojntfx/felicitas.pojtinger.com/api/indieauth"
	"github.com/pojntfx/felicitas.pojtinger.com/api/webmention"
)

// MicropubRequest represents a Micropub create request in JSON format
type MicropubRequest struct {
	Type       []string               `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// ConfigResponse represents the Micropub configuration query response
type ConfigResponse struct {
	MediaEndpoint string            `json:"media-endpoint,omitempty"`
	PostTypes     []PostTypeConfig  `json:"post-types,omitempty"`
	SyndicateTo   []SyndicateTarget `json:"syndicate-to,omitempty"`
}

// PostTypeConfig describes a supported post type
type PostTypeConfig struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// SyndicateTarget describes a syndication target
type SyndicateTarget struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}

// MicropubHandler handles Micropub requests
func MicropubHandler(rw http.ResponseWriter, r *http.Request, contentDir, baseURL string) {
	rw.Header().Set("Content-Type", "application/json")

	// Handle GET requests for configuration queries
	if r.Method == http.MethodGet {
		handleQuery(rw, r, baseURL)
		return
	}

	// POST requires authentication
	if r.Method != http.MethodPost {
		http.Error(rw, `{"error": "method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Extract and verify token
	token := indieauth.ExtractToken(r)
	if token == "" {
		rw.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(rw).Encode(map[string]string{
			"error":             "unauthorized",
			"error_description": "No access token provided",
		})
		return
	}

	tokenResp, err := indieauth.VerifyToken(
		token,
		indieauth.GetTokenEndpoint(),
		indieauth.GetExpectedMe(),
	)
	if err != nil {
		rw.WriteHeader(http.StatusForbidden)
		json.NewEncoder(rw).Encode(map[string]string{
			"error":             "forbidden",
			"error_description": err.Error(),
		})
		return
	}

	// Check for create scope
	if !indieauth.HasScope(tokenResp, "create") {
		rw.WriteHeader(http.StatusForbidden)
		json.NewEncoder(rw).Encode(map[string]string{
			"error":             "insufficient_scope",
			"error_description": "Token does not have 'create' scope",
		})
		return
	}

	// Parse the request based on content type
	var req MicropubRequest
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(map[string]string{
				"error":             "invalid_request",
				"error_description": "Failed to parse JSON: " + err.Error(),
			})
			return
		}
	} else {
		// Form-encoded request
		if err := r.ParseForm(); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(map[string]string{
				"error":             "invalid_request",
				"error_description": "Failed to parse form: " + err.Error(),
			})
			return
		}
		req = formToMicropub(r.Form)
	}

	// Determine post type
	postType := determinePostType(req)

	// Create the post
	location, err := createPost(req, postType, contentDir, baseURL)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(rw).Encode(map[string]string{
			"error":             "server_error",
			"error_description": "Failed to create post: " + err.Error(),
		})
		return
	}

	// Return success with Location header
	rw.Header().Set("Location", location)
	rw.WriteHeader(http.StatusCreated)

	// Send webmentions asynchronously after response
	go func() {
		// Wait a bit for Hugo to rebuild
		time.Sleep(5 * time.Second)

		log.Printf("Sending webmentions for: %s", location)
		response, err := webmention.SendWebmentions(location)
		if err != nil {
			log.Printf("Webmention send error: %v", err)
			return
		}

		successCount := 0
		for _, result := range response.Results {
			if result.Success {
				successCount++
			}
		}
		log.Printf("Webmentions sent: %d/%d successful", successCount, len(response.Results))
	}()
}

// handleQuery handles Micropub configuration queries
func handleQuery(rw http.ResponseWriter, r *http.Request, baseURL string) {
	q := r.URL.Query().Get("q")

	switch q {
	case "config":
		config := ConfigResponse{
			MediaEndpoint: baseURL + "/api/micropub/media",
			PostTypes: []PostTypeConfig{
				{Type: "note", Name: "Note"},
				{Type: "article", Name: "Article"},
				{Type: "like", Name: "Like"},
				{Type: "bookmark", Name: "Bookmark"},
				{Type: "repost", Name: "Repost"},
			},
			SyndicateTo: []SyndicateTarget{}, // No syndication targets configured yet
		}
		json.NewEncoder(rw).Encode(config)

	case "syndicate-to":
		json.NewEncoder(rw).Encode(map[string][]SyndicateTarget{
			"syndicate-to": {},
		})

	case "source":
		// Source query not implemented yet
		rw.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(rw).Encode(map[string]string{
			"error": "not_implemented",
		})

	default:
		rw.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(rw).Encode(map[string]string{
			"error":             "invalid_request",
			"error_description": "Unknown query: " + q,
		})
	}
}

// formToMicropub converts form-encoded data to MicropubRequest format
func formToMicropub(form url.Values) MicropubRequest {
	req := MicropubRequest{
		Properties: make(map[string]interface{}),
	}

	// Handle h parameter (type)
	if h := form.Get("h"); h != "" {
		req.Type = []string{"h-" + h}
	} else {
		req.Type = []string{"h-entry"}
	}

	// Map form fields to properties
	fieldMappings := map[string]string{
		"content":     "content",
		"name":        "name",
		"summary":     "summary",
		"category":    "category",
		"category[]":  "category",
		"like-of":     "like-of",
		"bookmark-of": "bookmark-of",
		"repost-of":   "repost-of",
		"in-reply-to": "in-reply-to",
		"mp-slug":     "mp-slug",
		"photo":       "photo",
		"photo[]":     "photo",
	}

	for formKey, propKey := range fieldMappings {
		if values, ok := form[formKey]; ok && len(values) > 0 {
			if len(values) == 1 {
				req.Properties[propKey] = []interface{}{values[0]}
			} else {
				iValues := make([]interface{}, len(values))
				for i, v := range values {
					iValues[i] = v
				}
				req.Properties[propKey] = iValues
			}
		}
	}

	return req
}

// determinePostType determines the type of post from the request
func determinePostType(req MicropubRequest) string {
	props := req.Properties

	if _, ok := props["like-of"]; ok {
		return "like"
	}
	if _, ok := props["bookmark-of"]; ok {
		return "bookmark"
	}
	if _, ok := props["repost-of"]; ok {
		return "repost"
	}
	if _, ok := props["in-reply-to"]; ok {
		return "reply"
	}
	if _, ok := props["name"]; ok {
		return "article"
	}
	return "note"
}

// createPost creates a Hugo content file from the Micropub request
func createPost(req MicropubRequest, postType, contentDir, baseURL string) (string, error) {
	now := time.Now()

	// Generate slug
	slug := generateSlug(req, now)

	// Determine content directory based on type
	var typeDir string
	switch postType {
	case "note":
		typeDir = "notes"
	case "like":
		typeDir = "likes"
	case "bookmark":
		typeDir = "bookmarks"
	case "repost":
		typeDir = "reposts"
	case "reply":
		typeDir = "replies"
	case "article":
		typeDir = "articles"
	default:
		typeDir = "notes"
	}

	// Create post directory
	postDir := filepath.Join(contentDir, typeDir, slug)
	if err := os.MkdirAll(postDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate frontmatter
	frontmatter := generateFrontmatter(req, postType, now)

	// Get content
	content := extractContent(req)

	// Write file
	filePath := filepath.Join(postDir, "index.md")
	fileContent := fmt.Sprintf("---\n%s---\n\n%s\n", frontmatter, content)

	if err := os.WriteFile(filePath, []byte(fileContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Return URL
	return fmt.Sprintf("%s/%s/%s/", baseURL, typeDir, slug), nil
}

// generateSlug creates a URL-safe slug for the post
func generateSlug(req MicropubRequest, t time.Time) string {
	props := req.Properties

	// Check for explicit slug
	if slugs, ok := props["mp-slug"]; ok {
		if slugArr, ok := slugs.([]interface{}); ok && len(slugArr) > 0 {
			if s, ok := slugArr[0].(string); ok {
				return sanitizeSlug(s)
			}
		}
	}

	// Use name if available
	if names, ok := props["name"]; ok {
		if nameArr, ok := names.([]interface{}); ok && len(nameArr) > 0 {
			if s, ok := nameArr[0].(string); ok {
				return sanitizeSlug(s)
			}
		}
	}

	// Fall back to timestamp
	return t.Format("2006-01-02-150405")
}

// sanitizeSlug converts a string to a URL-safe slug
func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)

	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove non-alphanumeric characters except hyphens
	reg := regexp.MustCompile(`[^a-z0-9\-]`)
	s = reg.ReplaceAllString(s, "")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	s = reg.ReplaceAllString(s, "-")

	// Trim hyphens from start and end
	s = strings.Trim(s, "-")

	// Limit length
	if len(s) > 50 {
		s = s[:50]
		s = strings.TrimRight(s, "-")
	}

	return s
}

// generateFrontmatter creates Hugo frontmatter from the request
func generateFrontmatter(req MicropubRequest, postType string, t time.Time) string {
	var lines []string
	props := req.Properties

	// Title (for articles)
	if names, ok := props["name"]; ok {
		if nameArr, ok := names.([]interface{}); ok && len(nameArr) > 0 {
			if s, ok := nameArr[0].(string); ok {
				lines = append(lines, fmt.Sprintf("title: %q", s))
			}
		}
	}

	// Date
	lines = append(lines, fmt.Sprintf("date: %s", t.Format(time.RFC3339)))

	// Post type
	lines = append(lines, fmt.Sprintf("type: %s", postType))

	// Summary (for articles)
	if summaries, ok := props["summary"]; ok {
		if sumArr, ok := summaries.([]interface{}); ok && len(sumArr) > 0 {
			if s, ok := sumArr[0].(string); ok {
				lines = append(lines, fmt.Sprintf("excerpt: %q", s))
			}
		}
	}

	// Type-specific properties
	switch postType {
	case "like":
		if urls, ok := props["like-of"]; ok {
			if urlArr, ok := urls.([]interface{}); ok && len(urlArr) > 0 {
				if s, ok := urlArr[0].(string); ok {
					lines = append(lines, fmt.Sprintf("likeOf: %q", s))
				}
			}
		}
	case "bookmark":
		if urls, ok := props["bookmark-of"]; ok {
			if urlArr, ok := urls.([]interface{}); ok && len(urlArr) > 0 {
				if s, ok := urlArr[0].(string); ok {
					lines = append(lines, fmt.Sprintf("bookmarkOf: %q", s))
				}
			}
		}
	case "repost":
		if urls, ok := props["repost-of"]; ok {
			if urlArr, ok := urls.([]interface{}); ok && len(urlArr) > 0 {
				if s, ok := urlArr[0].(string); ok {
					lines = append(lines, fmt.Sprintf("repostOf: %q", s))
				}
			}
		}
	case "reply":
		if urls, ok := props["in-reply-to"]; ok {
			if urlArr, ok := urls.([]interface{}); ok && len(urlArr) > 0 {
				if s, ok := urlArr[0].(string); ok {
					lines = append(lines, fmt.Sprintf("inReplyTo: %q", s))
				}
			}
		}
	}

	// Categories/tags
	if cats, ok := props["category"]; ok {
		if catArr, ok := cats.([]interface{}); ok && len(catArr) > 0 {
			var tags []string
			for _, c := range catArr {
				if s, ok := c.(string); ok {
					tags = append(tags, fmt.Sprintf("%q", s))
				}
			}
			if len(tags) > 0 {
				lines = append(lines, fmt.Sprintf("tags: [%s]", strings.Join(tags, ", ")))
			}
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

// extractContent extracts the content from the request
func extractContent(req MicropubRequest) string {
	props := req.Properties

	if contents, ok := props["content"]; ok {
		if contentArr, ok := contents.([]interface{}); ok && len(contentArr) > 0 {
			switch v := contentArr[0].(type) {
			case string:
				return v
			case map[string]interface{}:
				// HTML content
				if html, ok := v["html"].(string); ok {
					return html
				}
				// Text content
				if text, ok := v["text"].(string); ok {
					return text
				}
			}
		}
	}

	return ""
}
