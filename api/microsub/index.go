package microsub

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pojntfx/felicitas.pojtinger.com/api/indieauth"
)

// Channel represents a Microsub channel (category/folder)
type Channel struct {
	UID    string `json:"uid"`
	Name   string `json:"name"`
	Unread *int   `json:"unread,omitempty"` // nil = unknown, 0+ = count
}

// Feed represents a subscribed feed
type Feed struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	Name string `json:"name,omitempty"`
}

// Item represents a feed item in JF2 format
type Item struct {
	Type      string   `json:"type"`
	Name      string   `json:"name,omitempty"`
	Content   *Content `json:"content,omitempty"`
	Published string   `json:"published,omitempty"`
	URL       string   `json:"url,omitempty"`
	Author    *Author  `json:"author,omitempty"`
	Photo     []string `json:"photo,omitempty"`
	UID       string   `json:"_id,omitempty"`
	IsRead    bool     `json:"_is_read,omitempty"`
}

// Content represents item content
type Content struct {
	Text string `json:"text,omitempty"`
	HTML string `json:"html,omitempty"`
}

// Author represents an item author
type Author struct {
	Type  string `json:"type"`
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Photo string `json:"photo,omitempty"`
}

// MutedUser represents a muted user
type MutedUser struct {
	URL   string `json:"url"`
	Name  string `json:"name,omitempty"`
	Photo string `json:"photo,omitempty"`
}

// MicrosubData stores all Microsub state
type MicrosubData struct {
	Channels      []Channel            `json:"channels"`
	Subscriptions map[string][]Feed    `json:"subscriptions"`  // channel UID -> feeds
	ReadItems     map[string]bool      `json:"read_items"`     // item UID -> read status
	MutedUsers    map[string][]MutedUser `json:"muted_users"`  // channel UID -> muted users
	BlockedUsers  map[string][]MutedUser `json:"blocked_users"` // channel UID -> blocked users
}

var (
	dataLock sync.RWMutex
	data     *MicrosubData
	dataPath string
)

// loadData loads Microsub data from file
func loadData(cacheDir string) (*MicrosubData, error) {
	dataLock.Lock()
	defer dataLock.Unlock()

	if data != nil {
		return data, nil
	}

	dataPath = filepath.Join(cacheDir, "microsub_data.json")

	// Try to load existing data
	file, err := os.ReadFile(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default data with a "default" channel
			data = &MicrosubData{
				Channels: []Channel{
					{UID: "default", Name: "Home"},
					{UID: "notifications", Name: "Notifications"},
				},
				Subscriptions: make(map[string][]Feed),
				ReadItems:     make(map[string]bool),
				MutedUsers:    make(map[string][]MutedUser),
				BlockedUsers:  make(map[string][]MutedUser),
			}
			return data, saveDataLocked()
		}
		return nil, err
	}

	data = &MicrosubData{}
	if err := json.Unmarshal(file, data); err != nil {
		return nil, err
	}

	if data.Subscriptions == nil {
		data.Subscriptions = make(map[string][]Feed)
	}
	if data.ReadItems == nil {
		data.ReadItems = make(map[string]bool)
	}
	if data.MutedUsers == nil {
		data.MutedUsers = make(map[string][]MutedUser)
	}
	if data.BlockedUsers == nil {
		data.BlockedUsers = make(map[string][]MutedUser)
	}

	return data, nil
}

// saveDataLocked saves data (must hold dataLock)
func saveDataLocked() error {
	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dataPath, file, 0644)
}

// saveData saves data with locking
func saveData() error {
	dataLock.Lock()
	defer dataLock.Unlock()
	return saveDataLocked()
}

