package syndication

import (
	"context"
	"log"
	"strings"
	"sync"
)

// SyndicationTarget represents an available syndication destination
type SyndicationTarget struct {
	UID      string `json:"uid"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

// Manager orchestrates syndication to multiple platforms
type Manager struct {
	syndicators map[string]Syndicator
	targets     []SyndicationTarget
	mu          sync.RWMutex
}

// NewManager creates a new syndication manager
func NewManager() *Manager {
	return &Manager{
		syndicators: make(map[string]Syndicator),
		targets:     make([]SyndicationTarget, 0),
	}
}

// RegisterBluesky adds a Bluesky syndicator
func (m *Manager) RegisterBluesky(config BlueskyConfig, displayName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	syndicator := NewBluesky(config)
	m.syndicators["bluesky"] = syndicator
	m.targets = append(m.targets, SyndicationTarget{
		UID:      "bluesky",
		Name:     displayName,
		Platform: "bluesky",
	})
}

// RegisterMastodon adds a Mastodon syndicator
func (m *Manager) RegisterMastodon(config MastodonConfig, displayName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	syndicator := NewMastodon(config)
	m.syndicators["mastodon"] = syndicator
	m.targets = append(m.targets, SyndicationTarget{
		UID:      "mastodon",
		Name:     displayName,
		Platform: "mastodon",
	})
}

// GetTargets returns all available syndication targets
func (m *Manager) GetTargets() []SyndicationTarget {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent modification
	targets := make([]SyndicationTarget, len(m.targets))
	copy(targets, m.targets)
	return targets
}

// Syndicate sends a post to the specified targets
func (m *Manager) Syndicate(ctx context.Context, post *Post, targetUIDs []string) []SyndicationResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []SyndicationResult
	var wg sync.WaitGroup
	resultChan := make(chan SyndicationResult, len(targetUIDs))

	for _, uid := range targetUIDs {
		syndicator, ok := m.syndicators[uid]
		if !ok {
			results = append(results, SyndicationResult{
				Platform: uid,
				Success:  false,
				Error:    "unknown syndication target",
			})
			continue
		}

		wg.Add(1)
		go func(s Syndicator, targetUID string) {
			defer wg.Done()

			result := m.syndicateToTarget(ctx, s, post)
			resultChan <- *result
		}(syndicator, uid)
	}

	// Wait for all syndications to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// syndicateToTarget handles syndication to a single target
func (m *Manager) syndicateToTarget(ctx context.Context, syndicator Syndicator, post *Post) *SyndicationResult {
	var result *SyndicationResult
	var err error

	// Determine syndication strategy based on post type and content
	switch {
	case len(post.Images) > 0:
		// Post with images
		result, err = syndicator.PostWithImages(ctx, post)

	case post.Type == PostTypeBookmark || post.TargetURL != "":
		// Bookmark or post with target URL - use link card
		card, cardErr := FetchLinkCard(post.TargetURL)
		if cardErr != nil {
			log.Printf("Failed to fetch link card for %s: %v", post.TargetURL, cardErr)
			// Fall back to simple post
			result, err = syndicator.Post(ctx, post)
		} else {
			result, err = syndicator.PostWithLinkCard(ctx, post, card)
		}

	case post.Type == PostTypeArticle && len(post.Content) > 300:
		// Long article - use thread
		result, err = syndicator.PostThread(ctx, post)

	default:
		// Simple text post
		result, err = syndicator.Post(ctx, post)
	}

	if err != nil && result == nil {
		result = &SyndicationResult{
			Platform: syndicator.Platform(),
			Success:  false,
			Error:    err.Error(),
		}
	}

	return result
}

// SyndicateAll sends a post to all registered targets
func (m *Manager) SyndicateAll(ctx context.Context, post *Post) []SyndicationResult {
	m.mu.RLock()
	targetUIDs := make([]string, 0, len(m.syndicators))
	for uid := range m.syndicators {
		targetUIDs = append(targetUIDs, uid)
	}
	m.mu.RUnlock()

	return m.Syndicate(ctx, post, targetUIDs)
}

// HasTarget checks if a target UID is registered
func (m *Manager) HasTarget(uid string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.syndicators[uid]
	return ok
}

// ParseSyndicationTargets extracts target UIDs from mp-syndicate-to values
func ParseSyndicationTargets(values []string) []string {
	var targets []string
	seen := make(map[string]bool)

	for _, v := range values {
		// Handle both single values and comma-separated lists
		parts := strings.Split(v, ",")
		for _, part := range parts {
			uid := strings.TrimSpace(part)
			if uid != "" && !seen[uid] {
				seen[uid] = true
				targets = append(targets, uid)
			}
		}
	}

	return targets
}

// ResultsToSyndicationURLs converts results to a map of platform -> URL
func ResultsToSyndicationURLs(results []SyndicationResult) map[string]string {
	urls := make(map[string]string)
	for _, result := range results {
		if result.Success && result.URL != "" {
			urls[result.Platform] = result.URL
		}
	}
	return urls
}
