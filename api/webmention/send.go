package webmention

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"willnorris.com/go/webmention"
)

// SendResult represents the result of sending a webmention
type SendResult struct {
	Target   string `json:"target"`
	Endpoint string `json:"endpoint,omitempty"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// SendResponse represents the response from sending webmentions
type SendResponse struct {
	Source  string       `json:"source"`
	Results []SendResult `json:"results"`
}

// linkPattern matches URLs in content
var linkPattern = regexp.MustCompile(`https?://[^\s<>"'\)\]]+`)

// hrefPattern matches href attributes in HTML
var hrefPattern = regexp.MustCompile(`href=["']([^"']+)["']`)

// ExtractLinks extracts all external URLs from content (HTML or plain text)
func ExtractLinks(content, sourceHost string) []string {
	seen := make(map[string]bool)
	var links []string

	// Extract from href attributes
	hrefMatches := hrefPattern.FindAllStringSubmatch(content, -1)
	for _, match := range hrefMatches {
		if len(match) > 1 {
			link := match[1]
			if isExternalLink(link, sourceHost) && !seen[link] {
				seen[link] = true
				links = append(links, link)
			}
		}
	}

	// Extract plain URLs
	urlMatches := linkPattern.FindAllString(content, -1)
	for _, link := range urlMatches {
		// Clean up trailing punctuation
		link = strings.TrimRight(link, ".,;:!?)")
		if isExternalLink(link, sourceHost) && !seen[link] {
			seen[link] = true
			links = append(links, link)
		}
	}

	return links
}

// isExternalLink checks if a URL is external (not same host)
func isExternalLink(link, sourceHost string) bool {
	parsed, err := url.Parse(link)
	if err != nil {
		return false
	}

	// Must be http or https
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	// Must not be same host
	if strings.EqualFold(parsed.Host, sourceHost) {
		return false
	}

	// Skip common non-IndieWeb sites that don't accept webmentions
	skipHosts := []string{
		"github.com",
		"twitter.com",
		"x.com",
		"facebook.com",
		"instagram.com",
		"youtube.com",
		"youtu.be",
		"linkedin.com",
		"amazon.com",
		"google.com",
		"wikipedia.org",
	}
	for _, skip := range skipHosts {
		if strings.Contains(strings.ToLower(parsed.Host), skip) {
			return false
		}
	}

	return true
}

// SendWebmentions sends webmentions for all links in the given source URL
func SendWebmentions(sourceURL string) (*SendResponse, error) {
	// Parse source URL to get host
	sourceParsed, err := url.Parse(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid source URL: %w", err)
	}

	// Fetch the source page
	resp, err := http.Get(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch source: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read source: %w", err)
	}

	// Extract links
	links := ExtractLinks(string(body), sourceParsed.Host)

	response := &SendResponse{
		Source:  sourceURL,
		Results: make([]SendResult, 0, len(links)),
	}

	// Create webmention client
	client := webmention.New(nil)

	// Send webmentions concurrently (with limit)
	var wg sync.WaitGroup
	resultChan := make(chan SendResult, len(links))
	semaphore := make(chan struct{}, 5) // Max 5 concurrent requests

	for _, target := range links {
		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			result := SendResult{Target: targetURL}

			// Discover webmention endpoint
			endpoint, err := client.DiscoverEndpoint(targetURL)
			if err != nil {
				result.Error = fmt.Sprintf("no endpoint: %v", err)
				resultChan <- result
				return
			}

			if endpoint == "" {
				result.Error = "no webmention endpoint found"
				resultChan <- result
				return
			}

			result.Endpoint = endpoint

			// Send webmention
			_, err = client.SendWebmention(endpoint, sourceURL, targetURL)
			if err != nil {
				result.Error = fmt.Sprintf("send failed: %v", err)
				resultChan <- result
				return
			}

			result.Success = true
			log.Printf("Webmention sent: %s -> %s", sourceURL, targetURL)
			resultChan <- result
		}(target)
	}

	// Wait and collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		response.Results = append(response.Results, result)
	}

	return response, nil
}

// SendWebmentionHandler handles requests to send webmentions for a URL
func SendWebmentionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get source URL from request
	var sourceURL string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var req struct {
			Source string `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "invalid_json"}`, http.StatusBadRequest)
			return
		}
		sourceURL = req.Source
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, `{"error": "invalid_form"}`, http.StatusBadRequest)
			return
		}
		sourceURL = r.FormValue("source")
	}

	if sourceURL == "" {
		http.Error(w, `{"error": "source_required"}`, http.StatusBadRequest)
		return
	}

	// Send webmentions
	response, err := SendWebmentions(sourceURL)
	if err != nil {
		log.Printf("Webmention send error: %v", err)
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