// MicrosubHandler handles Microsub API requests
func MicrosubHandler(rw http.ResponseWriter, r *http.Request, cacheDir string) {
	// Verify IndieAuth token for all requests
	token := indieauth.ExtractToken(r)
	tokenResp, err := indieauth.VerifyToken(token, indieauth.GetTokenEndpoint(), indieauth.GetExpectedMe())
	if err != nil {
		log.Printf("Microsub auth error: %v", err)
		http.Error(rw, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Check for read scope (minimum required)
	if !indieauth.HasScope(tokenResp, "read") && !indieauth.HasScope(tokenResp, "follow") && !indieauth.HasScope(tokenResp, "channels") {
		http.Error(rw, `{"error": "insufficient_scope"}`, http.StatusForbidden)
		return
	}

	// Load data
	if _, err := loadData(cacheDir); err != nil {
		log.Printf("Failed to load microsub data: %v", err)
		http.Error(rw, `{"error": "server_error"}`, http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")

	// Handle based on method
	if r.Method == http.MethodGet {
		handleGet(rw, r, tokenResp)
	} else if r.Method == http.MethodPost {
		handlePost(rw, r, tokenResp)
	} else {
		http.Error(rw, `{"error": "method_not_allowed"}`, http.StatusMethodNotAllowed)
	}
}

func handleGet(rw http.ResponseWriter, r *http.Request, tokenResp *indieauth.TokenResponse) {
	action := r.URL.Query().Get("action")

	switch action {
	case "channels":
		handleGetChannels(rw, r)
	case "timeline":
		handleGetTimeline(rw, r)
	case "follow":
		handleGetFollowing(rw, r)
	case "mute":
		handleGetMuted(rw, r)
	case "block":
		handleGetBlocked(rw, r)
	default:
		// Default: return channels
		handleGetChannels(rw, r)
	}
}

func handlePost(rw http.ResponseWriter, r *http.Request, tokenResp *indieauth.TokenResponse) {
	if err := r.ParseForm(); err != nil {
		http.Error(rw, `{"error": "invalid_request"}`, http.StatusBadRequest)
		return
	}

	action := r.FormValue("action")

	switch action {
	case "channels":
		if !indieauth.HasScope(tokenResp, "channels") {
			http.Error(rw, `{"error": "insufficient_scope"}`, http.StatusForbidden)
			return
		}
		handlePostChannels(rw, r)
	case "follow":
		if !indieauth.HasScope(tokenResp, "follow") {
			http.Error(rw, `{"error": "insufficient_scope"}`, http.StatusForbidden)
			return
		}
		handleFollow(rw, r)
	case "unfollow":
		if !indieauth.HasScope(tokenResp, "follow") {
			http.Error(rw, `{"error": "insufficient_scope"}`, http.StatusForbidden)
			return
		}
		handleUnfollow(rw, r)
	case "search":
		handleSearch(rw, r)
	case "preview":
		handlePreview(rw, r)
	case "timeline":
		handleMarkRead(rw, r)
	case "subscribe":
		// Subscribe helper: discovers feed and subscribes in one step
		if !indieauth.HasScope(tokenResp, "follow") {
			http.Error(rw, `{"error": "insufficient_scope"}`, http.StatusForbidden)
			return
		}
		handleSubscribe(rw, r)
	case "mute":
		if !indieauth.HasScope(tokenResp, "mute") && !indieauth.HasScope(tokenResp, "channels") {
			http.Error(rw, `{"error": "insufficient_scope"}`, http.StatusForbidden)
			return
		}
		handleMute(rw, r)
	case "unmute":
		if !indieauth.HasScope(tokenResp, "mute") && !indieauth.HasScope(tokenResp, "channels") {
			http.Error(rw, `{"error": "insufficient_scope"}`, http.StatusForbidden)
			return
		}
		handleUnmute(rw, r)
	case "block":
		if !indieauth.HasScope(tokenResp, "block") && !indieauth.HasScope(tokenResp, "channels") {
			http.Error(rw, `{"error": "insufficient_scope"}`, http.StatusForbidden)
			return
		}
		handleBlock(rw, r)
	case "unblock":
		if !indieauth.HasScope(tokenResp, "block") && !indieauth.HasScope(tokenResp, "channels") {
			http.Error(rw, `{"error": "insufficient_scope"}`, http.StatusForbidden)
			return
		}
		handleUnblock(rw, r)
	default:
		http.Error(rw, `{"error": "invalid_action"}`, http.StatusBadRequest)
	}
}

func handleGetChannels(rw http.ResponseWriter, r *http.Request) {
	dataLock.RLock()
	channels := make([]Channel, len(data.Channels))
	copy(channels, data.Channels)
	subscriptions := data.Subscriptions
	readItems := data.ReadItems
	dataLock.RUnlock()

	// Calculate unread counts for each channel
	for i := range channels {
		feeds := subscriptions[channels[i].UID]
		if len(feeds) == 0 {
			zero := 0
			channels[i].Unread = &zero
			continue
		}

		unreadCount := 0
		for _, feed := range feeds {
			items, err := fetchFeed(feed.URL)
			if err != nil {
				continue
			}
			for _, item := range items {
				uid := item.UID
				if uid == "" {
					uid = generateItemID(&item)
				}
				if !readItems[uid] {
					unreadCount++
				}
			}
		}
		channels[i].Unread = &unreadCount
	}

	response := map[string]interface{}{
		"channels": channels,
	}
	json.NewEncoder(rw).Encode(response)
}

func handlePostChannels(rw http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	uid := r.FormValue("channel")
	method := r.FormValue("method")

	dataLock.Lock()
	defer dataLock.Unlock()

	switch method {
	case "delete":
		// Delete channel
		if uid == "default" || uid == "notifications" {
			http.Error(rw, `{"error": "cannot_delete_default_channel"}`, http.StatusBadRequest)
			return
		}
		newChannels := []Channel{}
		for _, ch := range data.Channels {
			if ch.UID != uid {
				newChannels = append(newChannels, ch)
			}
		}
		data.Channels = newChannels
		delete(data.Subscriptions, uid)
		saveDataLocked()
		json.NewEncoder(rw).Encode(map[string]string{"result": "deleted"})

	case "order":
		// Reorder channels based on provided order
		channelOrder := r.Form["channels[]"]
		if len(channelOrder) == 0 {
			channelOrder = r.Form["channels"]
		}
		if len(channelOrder) == 0 {
			http.Error(rw, `{"error": "channels_required"}`, http.StatusBadRequest)
			return
		}

		// Build new channel list in specified order
		channelMap := make(map[string]Channel)
		for _, ch := range data.Channels {
			channelMap[ch.UID] = ch
		}

		newChannels := make([]Channel, 0, len(channelOrder))
		for _, uid := range channelOrder {
			if ch, exists := channelMap[uid]; exists {
				newChannels = append(newChannels, ch)
				delete(channelMap, uid)
			}
		}
		// Append any channels not in the order list
		for _, ch := range channelMap {
			newChannels = append(newChannels, ch)
		}

		data.Channels = newChannels
		saveDataLocked()
		json.NewEncoder(rw).Encode(map[string]string{"result": "ok"})

	default:
		if uid != "" {
			// Update existing channel
			for i, ch := range data.Channels {
				if ch.UID == uid {
					data.Channels[i].Name = name
					saveDataLocked()
					json.NewEncoder(rw).Encode(data.Channels[i])
					return
				}
			}
			http.Error(rw, `{"error": "channel_not_found"}`, http.StatusNotFound)
		} else {
			// Create new channel
			newUID := fmt.Sprintf("channel-%d", time.Now().UnixNano())
			newChannel := Channel{UID: newUID, Name: name}
			data.Channels = append(data.Channels, newChannel)
			saveDataLocked()
			json.NewEncoder(rw).Encode(newChannel)
		}
	}
}

func handleGetFollowing(rw http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "default"
	}

	dataLock.RLock()
	defer dataLock.RUnlock()

	feeds := data.Subscriptions[channel]
	if feeds == nil {
		feeds = []Feed{}
	}

	response := map[string]interface{}{
		"items": feeds,
	}
	json.NewEncoder(rw).Encode(response)
}

func handleFollow(rw http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	channel := r.FormValue("channel")
	if channel == "" {
		channel = "default"
	}

	if url == "" {
		http.Error(rw, `{"error": "url_required"}`, http.StatusBadRequest)
		return
	}

	dataLock.Lock()
	defer dataLock.Unlock()

	// Check if already subscribed
	for _, feed := range data.Subscriptions[channel] {
		if feed.URL == url {
			json.NewEncoder(rw).Encode(feed)
			return
		}
	}

	// Add new subscription
	newFeed := Feed{
		Type: "feed",
		URL:  url,
	}
	data.Subscriptions[channel] = append(data.Subscriptions[channel], newFeed)
	saveDataLocked()

	json.NewEncoder(rw).Encode(newFeed)
}

func handleUnfollow(rw http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	channel := r.FormValue("channel")
	if channel == "" {
		channel = "default"
	}

	if url == "" {
		http.Error(rw, `{"error": "url_required"}`, http.StatusBadRequest)
		return
	}

	dataLock.Lock()
	defer dataLock.Unlock()

	newFeeds := []Feed{}
	for _, feed := range data.Subscriptions[channel] {
		if feed.URL != url {
			newFeeds = append(newFeeds, feed)
		}
	}
	data.Subscriptions[channel] = newFeeds
	saveDataLocked()

	json.NewEncoder(rw).Encode(map[string]string{"result": "unfollowed"})
}

func handleSearch(rw http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")
	if query == "" {
		http.Error(rw, `{"error": "query_required"}`, http.StatusBadRequest)
		return
	}

	// Discover feeds from the URL
	feeds := discoverFeeds(query)

	response := map[string]interface{}{
		"results": feeds,
	}
	json.NewEncoder(rw).Encode(response)
}

// handleSubscribe discovers a feed from a URL and subscribes to it in one action
func handleSubscribe(rw http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	channel := r.FormValue("channel")
	if channel == "" {
		channel = "default"
	}

	if url == "" {
		http.Error(rw, `{"error": "url_required"}`, http.StatusBadRequest)
		return
	}

	// Discover feeds from the URL
	feeds := discoverFeeds(url)
	if len(feeds) == 0 {
		http.Error(rw, `{"error": "no_feed_found"}`, http.StatusNotFound)
		return
	}

	// Use the first discovered feed
	discoveredFeed := feeds[0]

	dataLock.Lock()
	defer dataLock.Unlock()

	// Check if channel exists
	channelExists := false
	for _, ch := range data.Channels {
		if ch.UID == channel {
			channelExists = true
			break
		}
	}
	if !channelExists {
		http.Error(rw, `{"error": "channel_not_found"}`, http.StatusBadRequest)
		return
	}

	// Check if already subscribed
	for _, feed := range data.Subscriptions[channel] {
		if feed.URL == discoveredFeed.URL {
			// Already subscribed, return existing
			json.NewEncoder(rw).Encode(map[string]interface{}{
				"type":       "feed",
				"url":        feed.URL,
				"name":       feed.Name,
				"channel":    channel,
				"subscribed": true,
				"already":    true,
			})
			return
		}
	}

	// Add new subscription
	newFeed := Feed{
		Type: "feed",
		URL:  discoveredFeed.URL,
		Name: discoveredFeed.Name,
	}

	// If no name from discovery, try to get it from preview
	if newFeed.Name == "" {
		items, err := fetchFeed(newFeed.URL)
		if err == nil && len(items) > 0 && items[0].Author != nil {
			newFeed.Name = items[0].Author.Name
		}
	}

	data.Subscriptions[channel] = append(data.Subscriptions[channel], newFeed)
	if err := saveDataLocked(); err != nil {
		http.Error(rw, `{"error": "save_failed"}`, http.StatusInternalServerError)
		return
	}

	// Return success with subscription details
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"type":              "feed",
		"url":               newFeed.URL,
		"name":              newFeed.Name,
		"channel":           channel,
		"subscribed":        true,
		"discovered_from":   url,
		"all_feeds_found":   len(feeds),
	})
}

// discoverFeeds attempts to find RSS/Atom/JSON feeds from a URL
func discoverFeeds(url string) []Feed {
	var feeds []Feed

	// First, try to fetch the URL and look for feed links
	resp, err := http.Get(url)
	if err != nil {
		// Fall back to returning the URL as-is
		return []Feed{{Type: "feed", URL: url}}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []Feed{{Type: "feed", URL: url}}
	}

	content := string(body)
	contentType := resp.Header.Get("Content-Type")

	// Check if this is already a feed
	if strings.Contains(contentType, "xml") || strings.Contains(contentType, "rss") ||
		strings.Contains(contentType, "atom") || strings.Contains(contentType, "json") {
		// This URL is itself a feed
		feedName := extractFeedTitle(content)
		return []Feed{{Type: "feed", URL: url, Name: feedName}}
	}

	// If JSON, check for JSON Feed format
	if strings.HasPrefix(strings.TrimSpace(content), "{") {
		var jsonFeed struct {
			Version string `json:"version"`
			Title   string `json:"title"`
		}
		if json.Unmarshal(body, &jsonFeed) == nil && strings.Contains(jsonFeed.Version, "jsonfeed") {
			return []Feed{{Type: "feed", URL: url, Name: jsonFeed.Title}}
		}
	}

	// Look for <link> tags pointing to feeds
	feeds = append(feeds, extractFeedLinks(content, url)...)

	// If no feeds found from link tags, try common feed paths
	if len(feeds) == 0 {
		feeds = tryCommonFeedPaths(url)
	}

	// If still no feeds found, return the original URL
	if len(feeds) == 0 {
		return []Feed{{Type: "feed", URL: url}}
	}

	return feeds
}

// extractFeedLinks extracts feed URLs from HTML <link> tags
func extractFeedLinks(html, baseURL string) []Feed {
	var feeds []Feed

	// Parse base URL for resolving relative links
	base, err := parseURL(baseURL)
	if err != nil {
		return feeds
	}

	// Look for link tags with feed types
	// <link rel="alternate" type="application/rss+xml" href="..." title="...">
	linkPattern := `<link[^>]+>`
	for _, match := range findAllStrings(html, linkPattern) {
		rel := extractAttr(match, "rel")
		linkType := extractAttr(match, "type")
		href := extractAttr(match, "href")
		title := extractAttr(match, "title")

		// Check if this is a feed link
		if !strings.Contains(rel, "alternate") {
			continue
		}

		isFeed := strings.Contains(linkType, "rss") ||
			strings.Contains(linkType, "atom") ||
			strings.Contains(linkType, "feed") ||
			strings.Contains(linkType, "json")

		if !isFeed || href == "" {
			continue
		}

		// Resolve relative URL
		feedURL := resolveURL(base, href)
		if feedURL == "" {
			continue
		}

		feeds = append(feeds, Feed{
			Type: "feed",
			URL:  feedURL,
			Name: title,
		})
	}

	return feeds
}

// tryCommonFeedPaths tries common feed URL patterns
func tryCommonFeedPaths(baseURL string) []Feed {
	var feeds []Feed

	base, err := parseURL(baseURL)
	if err != nil {
		return feeds
	}

	// Common feed paths to try
	commonPaths := []string{
		"/feed",
		"/feed/",
		"/feed.xml",
		"/rss",
		"/rss.xml",
		"/atom.xml",
		"/index.xml",
		"/feed/atom",
		"/feed/rss",
		"/feeds/posts/default",
		"/.rss",
		"/blog/feed",
		"/blog/rss",
		"/blog/index.xml",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	for _, path := range commonPaths {
		testURL := base.Scheme + "://" + base.Host + path
		resp, err := client.Head(testURL)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "xml") ||
				strings.Contains(contentType, "rss") ||
				strings.Contains(contentType, "atom") ||
				strings.Contains(contentType, "json") {
				feeds = append(feeds, Feed{
					Type: "feed",
					URL:  testURL,
				})
				// Return first found feed
				break
			}
		}
	}

	return feeds
}

