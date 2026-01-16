package syndication

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
)

const (
	blueskyMaxPostLength = 300 // graphemes
	blueskyBaseURL       = "https://bsky.app"
)

// BlueskyConfig holds configuration for the Bluesky syndicator
type BlueskyConfig struct {
	Server     string
	Identifier string
	Password   string
}

// BlueskySyndicator implements syndication to Bluesky
type BlueskySyndicator struct {
	config     BlueskyConfig
	client     *xrpc.Client
	did        string
	authorized bool
}

// NewBluesky creates a new Bluesky syndicator
func NewBluesky(config BlueskyConfig) *BlueskySyndicator {
	if config.Server == "" {
		config.Server = "https://bsky.social"
	}

	return &BlueskySyndicator{
		config: config,
		client: &xrpc.Client{
			Host: config.Server,
			Client: &http.Client{
				Timeout: 30 * time.Second,
			},
		},
	}
}

// Platform returns the platform name
func (b *BlueskySyndicator) Platform() string {
	return "bluesky"
}

// applySelfLabels applies content warning labels to a Bluesky post
func (b *BlueskySyndicator) applySelfLabels(feedPost *bsky.FeedPost, post *Post) {
	if !post.Sensitive && post.ContentWarning == "" {
		return
	}

	var labels []*atproto.LabelDefs_SelfLabel

	// Check content warning text for known label keywords
	cwLower := strings.ToLower(post.ContentWarning)

	// Map common content warning phrases to Bluesky labels
	switch {
	case strings.Contains(cwLower, "porn") || strings.Contains(cwLower, "explicit"):
		labels = append(labels, &atproto.LabelDefs_SelfLabel{Val: "porn"})
	case strings.Contains(cwLower, "sexual") || strings.Contains(cwLower, "nsfw"):
		labels = append(labels, &atproto.LabelDefs_SelfLabel{Val: "sexual"})
	case strings.Contains(cwLower, "nude") || strings.Contains(cwLower, "nudity"):
		labels = append(labels, &atproto.LabelDefs_SelfLabel{Val: "nudity"})
	case strings.Contains(cwLower, "gore") || strings.Contains(cwLower, "violence") ||
		strings.Contains(cwLower, "graphic") || strings.Contains(cwLower, "blood"):
		labels = append(labels, &atproto.LabelDefs_SelfLabel{Val: "graphic-media"})
	case post.Sensitive:
		// Generic sensitive content - use graphic-media as catch-all
		labels = append(labels, &atproto.LabelDefs_SelfLabel{Val: "graphic-media"})
	}

	if len(labels) > 0 {
		feedPost.Labels = &bsky.FeedPost_Labels{
			LabelDefs_SelfLabels: &atproto.LabelDefs_SelfLabels{
				Values: labels,
			},
		}
	}
}

