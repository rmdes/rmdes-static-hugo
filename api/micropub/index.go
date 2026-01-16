package micropub

import (
	"context"
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
	"github.com/pojntfx/felicitas.pojtinger.com/api/syndication"
	"github.com/pojntfx/felicitas.pojtinger.com/api/webmention"
)

// syndicationManager is set by the API server to enable cross-posting
var syndicationManager *syndication.Manager

// SetSyndicationManager sets the syndication manager for cross-posting
func SetSyndicationManager(m *syndication.Manager) {
	syndicationManager = m
}

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

	// Extract syndication targets
	syndicationTargets := extractSyndicationTargets(req)

	// Create the post
	location, filePath, err := createPost(req, postType, contentDir, baseURL)
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

	// Handle syndication and webmentions asynchronously
	go func() {
		// Wait a bit for Hugo to rebuild
		time.Sleep(5 * time.Second)

		// Syndicate to selected targets
		if syndicationManager != nil && len(syndicationTargets) > 0 {
			log.Printf("Syndicating to: %v", syndicationTargets)

			syndicationPost := createSyndicationPost(req, postType, location)
			results := syndicationManager.Syndicate(context.Background(), syndicationPost, syndicationTargets)

			// Update frontmatter with syndication URLs
			syndicationURLs := syndication.ResultsToSyndicationURLs(results)
			if len(syndicationURLs) > 0 {
				if err := appendSyndicationToFrontmatter(filePath, syndicationURLs); err != nil {
					log.Printf("Failed to update frontmatter with syndication URLs: %v", err)
				}
			}

			for _, result := range results {
				if result.Success {
					log.Printf("Syndicated to %s: %s", result.Platform, result.URL)
				} else {
					log.Printf("Syndication to %s failed: %s", result.Platform, result.Error)
				}
			}
		}

		// Send webmentions
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

// extractSyndicationTargets extracts syndication target UIDs from the request
func extractSyndicationTargets(req MicropubRequest) []string {
	props := req.Properties

	if targets, ok := props["mp-syndicate-to"]; ok {
		if targetArr, ok := targets.([]interface{}); ok {
			var result []string
			for _, t := range targetArr {
				if s, ok := t.(string); ok {
					result = append(result, s)
				}
			}
			return syndication.ParseSyndicationTargets(result)
		}
	}

	return nil
}

// createSyndicationPost converts a Micropub request to a syndication Post
func createSyndicationPost(req MicropubRequest, postType, postURL string) *syndication.Post {
	post := &syndication.Post{
		URL:  postURL,
		Type: syndication.PostType(postType),
	}

	props := req.Properties

	// Extract content
	post.Content = extractContent(req)

	// Extract title
	if names, ok := props["name"]; ok {
		if nameArr, ok := names.([]interface{}); ok && len(nameArr) > 0 {
			if s, ok := nameArr[0].(string); ok {
				post.Title = s
			}
		}
	}

	// Extract target URL for bookmarks, likes, reposts, replies
	for _, key := range []string{"bookmark-of", "like-of", "repost-of", "in-reply-to"} {
		if urls, ok := props[key]; ok {
			if urlArr, ok := urls.([]interface{}); ok && len(urlArr) > 0 {
				if s, ok := urlArr[0].(string); ok {
					post.TargetURL = s
					break
				}
			}
		}
	}

	// Extract tags
	if cats, ok := props["category"]; ok {
		if catArr, ok := cats.([]interface{}); ok {
			for _, c := range catArr {
				if s, ok := c.(string); ok {
					post.Tags = append(post.Tags, s)
				}
			}
		}
	}

	return post
}

// appendSyndicationToFrontmatter adds syndication URLs to a post's frontmatter
func appendSyndicationToFrontmatter(filePath string, syndicationURLs map[string]string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	fileContent := string(content)

	// Find the end of frontmatter
	parts := strings.SplitN(fileContent, "---", 3)
	if len(parts) < 3 {
		return fmt.Errorf("invalid frontmatter format")
	}

	// Build syndication YAML
	var syndicationLines []string
	syndicationLines = append(syndicationLines, "syndication:")
	for platform, url := range syndicationURLs {
		syndicationLines = append(syndicationLines, fmt.Sprintf("  %s: %q", platform, url))
	}

	// Insert syndication before the closing ---
	newFrontmatter := strings.TrimRight(parts[1], "\n") + "\n" + strings.Join(syndicationLines, "\n") + "\n"
	newContent := "---" + newFrontmatter + "---" + parts[2]

	return os.WriteFile(filePath, []byte(newContent), 0644)
}

// handleQuery handles Micropub configuration queries
func handleQuery(rw http.ResponseWriter, r *http.Request, baseURL string) {
	q := r.URL.Query().Get("q")

	switch q {
	case "config":
		var syndicateTargets []SyndicateTarget
		if syndicationManager != nil {
			targets := syndicationManager.GetTargets()
			syndicateTargets = make([]SyndicateTarget, len(targets))
			for i, t := range targets {
				syndicateTargets[i] = SyndicateTarget{
					UID:  t.UID,
					Name: t.Name,
				}
			}
		}

		config := ConfigResponse{
			MediaEndpoint: baseURL + "/api/micropub/media",
			PostTypes: []PostTypeConfig{
				{Type: "note", Name: "Note"},
				{Type: "article", Name: "Article"},
				{Type: "like", Name: "Like"},
				{Type: "bookmark", Name: "Bookmark"},
				{Type: "repost", Name: "Repost"},
			},
			SyndicateTo: syndicateTargets,
		}
		json.NewEncoder(rw).Encode(config)

	case "syndicate-to":
		var syndicateTargets []SyndicateTarget
		if syndicationManager != nil {
			targets := syndicationManager.GetTargets()
			syndicateTargets = make([]SyndicateTarget, len(targets))
			for i, t := range targets {
				syndicateTargets[i] = SyndicateTarget{
					UID:  t.UID,
					Name: t.Name,
				}
			}
		}
		json.NewEncoder(rw).Encode(map[string][]SyndicateTarget{
			"syndicate-to": syndicateTargets,
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
		"content":           "content",
		"name":              "name",
		"summary":           "summary",
		"category":          "category",
		"category[]":        "category",
		"like-of":           "like-of",
		"bookmark-of":       "bookmark-of",
		"repost-of":         "repost-of",
		"in-reply-to":       "in-reply-to",
		"mp-slug":           "mp-slug",
		"photo":             "photo",
		"photo[]":           "photo",
		"mp-syndicate-to":   "mp-syndicate-to",
		"mp-syndicate-to[]": "mp-syndicate-to",
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
// Returns the post URL and the file path to the created file
func createPost(req MicropubRequest, postType, contentDir, baseURL string) (string, string, error) {
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
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate frontmatter
	frontmatter := generateFrontmatter(req, postType, now)

	// Get content
	content := extractContent(req)

	// Write file
	filePath := filepath.Join(postDir, "index.md")
	fileContent := fmt.Sprintf("---\n%s---\n\n%s\n", frontmatter, content)

	if err := os.WriteFile(filePath, []byte(fileContent), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write file: %w", err)
	}

	// Return URL and file path
	return fmt.Sprintf("%s/%s/%s/", baseURL, typeDir, slug), filePath, nil
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