// extractFeedTitle extracts the title from RSS/Atom/JSON feed content
func extractFeedTitle(content string) string {
	// Try JSON Feed first
	if strings.HasPrefix(strings.TrimSpace(content), "{") {
		var jsonFeed struct {
			Title string `json:"title"`
		}
		if json.Unmarshal([]byte(content), &jsonFeed) == nil && jsonFeed.Title != "" {
			return jsonFeed.Title
		}
	}

	// Try to extract <title> from XML
	title := extractXMLElement(content, "title")
	if title != "" {
		return title
	}

	// Try channel > title for RSS
	title = extractXMLElement(content, "channel>title")
	return title
}

// Helper to parse URL
func parseURL(urlStr string) (*struct{ Scheme, Host, Path string }, error) {
	// Simple URL parsing
	if !strings.HasPrefix(urlStr, "http") {
		urlStr = "https://" + urlStr
	}

	parts := strings.SplitN(strings.TrimPrefix(strings.TrimPrefix(urlStr, "https://"), "http://"), "/", 2)
	scheme := "https"
	if strings.HasPrefix(urlStr, "http://") {
		scheme = "http"
	}

	result := &struct{ Scheme, Host, Path string }{
		Scheme: scheme,
		Host:   parts[0],
		Path:   "/",
	}
	if len(parts) > 1 {
		result.Path = "/" + parts[1]
	}
	return result, nil
}

