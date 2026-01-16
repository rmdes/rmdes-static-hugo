package webmention

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Webmention represents a single webmention from webmention.io
type Webmention struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	AuthorName  string `json:"author_name"`
	AuthorPhoto string `json:"author_photo"`
	AuthorURL   string `json:"author_url"`
	Content     string `json:"content"`
	Published   string `json:"published"`
	Type        string `json:"type"` // like, reply, repost, mention, bookmark
	WmID        int    `json:"wm_id"`
}

// WebmentionCache holds cached webmentions with timestamp
type WebmentionCache struct {
	Mentions  []Webmention `json:"mentions"`
	FetchedAt time.Time    `json:"fetched_at"`
}

// GroupedWebmentions organizes webmentions by type
type GroupedWebmentions struct {
	Likes    []Webmention `json:"likes"`
	Reposts  []Webmention `json:"reposts"`
	Replies  []Webmention `json:"replies"`
	Mentions []Webmention `json:"mentions"`
}

// webmention.io JF2 response structures
type jf2Response struct {
	Type     string      `json:"type"`
	Children []jf2Entry  `json:"children"`
}

type jf2Entry struct {
	Type       string    `json:"type"`
	Author     jf2Author `json:"author"`
	URL        string    `json:"url"`
	Published  string    `json:"published"`
	WmReceived string    `json:"wm-received"`
	WmID       int       `json:"wm-id"`
	WmSource   string    `json:"wm-source"`
	WmTarget   string    `json:"wm-target"`
	WmProperty string    `json:"wm-property"` // like-of, repost-of, in-reply-to, mention-of
	Content    *jf2Content `json:"content,omitempty"`
	LikeOf     string    `json:"like-of,omitempty"`
	RepostOf   string    `json:"repost-of,omitempty"`
	InReplyTo  string    `json:"in-reply-to,omitempty"`
	BookmarkOf string    `json:"bookmark-of,omitempty"`
}

type jf2Author struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Photo string `json:"photo"`
	URL   string `json:"url"`
}

type jf2Content struct {
	HTML string `json:"html,omitempty"`
	Text string `json:"text,omitempty"`
}

var (
	cacheMu sync.RWMutex
	cache   *WebmentionCache
)

// WebmentionHandler handles requests for webmentions
func WebmentionHandler(rw http.ResponseWriter, r *http.Request, cacheDir, token string, ttl int) {
	rw.Header().Set("Content-Type", "application/json")

	target := r.URL.Query().Get("target")

	// Load or refresh cache
	mentions, err := getCachedMentions(cacheDir, token, ttl)
	if err != nil {
		json.NewEncoder(rw).Encode(GroupedWebmentions{})
		return
	}

	// Filter by target if specified
	if target != "" {
		mentions = filterByTarget(mentions, target)
	}

	// Group by type
	result := groupByType(mentions)

	json.NewEncoder(rw).Encode(result)
}

func getCachedMentions(cacheDir, token string, ttl int) ([]Webmention, error) {
	cacheFile := filepath.Join(cacheDir, "webmentions_cache.json")

	cacheMu.RLock()
	// Check in-memory cache first
	if cache != nil && time.Since(cache.FetchedAt) < time.Duration(ttl)*time.Second {
		result := cache.Mentions
		cacheMu.RUnlock()
		return result, nil
	}
	cacheMu.RUnlock()

	// Check file cache
	if info, err := os.Stat(cacheFile); err == nil {
		if time.Since(info.ModTime()) < time.Duration(ttl)*time.Second {
			data, err := os.ReadFile(cacheFile)
			if err == nil {
				var cached WebmentionCache
				if json.Unmarshal(data, &cached) == nil {
					cacheMu.Lock()
					cache = &cached
					cacheMu.Unlock()
					return cached.Mentions, nil
				}
			}
		}
	}

	// Token required for fetching from webmention.io
	if token == "" {
		return nil, fmt.Errorf("no webmention.io token configured")
	}

	// Fetch from webmention.io
	mentions, err := fetchFromWebmentionIO(token)
	if err != nil {
		return nil, err
	}

	// Update cache
	newCache := WebmentionCache{
		Mentions:  mentions,
		FetchedAt: time.Now(),
	}

	cacheMu.Lock()
	cache = &newCache
	cacheMu.Unlock()

	// Write file cache
	data, _ := json.MarshalIndent(newCache, "", "  ")
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(cacheFile, data, 0644)

	return mentions, nil
}

func fetchFromWebmentionIO(token string) ([]Webmention, error) {
	apiURL := fmt.Sprintf("https://webmention.io/api/mentions.jf2?token=%s&per-page=100", url.QueryEscape(token))

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch webmentions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("webmention.io returned %d: %s", resp.StatusCode, string(body))
	}

	var jf2Resp jf2Response
	if err := json.NewDecoder(resp.Body).Decode(&jf2Resp); err != nil {
		return nil, fmt.Errorf("failed to parse webmention.io response: %w", err)
	}

	// Convert JF2 entries to our Webmention format
	mentions := make([]Webmention, 0, len(jf2Resp.Children))
	for _, entry := range jf2Resp.Children {
		wm := Webmention{
			Source:      entry.WmSource,
			Target:      entry.WmTarget,
			AuthorName:  entry.Author.Name,
			AuthorPhoto: entry.Author.Photo,
			AuthorURL:   entry.Author.URL,
			Published:   entry.Published,
			WmID:        entry.WmID,
		}

		// Extract content if present
		if entry.Content != nil {
			if entry.Content.Text != "" {
				wm.Content = entry.Content.Text
			} else if entry.Content.HTML != "" {
				wm.Content = entry.Content.HTML
			}
		}

		// Determine type from wm-property
		switch entry.WmProperty {
		case "like-of":
			wm.Type = "like"
		case "repost-of":
			wm.Type = "repost"
		case "in-reply-to":
			wm.Type = "reply"
		case "bookmark-of":
			wm.Type = "bookmark"
		default:
			wm.Type = "mention"
		}

		mentions = append(mentions, wm)
	}

	return mentions, nil
}

func filterByTarget(mentions []Webmention, target string) []Webmention {
	// Normalize target URL for comparison
	target = normalizeURL(target)

	filtered := make([]Webmention, 0)
	for _, m := range mentions {
		if normalizeURL(m.Target) == target {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func normalizeURL(u string) string {
	// Remove trailing slash and normalize for comparison
	u = strings.TrimSuffix(u, "/")
	// Remove protocol for comparison
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	return strings.ToLower(u)
}

func groupByType(mentions []Webmention) GroupedWebmentions {
	result := GroupedWebmentions{
		Likes:    make([]Webmention, 0),
		Reposts:  make([]Webmention, 0),
		Replies:  make([]Webmention, 0),
		Mentions: make([]Webmention, 0),
	}

	for _, m := range mentions {
		switch m.Type {
		case "like":
			result.Likes = append(result.Likes, m)
		case "repost":
			result.Reposts = append(result.Reposts, m)
		case "reply":
			result.Replies = append(result.Replies, m)
		default:
			result.Mentions = append(result.Mentions, m)
		}
	}

	return result
}
