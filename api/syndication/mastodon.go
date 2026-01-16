package syndication

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-mastodon"
)

const (
	mastodonMaxPostLength = 500 // characters
)

// MastodonConfig holds configuration for the Mastodon syndicator
type MastodonConfig struct {
	Server       string
	ClientID     string
	ClientSecret string
	AccessToken  string
}

// MastodonSyndicator implements syndication to Mastodon
type MastodonSyndicator struct {
	config MastodonConfig
	client *mastodon.Client
}

// NewMastodon creates a new Mastodon syndicator
func NewMastodon(config MastodonConfig) *MastodonSyndicator {
	client := mastodon.NewClient(&mastodon.Config{
		Server:       config.Server,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		AccessToken:  config.AccessToken,
	})

	return &MastodonSyndicator{
		config: config,
		client: client,
	}
}

// Platform returns the platform name
func (m *MastodonSyndicator) Platform() string {
	return "mastodon"
}

// getVisibility converts our Visibility type to mastodon.Visibility
func (m *MastodonSyndicator) getVisibility(v Visibility) string {
	switch v {
	case VisibilityUnlisted:
		return mastodon.VisibilityUnlisted
	case VisibilityPrivate:
		return mastodon.VisibilityFollowersOnly
	case VisibilityDirect:
		return mastodon.VisibilityDirectMessage
	default:
		return mastodon.VisibilityPublic
	}
}

// applyContentWarning applies content warning and sensitive settings to a toot
func (m *MastodonSyndicator) applyContentWarning(toot *mastodon.Toot, post *Post) {
	if post.ContentWarning != "" {
		toot.SpoilerText = post.ContentWarning
	}
	if post.Sensitive {
		toot.Sensitive = true
	}
}

// Post creates a simple text post
func (m *MastodonSyndicator) Post(ctx context.Context, post *Post) (*SyndicationResult, error) {
	content := m.formatContent(post)

	toot := &mastodon.Toot{
		Status:     content,
		Visibility: m.getVisibility(post.Visibility),
	}

	if post.Language != "" {
		toot.Language = post.Language
	}
	m.applyContentWarning(toot, post)

	status, err := m.client.PostStatus(ctx, toot)
	if err != nil {
		return &SyndicationResult{
			Platform: m.Platform(),
			Success:  false,
			Error:    err.Error(),
		}, err
	}

	return &SyndicationResult{
		Platform: m.Platform(),
		Success:  true,
		URL:      status.URL,
	}, nil
}

// PostWithLinkCard creates a post with a link (Mastodon auto-generates cards)
func (m *MastodonSyndicator) PostWithLinkCard(ctx context.Context, post *Post, card *LinkCard) (*SyndicationResult, error) {
	// Mastodon automatically generates link cards from URLs in the post
	// Just ensure the URL is in the content
	content := m.formatContent(post)

	// Make sure the target URL is included for card generation
	if card != nil && card.URL != "" && !strings.Contains(content, card.URL) {
		content = fmt.Sprintf("%s\n\n%s", content, card.URL)
	}

	toot := &mastodon.Toot{
		Status:     content,
		Visibility: m.getVisibility(post.Visibility),
	}

	if post.Language != "" {
		toot.Language = post.Language
	}
	m.applyContentWarning(toot, post)

	status, err := m.client.PostStatus(ctx, toot)
	if err != nil {
		return &SyndicationResult{
			Platform: m.Platform(),
			Success:  false,
			Error:    err.Error(),
		}, err
	}

	return &SyndicationResult{
		Platform: m.Platform(),
		Success:  true,
		URL:      status.URL,
	}, nil
}

// PostWithImages creates a post with image attachments
func (m *MastodonSyndicator) PostWithImages(ctx context.Context, post *Post) (*SyndicationResult, error) {
	content := m.formatContent(post)

	var mediaIDs []mastodon.ID

	for _, img := range post.Images {
		// Use UploadMediaFromMedia to set description during upload
		media := &mastodon.Media{
			File:        bytes.NewReader(img.Data),
			Description: img.AltText,
		}

		attachment, err := m.client.UploadMediaFromMedia(ctx, media)
		if err != nil {
			continue
		}

		mediaIDs = append(mediaIDs, attachment.ID)
	}

	toot := &mastodon.Toot{
		Status:     content,
		Visibility: m.getVisibility(post.Visibility),
		MediaIDs:   mediaIDs,
	}

	if post.Language != "" {
		toot.Language = post.Language
	}
	m.applyContentWarning(toot, post)

	status, err := m.client.PostStatus(ctx, toot)
	if err != nil {
		return &SyndicationResult{
			Platform: m.Platform(),
			Success:  false,
			Error:    err.Error(),
		}, err
	}

	return &SyndicationResult{
		Platform: m.Platform(),
		Success:  true,
		URL:      status.URL,
	}, nil
}