// resolveURL resolves a relative URL against a base
func resolveURL(base *struct{ Scheme, Host, Path string }, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return base.Scheme + ":" + href
	}
	if strings.HasPrefix(href, "/") {
		return base.Scheme + "://" + base.Host + href
	}
	// Relative path
	return base.Scheme + "://" + base.Host + "/" + href
}

// findAllStrings finds all matches of a simple pattern in content
func findAllStrings(content, pattern string) []string {
	var results []string

	// Simple pattern matching for <link...>
	if pattern == `<link[^>]+>` {
		lower := strings.ToLower(content)
		for {
			start := strings.Index(lower, "<link")
			if start == -1 {
				break
			}
			end := strings.Index(lower[start:], ">")
			if end == -1 {
				break
			}
			results = append(results, content[start:start+end+1])
			content = content[start+end+1:]
			lower = lower[start+end+1:]
		}
	}

	return results
}

// extractAttr extracts an attribute value from an HTML tag
func extractAttr(tag, attr string) string {
	lower := strings.ToLower(tag)
	attrLower := strings.ToLower(attr)

	// Try double quotes
	start := strings.Index(lower, attrLower+"=\"")
	if start != -1 {
		valueStart := start + len(attr) + 2
		end := strings.Index(tag[valueStart:], "\"")
		if end != -1 {
			return tag[valueStart : valueStart+end]
		}
	}

	// Try single quotes
	start = strings.Index(lower, attrLower+"='")
	if start != -1 {
		valueStart := start + len(attr) + 2
		end := strings.Index(tag[valueStart:], "'")
		if end != -1 {
			return tag[valueStart : valueStart+end]
		}
	}

	return ""
}

