package syndication

import "context"

// PostType represents the type of content being syndicated
type PostType string

const (
	PostTypeNote     PostType = "note"
	PostTypeArticle  PostType = "article"
	PostTypeBookmark PostType = "bookmark"
	PostTypeLike     PostType = "like"
	PostTypeRepost   PostType = "repost"
	PostTypeReply    PostType = "reply"
)

// Visibility represents post visibility level
type Visibility string

const (
	VisibilityPublic   Visibility = "public"   // Visible to everyone
	VisibilityUnlisted Visibility = "unlisted" // Visible but not in public timelines
	VisibilityPrivate  Visibility = "private"  // Followers only
	VisibilityDirect   Visibility = "direct"   // Direct message to mentioned users
)

// Post represents content to be syndicated
type Post struct {
	// Content is the main text content
	Content string

	// Title is optional, used for articles
	Title string

	// URL is the canonical URL of the post on the origin site
	URL string

	// Type indicates what kind of post this is
	Type PostType

	// TargetURL is the URL being bookmarked, liked, reposted, or replied to
	TargetURL string

	// Images contains any images to attach
	Images []Image

	// Tags contains hashtags for the post
	Tags []string

	// Language is the ISO 639-1 language code
	Language string

	// Visibility controls who can see the post (Mastodon-specific)
	// Values: public, unlisted, private, direct
	Visibility Visibility

	// ContentWarning is the spoiler text / content warning message
	ContentWarning string

	// Sensitive marks the post as containing sensitive content
	// On Mastodon: hides media behind a warning
	// On Bluesky: applies content labels
	Sensitive bool
}

// Image represents an image attachment
type Image struct {
	// Data is the raw image bytes
	Data []byte

	// MimeType is the image MIME type (e.g., "image/jpeg")
	MimeType string

	// AltText is the accessibility description
	AltText string
}

// LinkCard represents OpenGraph metadata for link embeds
type LinkCard struct {
	URL         string
	Title       string
	Description string
	Image       []byte
	ImageType   string
}

// SyndicationResult represents the result of syndicating to a platform
type SyndicationResult struct {
	// Platform is the name of the platform (e.g., "bluesky", "mastodon")
	Platform string

	// URL is the URL of the syndicated post
	URL string

	// Success indicates whether syndication succeeded
	Success bool

	// Error contains any error message
	Error string

	// ThreadURLs contains URLs if the post was split into a thread
	ThreadURLs []string
}

// Syndicator is the interface for platform-specific syndication
type Syndicator interface {
	// Post creates a simple text post
	Post(ctx context.Context, post *Post) (*SyndicationResult, error)

	// PostWithLinkCard creates a post with an embedded link card
	PostWithLinkCard(ctx context.Context, post *Post, card *LinkCard) (*SyndicationResult, error)

	// PostWithImages creates a post with image attachments
	PostWithImages(ctx context.Context, post *Post) (*SyndicationResult, error)

	// PostThread splits long content into a thread
	PostThread(ctx context.Context, post *Post) (*SyndicationResult, error)

	// Platform returns the platform name
	Platform() string
}