// PostThread splits long content into a thread
func (m *MastodonSyndicator) PostThread(ctx context.Context, post *Post) (*SyndicationResult, error) {
	content := m.formatContent(post)
	parts := m.splitIntoThread(content)

	if len(parts) == 1 {
		return m.Post(ctx, post)
	}

	result := &SyndicationResult{
		Platform:   m.Platform(),
		Success:    true,
		ThreadURLs: make([]string, 0, len(parts)),
	}

	var parentID mastodon.ID

	for i, part := range parts {
		toot := &mastodon.Toot{
			Status:     part,
			Visibility: m.getVisibility(post.Visibility),
		}

		if post.Language != "" {
			toot.Language = post.Language
		}
		// Only apply content warning to first post in thread
		if i == 0 {
			m.applyContentWarning(toot, post)
		}

		if i > 0 {
			toot.InReplyToID = parentID
		}

		status, err := m.client.PostStatus(ctx, toot)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create thread part %d: %v", i+1, err)
			return result, err
		}

		result.ThreadURLs = append(result.ThreadURLs, status.URL)

		if i == 0 {
			result.URL = status.URL
		}

		parentID = status.ID
	}

	return result, nil
}

// formatContent formats the post content for Mastodon
func (m *MastodonSyndicator) formatContent(post *Post) string {
	var content string

	switch post.Type {
	case PostTypeArticle:
		if post.Title != "" {
			content = fmt.Sprintf("%s\n\n%s", post.Title, post.URL)
		} else {
			content = post.URL
		}
	case PostTypeBookmark:
		if post.Content != "" {
			content = fmt.Sprintf("🔖 %s\n\n%s", post.Content, post.TargetURL)
		} else {
			content = fmt.Sprintf("🔖 %s", post.TargetURL)
		}
	case PostTypeLike:
		content = fmt.Sprintf("❤️ %s", post.TargetURL)
	case PostTypeRepost:
		content = fmt.Sprintf("🔁 %s", post.TargetURL)
	case PostTypeReply:
		content = post.Content
	default:
		content = post.Content
		if post.URL != "" && !strings.Contains(content, post.URL) {
			content = fmt.Sprintf("%s\n\n%s", content, post.URL)
		}
	}

	// Add hashtags
	if len(post.Tags) > 0 {
		tags := make([]string, len(post.Tags))
		for i, tag := range post.Tags {
			if !strings.HasPrefix(tag, "#") {
				tags[i] = "#" + tag
			} else {
				tags[i] = tag
			}
		}
		content = fmt.Sprintf("%s\n\n%s", content, strings.Join(tags, " "))
	}

	return content
}

// splitIntoThread splits long content into thread parts
func (m *MastodonSyndicator) splitIntoThread(content string) []string {
	charCount := utf8.RuneCountInString(content)

	if charCount <= mastodonMaxPostLength {
		return []string{content}
	}

	var parts []string
	paragraphs := strings.Split(content, "\n\n")

	var currentPart strings.Builder
	partNum := 1
	totalParts := (charCount / (mastodonMaxPostLength - 10)) + 1

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		suffix := fmt.Sprintf(" (%d/%d)", partNum, totalParts)
		suffixLen := utf8.RuneCountInString(suffix)
		maxLen := mastodonMaxPostLength - suffixLen

		if currentPart.Len() > 0 {
			testContent := currentPart.String() + "\n\n" + para
			if utf8.RuneCountInString(testContent) <= maxLen {
				currentPart.WriteString("\n\n")
				currentPart.WriteString(para)
				continue
			}

			parts = append(parts, currentPart.String()+suffix)
			partNum++
			currentPart.Reset()
		}

		if utf8.RuneCountInString(para) > maxLen {
			words := strings.Fields(para)
			for _, word := range words {
				if currentPart.Len() > 0 {
					testContent := currentPart.String() + " " + word
					if utf8.RuneCountInString(testContent) > maxLen {
						parts = append(parts, currentPart.String()+suffix)
						partNum++
						currentPart.Reset()
					} else {
						currentPart.WriteString(" ")
					}
				}
				currentPart.WriteString(word)
			}
		} else {
			currentPart.WriteString(para)
		}
	}

	if currentPart.Len() > 0 {
		suffix := fmt.Sprintf(" (%d/%d)", partNum, totalParts)
		parts = append(parts, currentPart.String()+suffix)
	}

	// Recalculate part numbers
	totalParts = len(parts)
	for i := range parts {
		idx := strings.LastIndex(parts[i], " (")
		if idx > 0 {
			parts[i] = parts[i][:idx]
		}
		parts[i] = fmt.Sprintf("%s (%d/%d)", parts[i], i+1, totalParts)
	}

	return parts
}