func handlePreview(rw http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if url == "" {
		http.Error(rw, `{"error": "url_required"}`, http.StatusBadRequest)
		return
	}

	// Fetch and parse the feed
	items, err := fetchFeed(url)
	if err != nil {
		log.Printf("Failed to fetch feed %s: %v", url, err)
		http.Error(rw, fmt.Sprintf(`{"error": "fetch_failed", "error_description": "%s"}`, err.Error()), http.StatusBadGateway)
		return
	}

	response := map[string]interface{}{
		"items": items,
	}
	json.NewEncoder(rw).Encode(response)
}

func handleGetTimeline(rw http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "default"
	}

	// Parse paging parameters
	beforeCursor := r.URL.Query().Get("before")
	afterCursor := r.URL.Query().Get("after")
	limitStr := r.URL.Query().Get("limit")
	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	dataLock.RLock()
	feeds := data.Subscriptions[channel]
	readItems := data.ReadItems
	mutedUsers := data.MutedUsers[channel]
	blockedUsers := data.BlockedUsers[channel]
	dataLock.RUnlock()

	// Build set of muted/blocked URLs for filtering
	filteredURLs := make(map[string]bool)
	for _, u := range mutedUsers {
		filteredURLs[u.URL] = true
	}
	for _, u := range blockedUsers {
		filteredURLs[u.URL] = true
	}

	// Fetch all feeds and aggregate items
	var allItems []Item
	for _, feed := range feeds {
		items, err := fetchFeed(feed.URL)
		if err != nil {
			log.Printf("Failed to fetch feed %s: %v", feed.URL, err)
			continue
		}
		allItems = append(allItems, items...)
	}

	// Ensure each item has a unique _id
	for i := range allItems {
		if allItems[i].UID == "" {
			allItems[i].UID = generateItemID(&allItems[i])
		}
	}

	// Filter out muted/blocked authors
	if len(filteredURLs) > 0 {
		var filtered []Item
		for _, item := range allItems {
			if item.Author != nil && filteredURLs[item.Author.URL] {
				continue
			}
			filtered = append(filtered, item)
		}
		allItems = filtered
	}

	// Sort by published date (newest first)
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Published > allItems[j].Published
	})

	// Apply cursor-based paging
	// "before" means items published before (older than) this cursor - for next page
	// "after" means items published after (newer than) this cursor - for previous page
	var startIdx, endIdx int
	if afterCursor != "" {
		// Find items newer than the cursor (going back towards top)
		for i, item := range allItems {
			if item.UID == afterCursor || item.Published == afterCursor {
				startIdx = 0
				endIdx = i
				if endIdx > limit {
					startIdx = endIdx - limit
				}
				break
			}
		}
	} else if beforeCursor != "" {
		// Find items older than the cursor (going forward/down)
		for i, item := range allItems {
			if item.UID == beforeCursor || item.Published == beforeCursor {
				startIdx = i + 1
				endIdx = startIdx + limit
				if endIdx > len(allItems) {
					endIdx = len(allItems)
				}
				break
			}
		}
	} else {
		// No cursor, start from beginning
		startIdx = 0
		endIdx = limit
		if endIdx > len(allItems) {
			endIdx = len(allItems)
		}
	}

	// Slice to the requested page
	var pageItems []Item
	if startIdx < len(allItems) && startIdx < endIdx {
		pageItems = allItems[startIdx:endIdx]
	}

	// Mark read status
	for i := range pageItems {
		if pageItems[i].UID != "" {
			pageItems[i].IsRead = readItems[pageItems[i].UID]
		}
	}

	// Build response with paging info
	response := map[string]interface{}{
		"items": pageItems,
	}

	// Add paging cursors if there are more items
	paging := make(map[string]string)
	if endIdx < len(allItems) && len(pageItems) > 0 {
		// There are more older items - use last item as "before" cursor
		paging["before"] = pageItems[len(pageItems)-1].UID
	}
	if startIdx > 0 && len(pageItems) > 0 {
		// There are newer items - use first item as "after" cursor
		paging["after"] = pageItems[0].UID
	}
	if len(paging) > 0 {
		response["paging"] = paging
	}

	json.NewEncoder(rw).Encode(response)
}