// authorize authenticates with Bluesky and stores the session
func (b *BlueskySyndicator) authorize(ctx context.Context) error {
	if b.authorized {
		return nil
	}

	session, err := atproto.ServerCreateSession(ctx, b.client, &atproto.ServerCreateSession_Input{
		Identifier: b.config.Identifier,
		Password:   b.config.Password,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	b.client.Auth = &xrpc.AuthInfo{
		AccessJwt:  session.AccessJwt,
		RefreshJwt: session.RefreshJwt,
		Handle:     session.Handle,
		Did:        session.Did,
	}
	b.did = session.Did
	b.authorized = true

	return nil
}

// Post creates a simple text post
func (b *BlueskySyndicator) Post(ctx context.Context, post *Post) (*SyndicationResult, error) {
	if err := b.authorize(ctx); err != nil {
		return &SyndicationResult{
			Platform: b.Platform(),
			Success:  false,
			Error:    err.Error(),
		}, err
	}

	content := b.formatContent(post)
	facets, err := b.extractFacets(ctx, content)
	if err != nil {
		// Non-fatal, continue without facets
		facets = nil
	}

	feedPost := &bsky.FeedPost{
		Text:      content,
		CreatedAt: time.Now().Format(time.RFC3339),
		Facets:    facets,
	}

	if post.Language != "" {
		feedPost.Langs = []string{post.Language}
	}
	b.applySelfLabels(feedPost, post)

	return b.createRecord(ctx, feedPost)
}

// PostWithLinkCard creates a post with an embedded link card
func (b *BlueskySyndicator) PostWithLinkCard(ctx context.Context, post *Post, card *LinkCard) (*SyndicationResult, error) {
	if err := b.authorize(ctx); err != nil {
		return &SyndicationResult{
			Platform: b.Platform(),
			Success:  false,
			Error:    err.Error(),
		}, err
	}

	content := b.formatContent(post)
	facets, _ := b.extractFacets(ctx, content)

	external := &bsky.EmbedExternal_External{
		Uri:         card.URL,
		Title:       card.Title,
		Description: card.Description,
	}

	// Upload thumbnail if available
	if len(card.Image) > 0 {
		blob, err := b.uploadBlob(ctx, card.Image, card.ImageType)
		if err == nil {
			external.Thumb = blob
		}
	}

	feedPost := &bsky.FeedPost{
		Text:      content,
		CreatedAt: time.Now().Format(time.RFC3339),
		Facets:    facets,
		Embed: &bsky.FeedPost_Embed{
			EmbedExternal: &bsky.EmbedExternal{
				External: external,
			},
		},
	}

	if post.Language != "" {
		feedPost.Langs = []string{post.Language}
	}
	b.applySelfLabels(feedPost, post)

	return b.createRecord(ctx, feedPost)
}

// PostWithImages creates a post with image attachments
func (b *BlueskySyndicator) PostWithImages(ctx context.Context, post *Post) (*SyndicationResult, error) {
	if err := b.authorize(ctx); err != nil {
		return &SyndicationResult{
			Platform: b.Platform(),
			Success:  false,
			Error:    err.Error(),
		}, err
	}

	content := b.formatContent(post)
	facets, _ := b.extractFacets(ctx, content)

	var images []*bsky.EmbedImages_Image
	for _, img := range post.Images {
		blob, err := b.uploadBlob(ctx, img.Data, img.MimeType)
		if err != nil {
			continue
		}

		images = append(images, &bsky.EmbedImages_Image{
			Image: blob,
			Alt:   img.AltText,
		})
	}

	feedPost := &bsky.FeedPost{
		Text:      content,
		CreatedAt: time.Now().Format(time.RFC3339),
		Facets:    facets,
	}

	if len(images) > 0 {
		feedPost.Embed = &bsky.FeedPost_Embed{
			EmbedImages: &bsky.EmbedImages{
				Images: images,
			},
		}
	}

	if post.Language != "" {
		feedPost.Langs = []string{post.Language}
	}
	b.applySelfLabels(feedPost, post)

	return b.createRecord(ctx, feedPost)
}

// PostThread splits long content into a thread
func (b *BlueskySyndicator) PostThread(ctx context.Context, post *Post) (*SyndicationResult, error) {
	if err := b.authorize(ctx); err != nil {
		return &SyndicationResult{
			Platform: b.Platform(),
			Success:  false,
			Error:    err.Error(),
		}, err
	}

	content := b.formatContent(post)
	parts := b.splitIntoThread(content)

	if len(parts) == 1 {
		// Short enough for a single post
		return b.Post(ctx, post)
	}

	result := &SyndicationResult{
		Platform:   b.Platform(),
		Success:    true,
		ThreadURLs: make([]string, 0, len(parts)),
	}

	var rootURI, rootCID string
	var parentURI, parentCID string

	for i, part := range parts {
		facets, _ := b.extractFacets(ctx, part)

		feedPost := &bsky.FeedPost{
			Text:      part,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second).Format(time.RFC3339),
			Facets:    facets,
		}

		if post.Language != "" {
			feedPost.Langs = []string{post.Language}
		}
		// Only apply content warning to first post in thread
		if i == 0 {
			b.applySelfLabels(feedPost, post)
		}

		// Add reply reference for thread continuation
		if i > 0 {
			feedPost.Reply = &bsky.FeedPost_ReplyRef{
				Root: &atproto.RepoStrongRef{
					Uri: rootURI,
					Cid: rootCID,
				},
				Parent: &atproto.RepoStrongRef{
					Uri: parentURI,
					Cid: parentCID,
				},
			}
		}

		resp, err := atproto.RepoCreateRecord(ctx, b.client, &atproto.RepoCreateRecord_Input{
			Collection: "app.bsky.feed.post",
			Repo:       b.did,
			Record:     &lexutil.LexiconTypeDecoder{Val: feedPost},
		})

		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create thread part %d: %v", i+1, err)
			return result, err
		}

		postURL := fmt.Sprintf("%s/profile/%s/post/%s", blueskyBaseURL, b.client.Auth.Handle, path.Base(resp.Uri))
		result.ThreadURLs = append(result.ThreadURLs, postURL)

		if i == 0 {
			rootURI = resp.Uri
			rootCID = resp.Cid
			result.URL = postURL
		}

		parentURI = resp.Uri
		parentCID = resp.Cid
	}

	return result, nil
}

// uploadBlob uploads binary data to Bluesky
func (b *BlueskySyndicator) uploadBlob(ctx context.Context, data []byte, mimeType string) (*lexutil.LexBlob, error) {
	resp, err := atproto.RepoUploadBlob(ctx, b.client, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to upload blob: %w", err)
	}

	return &lexutil.LexBlob{
		Ref:      resp.Blob.Ref,
		MimeType: mimeType,
		Size:     int64(len(data)),
	}, nil
}

// createRecord creates a post record
func (b *BlueskySyndicator) createRecord(ctx context.Context, post *bsky.FeedPost) (*SyndicationResult, error) {
	resp, err := atproto.RepoCreateRecord(ctx, b.client, &atproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       b.did,
		Record:     &lexutil.LexiconTypeDecoder{Val: post},
	})

	if err != nil {
		return &SyndicationResult{
			Platform: b.Platform(),
			Success:  false,
			Error:    err.Error(),
		}, err
	}

	postURL := fmt.Sprintf("%s/profile/%s/post/%s", blueskyBaseURL, b.client.Auth.Handle, path.Base(resp.Uri))

	return &SyndicationResult{
		Platform: b.Platform(),
		Success:  true,
		URL:      postURL,
	}, nil
}