// generateItemID creates a unique ID for an item based on its URL and published date
func generateItemID(item *Item) string {
	source := item.URL
	if source == "" {
		source = item.Published + item.Name
	}
	if item.Author != nil {
		source += item.Author.URL
	}
	hash := sha256.Sum256([]byte(source))
	return base64.RawURLEncoding.EncodeToString(hash[:12])
}

func handleMarkRead(rw http.ResponseWriter, r *http.Request) {
	method := r.FormValue("method")
	if method != "mark_read" && method != "mark_unread" {
		http.Error(rw, `{"error": "invalid_method"}`, http.StatusBadRequest)
		return
	}

	// Get entries to mark
	entries := r.Form["entry[]"]
	if len(entries) == 0 {
		entries = r.Form["entry"]
	}

	if len(entries) == 0 {
		// Check for last_read_entry (mark all before this as read)
		lastRead := r.FormValue("last_read_entry")
		if lastRead == "" {
			http.Error(rw, `{"error": "entry_required"}`, http.StatusBadRequest)
			return
		}
		entries = []string{lastRead}
	}

	dataLock.Lock()
	defer dataLock.Unlock()

	for _, entry := range entries {
		if method == "mark_read" {
			data.ReadItems[entry] = true
		} else {
			delete(data.ReadItems, entry)
		}
	}
	saveDataLocked()

	json.NewEncoder(rw).Encode(map[string]string{"result": "ok"})
}

// fetchFeed fetches and parses a feed URL into JF2 items
func fetchFeed(url string) ([]Item, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	content := string(body)
	contentType := resp.Header.Get("Content-Type")

	// Try to parse as JSON Feed first
	if strings.Contains(contentType, "json") || strings.HasPrefix(content, "{") {
		return parseJSONFeed(body)
	}

	// Try to parse as RSS/Atom
	return parseXMLFeed(body, url)
}

// parseJSONFeed parses a JSON Feed into JF2 items
func parseJSONFeed(data []byte) ([]Item, error) {
	var feed struct {
		Items []struct {
			ID            string `json:"id"`
			URL           string `json:"url"`
			Title         string `json:"title"`
			ContentHTML   string `json:"content_html"`
			ContentText   string `json:"content_text"`
			DatePublished string `json:"date_published"`
			Author        struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"author"`
		} `json:"items"`
	}

	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	var items []Item
	for _, item := range feed.Items {
		jf2Item := Item{
			Type:      "entry",
			Name:      item.Title,
			URL:       item.URL,
			UID:       item.ID,
			Published: item.DatePublished,
		}

		if item.ContentHTML != "" || item.ContentText != "" {
			jf2Item.Content = &Content{
				HTML: item.ContentHTML,
				Text: item.ContentText,
			}
		}

		if item.Author.Name != "" {
			jf2Item.Author = &Author{
				Type: "card",
				Name: item.Author.Name,
				URL:  item.Author.URL,
			}
		}

		items = append(items, jf2Item)
	}

	return items, nil
}

// parseXMLFeed parses RSS/Atom feed into JF2 items (simplified)
func parseXMLFeed(data []byte, feedURL string) ([]Item, error) {
	content := string(data)
	var items []Item

	// Very simple RSS parsing - look for <item> or <entry> elements
	isAtom := strings.Contains(content, "<feed")

	if isAtom {
		// Parse Atom
		entries := splitXMLElements(content, "entry")
		for _, entry := range entries {
			item := Item{
				Type:      "entry",
				Name:      extractXMLElement(entry, "title"),
				URL:       extractXMLAttr(entry, "link", "href"),
				UID:       extractXMLElement(entry, "id"),
				Published: extractXMLElement(entry, "published"),
			}

			if item.URL == "" {
				item.URL = extractXMLElement(entry, "link")
			}
			if item.Published == "" {
				item.Published = extractXMLElement(entry, "updated")
			}

			contentHTML := extractXMLElement(entry, "content")
			if contentHTML == "" {
				contentHTML = extractXMLElement(entry, "summary")
			}
			if contentHTML != "" {
				item.Content = &Content{HTML: contentHTML}
			}

			authorName := extractXMLElement(entry, "author>name")
			if authorName == "" {
				authorName = extractXMLElement(entry, "name")
			}
			if authorName != "" {
				item.Author = &Author{Type: "card", Name: authorName}
			}

			if item.UID == "" {
				item.UID = item.URL
			}

			items = append(items, item)
		}
	} else {
		// Parse RSS
		rssItems := splitXMLElements(content, "item")
		for _, rssItem := range rssItems {
			item := Item{
				Type:      "entry",
				Name:      extractXMLElement(rssItem, "title"),
				URL:       extractXMLElement(rssItem, "link"),
				UID:       extractXMLElement(rssItem, "guid"),
				Published: extractXMLElement(rssItem, "pubDate"),
			}

			if item.Published == "" {
				item.Published = extractXMLElement(rssItem, "dc:date")
			}

			description := extractXMLElement(rssItem, "description")
			contentEncoded := extractXMLElement(rssItem, "content:encoded")
			if contentEncoded != "" {
				item.Content = &Content{HTML: contentEncoded}
			} else if description != "" {
				item.Content = &Content{HTML: description}
			}

			author := extractXMLElement(rssItem, "author")
			if author == "" {
				author = extractXMLElement(rssItem, "dc:creator")
			}
			if author != "" {
				item.Author = &Author{Type: "card", Name: author}
			}

			if item.UID == "" {
				item.UID = item.URL
			}

			items = append(items, item)
		}
	}

	return items, nil
}

// Helper functions for simple XML parsing

func splitXMLElements(content, tag string) []string {
	var elements []string
	startTag := "<" + tag
	endTag := "</" + tag + ">"

	for {
		start := strings.Index(content, startTag)
		if start == -1 {
			break
		}

		// Find the end of the start tag
		tagEnd := strings.Index(content[start:], ">")
		if tagEnd == -1 {
			break
		}

		end := strings.Index(content[start:], endTag)
		if end == -1 {
			break
		}

		elements = append(elements, content[start:start+end+len(endTag)])
		content = content[start+end+len(endTag):]
	}

	return elements
}

func extractXMLElement(content, path string) string {
	parts := strings.Split(path, ">")
	current := content

	for _, tag := range parts {
		startTag := "<" + tag
		start := strings.Index(current, startTag)
		if start == -1 {
			return ""
		}

		// Find end of start tag
		tagEnd := strings.Index(current[start:], ">")
		if tagEnd == -1 {
			return ""
		}
		valueStart := start + tagEnd + 1

		// Find closing tag
		endTag := "</" + tag + ">"
		end := strings.Index(current[valueStart:], endTag)
		if end == -1 {
			// Try self-closing or no explicit close
			return ""
		}

		current = current[valueStart : valueStart+end]
	}

	// Clean CDATA
	current = strings.TrimPrefix(current, "<![CDATA[")
	current = strings.TrimSuffix(current, "]]>")

	return strings.TrimSpace(current)
}

func extractXMLAttr(content, tag, attr string) string {
	startTag := "<" + tag
	start := strings.Index(content, startTag)
	if start == -1 {
		return ""
	}

	// Find end of tag
	tagEnd := strings.Index(content[start:], ">")
	if tagEnd == -1 {
		return ""
	}

	tagContent := content[start : start+tagEnd]

	// Find attribute
	attrStart := strings.Index(tagContent, attr+"=\"")
	if attrStart == -1 {
		attrStart = strings.Index(tagContent, attr+"='")
		if attrStart == -1 {
			return ""
		}
	}

	valueStart := attrStart + len(attr) + 2
	quote := tagContent[attrStart+len(attr)+1]
	valueEnd := strings.Index(tagContent[valueStart:], string(quote))
	if valueEnd == -1 {
		return ""
	}

	return tagContent[valueStart : valueStart+valueEnd]
}

// handleGetMuted returns the list of muted users for a channel
func handleGetMuted(rw http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "global"
	}

	dataLock.RLock()
	defer dataLock.RUnlock()

	muted := data.MutedUsers[channel]
	if muted == nil {
		muted = []MutedUser{}
	}

	// Convert to JF2 card format
	items := make([]map[string]interface{}, len(muted))
	for i, user := range muted {
		items[i] = map[string]interface{}{
			"type": "card",
			"url":  user.URL,
		}
		if user.Name != "" {
			items[i]["name"] = user.Name
		}
		if user.Photo != "" {
			items[i]["photo"] = user.Photo
		}
	}

	json.NewEncoder(rw).Encode(map[string]interface{}{
		"items": items,
	})
}

// handleMute mutes a user in a channel
func handleMute(rw http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	channel := r.FormValue("channel")
	if channel == "" {
		channel = "global"
	}

	if url == "" {
		http.Error(rw, `{"error": "url_required"}`, http.StatusBadRequest)
		return
	}

	dataLock.Lock()
	defer dataLock.Unlock()

	// Check if already muted
	for _, user := range data.MutedUsers[channel] {
		if user.URL == url {
			json.NewEncoder(rw).Encode(map[string]string{"result": "ok"})
			return
		}
	}

	// Add to muted list
	data.MutedUsers[channel] = append(data.MutedUsers[channel], MutedUser{URL: url})
	saveDataLocked()

	json.NewEncoder(rw).Encode(map[string]string{"result": "ok"})
}

// handleUnmute removes a user from the muted list
func handleUnmute(rw http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	channel := r.FormValue("channel")
	if channel == "" {
		channel = "global"
	}

	if url == "" {
		http.Error(rw, `{"error": "url_required"}`, http.StatusBadRequest)
		return
	}

	dataLock.Lock()
	defer dataLock.Unlock()

	newMuted := []MutedUser{}
	for _, user := range data.MutedUsers[channel] {
		if user.URL != url {
			newMuted = append(newMuted, user)
		}
	}
	data.MutedUsers[channel] = newMuted
	saveDataLocked()

	json.NewEncoder(rw).Encode(map[string]string{"result": "ok"})
}

// handleGetBlocked returns the list of blocked users for a channel
func handleGetBlocked(rw http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "global"
	}

	dataLock.RLock()
	defer dataLock.RUnlock()

	blocked := data.BlockedUsers[channel]
	if blocked == nil {
		blocked = []MutedUser{}
	}

	// Convert to JF2 card format
	items := make([]map[string]interface{}, len(blocked))
	for i, user := range blocked {
		items[i] = map[string]interface{}{
			"type": "card",
			"url":  user.URL,
		}
		if user.Name != "" {
			items[i]["name"] = user.Name
		}
		if user.Photo != "" {
			items[i]["photo"] = user.Photo
		}
	}

	json.NewEncoder(rw).Encode(map[string]interface{}{
		"items": items,
	})
}

// handleBlock blocks a user in a channel
func handleBlock(rw http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	channel := r.FormValue("channel")
	if channel == "" {
		channel = "global"
	}

	if url == "" {
		http.Error(rw, `{"error": "url_required"}`, http.StatusBadRequest)
		return
	}

	dataLock.Lock()
	defer dataLock.Unlock()

	// Check if already blocked
	for _, user := range data.BlockedUsers[channel] {
		if user.URL == url {
			json.NewEncoder(rw).Encode(map[string]string{"result": "ok"})
			return
		}
	}

	// Add to blocked list
	data.BlockedUsers[channel] = append(data.BlockedUsers[channel], MutedUser{URL: url})
	saveDataLocked()

	json.NewEncoder(rw).Encode(map[string]string{"result": "ok"})
}

// handleUnblock removes a user from the blocked list
func handleUnblock(rw http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	channel := r.FormValue("channel")
	if channel == "" {
		channel = "global"
	}

	if url == "" {
		http.Error(rw, `{"error": "url_required"}`, http.StatusBadRequest)
		return
	}

	dataLock.Lock()
	defer dataLock.Unlock()

	newBlocked := []MutedUser{}
	for _, user := range data.BlockedUsers[channel] {
		if user.URL != url {
			newBlocked = append(newBlocked, user)
		}
	}
	data.BlockedUsers[channel] = newBlocked
	saveDataLocked()

	json.NewEncoder(rw).Encode(map[string]string{"result": "ok"})
}