// formatContent formats the post content for Bluesky
func (b *BlueskySyndicator) formatContent(post *Post) string {
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
		// Note or default
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
func (b *BlueskySyndicator) splitIntoThread(content string) []string {
	graphemeCount := utf8.RuneCountInString(content)

	if graphemeCount <= blueskyMaxPostLength {
		return []string{content}
	}

	// Split by sentences or paragraphs
	var parts []string
	paragraphs := strings.Split(content, "\n\n")

	var currentPart strings.Builder
	partNum := 1
	totalParts := (graphemeCount / (blueskyMaxPostLength - 10)) + 1 // Reserve space for "[n/m]"

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		suffix := fmt.Sprintf(" [%d/%d]", partNum, totalParts)
		suffixLen := utf8.RuneCountInString(suffix)
		maxLen := blueskyMaxPostLength - suffixLen

		if currentPart.Len() > 0 {
			testContent := currentPart.String() + "\n\n" + para
			if utf8.RuneCountInString(testContent) <= maxLen {
				currentPart.WriteString("\n\n")
				currentPart.WriteString(para)
				continue
			}

			// Current part is full, save it
			parts = append(parts, currentPart.String()+suffix)
			partNum++
			currentPart.Reset()
		}

		// Handle paragraphs that are too long themselves
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

	// Don't forget the last part
	if currentPart.Len() > 0 {
		suffix := fmt.Sprintf(" [%d/%d]", partNum, totalParts)
		parts = append(parts, currentPart.String()+suffix)
	}

	// Recalculate part numbers now that we know the total
	totalParts = len(parts)
	for i := range parts {
		// Remove old suffix and add correct one
		oldSuffix := regexp.MustCompile(`\s*\[\d+/\d+\]$`)
		parts[i] = oldSuffix.ReplaceAllString(parts[i], "")
		parts[i] = fmt.Sprintf("%s [%d/%d]", parts[i], i+1, totalParts)
	}

	return parts
}

// extractFacets extracts rich text facets (links, mentions, hashtags) from content
func (b *BlueskySyndicator) extractFacets(ctx context.Context, text string) ([]*bsky.RichtextFacet, error) {
	var facets []*bsky.RichtextFacet

	// Extract URLs
	urlPattern := regexp.MustCompile(`https?://[^\s<>\[\]]+`)
	for _, match := range urlPattern.FindAllStringIndex(text, -1) {
		url := text[match[0]:match[1]]
		// Clean trailing punctuation
		url = strings.TrimRight(url, ".,;:!?)")

		facets = append(facets, &bsky.RichtextFacet{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(match[0]),
				ByteEnd:   int64(match[0] + len(url)),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{
				{
					RichtextFacet_Link: &bsky.RichtextFacet_Link{
						Uri: url,
					},
				},
			},
		})
	}

	// Extract hashtags
	hashtagPattern := regexp.MustCompile(`#[a-zA-Z0-9_]+`)
	for _, match := range hashtagPattern.FindAllStringIndex(text, -1) {
		tag := text[match[0]+1 : match[1]] // Remove # prefix for tag value

		facets = append(facets, &bsky.RichtextFacet{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(match[0]),
				ByteEnd:   int64(match[1]),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{
				{
					RichtextFacet_Tag: &bsky.RichtextFacet_Tag{
						Tag: tag,
					},
				},
			},
		})
	}

	// Extract mentions (@handle.bsky.social or @did:plc:xxx)
	mentionPattern := regexp.MustCompile(`@([a-zA-Z0-9._-]+(\.[a-zA-Z0-9._-]+)+|did:plc:[a-zA-Z0-9]+)`)
	for _, matchIdx := range mentionPattern.FindAllStringIndex(text, -1) {
		handle := text[matchIdx[0]+1 : matchIdx[1]] // Remove @ prefix

		// Resolve handle to DID
		did := handle
		if !strings.HasPrefix(handle, "did:") {
			resolvedDID, err := b.resolveHandle(ctx, handle)
			if err != nil {
				continue
			}
			did = resolvedDID
		}

		facets = append(facets, &bsky.RichtextFacet{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(matchIdx[0]),
				ByteEnd:   int64(matchIdx[1]),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{
				{
					RichtextFacet_Mention: &bsky.RichtextFacet_Mention{
						Did: did,
					},
				},
			},
		})
	}

	return facets, nil
}

// resolveHandle resolves a Bluesky handle to a DID
func (b *BlueskySyndicator) resolveHandle(ctx context.Context, handle string) (string, error) {
	resp, err := atproto.IdentityResolveHandle(ctx, b.client, handle)
	if err != nil {
		return "", err
	}
	return resp.Did, nil
}
