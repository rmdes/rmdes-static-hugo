# rmendes.net - Personal Site

A Hugo + Go personal website with full IndieWeb support, forked from [pojntfx/felicitas.pojtinger.com](https://github.com/pojntfx/felicitas.pojtinger.com).

![Go Version](https://img.shields.io/badge/go%20version-%3E=1.24-61CFDD.svg)

## Overview

This is a personal website template featuring:

- **Personal profile** with name, profession, bio, and social links
- **GitHub activity carousel** showing recent commits across repositories
- **Project showcase** fetched from GitHub/Forgejo/Codeberg
- **Starred repositories** section
- **Blog/Articles** with Giscus comments
- **Social feeds** (Bluesky, Mastodon)
- **Livestream status** (Twitch, YouTube)
- **Full IndieWeb support** (see below)

## Fork Enhancements

This fork adds several features not present in the upstream:

### GitHub Activity Carousel

A dynamic carousel showing recent GitHub activity:
- Displays commits across all your repositories
- Shows commit message, repository name, and timestamp
- Auto-rotates with smooth transitions
- Loading skeleton prevents layout shifts

### OpenGraph Support

Enhanced social sharing with proper meta tags:
- Title, description, and image for all pages
- Twitter/X card support
- Configurable default image

### IndieWeb Integration

Full IndieWeb protocol support for the open, decentralized social web:

#### Microformats2

Semantic markup for machine-readable content:
- **h-card**: Author identity on homepage and articles
- **h-entry**: Individual posts (articles, notes, likes, bookmarks, reposts)
- **h-feed**: List pages for content discovery

#### Webmention

Receive and display interactions from across the web:
- Integration with [webmention.io](https://webmention.io) for receiving webmentions
- Display likes, reposts, replies, and mentions on articles
- Cached API responses for performance

#### IndieAuth

Decentralized authentication:
- `rel="me"` links for identity verification
- Authorization and token endpoint discovery
- Token verification for Micropub/Microsub

#### Micropub

Create content from any Micropub client:
- Support for notes, articles, likes, bookmarks, and reposts
- Automatic Hugo content file generation
- Proper frontmatter and slug generation
- Token-based authentication

#### Microsub

Feed reader functionality:
- Channel management (folders/categories)
- Subscribe to RSS/Atom/JSON feeds
- Timeline aggregation with read status
- JF2 format output

### Cloudron Deployment

Optimized for [Cloudron](https://cloudron.io) deployment:
- Multi-stage Docker build
- Persistent data handling
- Environment variable configuration
- Automatic content preservation across updates

## Configuration

### Environment Variables

Create a `.env` or `env.sh` file with:

```bash
# GitHub (for projects and activity)
export FORGE_TOKENS='{"github.com": "your-github-token"}'
export GITHUB_USERNAME="yourusername"
export GITHUB_ACTIVITY_LIMIT="5"

# YouTube (optional)
export YOUTUBE_TOKEN="your-youtube-api-key"

# Bluesky (optional)
export BLUESKY_IDENTIFIER="you.bsky.social"
export BLUESKY_APP_PASSWORD="your-app-password"

# Mastodon (optional)
export MASTODON_ACCESS_TOKEN="your-token"
export MASTODON_INSTANCE="mastodon.social"

# IndieWeb
export WEBMENTION_IO_TOKEN="your-webmention-io-token"
export INDIEAUTH_ME="https://yoursite.com/"
export INDIEAUTH_TOKEN_ENDPOINT=""  # default: https://tokens.indieauth.com/token
export SITE_BASE_URL="https://yoursite.com"
```

### Data Files

Configuration is stored in `data/` YAML files:

- **`person.yaml`**: Name, profession, bio, profile image
- **`links.yaml`**: Social media links (with `rel="me"` support)
- **`site.yaml`**: Site title, OpenGraph settings
- **`forges.yaml`**: GitHub/Forgejo instances for projects
- **`projects.yaml`**: Manual project definitions
- **`integrations.yaml`**: Toggle features (Giscus, IndieWeb, etc.)

### IndieWeb Configuration

In `data/integrations.yaml`:

```yaml
indieweb:
  enabled: true
  webmention:
    enabled: true
    endpoint: "https://webmention.io/yoursite.com/webmention"
  indieauth:
    enabled: true
    endpoint: "https://indieauth.com/auth"
    tokenEndpoint: "https://tokens.indieauth.com/token"
  micropub:
    enabled: true
  microsub:
    enabled: true
  bridgy:
    enabled: true
    mastodon: true
    bluesky: true
```

## Development

### Prerequisites

- Go 1.24+
- Hugo (extended version recommended)
- Node.js (for SCSS compilation)

### Local Development

```bash
# Clone the repository
git clone https://github.com/rmdes/rmdes-static-hugo.git
cd rmdes-static-hugo

# Install dependencies
make depend

# Set environment variables (see Configuration above)
export FORGE_TOKENS='{"github.com": "your-token"}'

# Generate project data
make generate

# Start development server
make dev
```

### Building

```bash
# Build static site
make build

# Build all Go binaries
go build ./...

# Build Docker image
docker build -t personal-site .
```

### Project Structure

```
.
├── api/                    # Go API handlers
│   ├── blog/              # Blog feed API
│   ├── bluesky/           # Bluesky feed
│   ├── forges/            # GitHub/Forgejo API
│   ├── indieauth/         # IndieAuth token verification
│   ├── mastodon/          # Mastodon feed
│   ├── micropub/          # Micropub endpoint
│   ├── microsub/          # Microsub endpoint
│   ├── webmention/        # Webmention.io integration
│   └── youtube/           # YouTube status
├── assets/
│   └── scss/              # Stylesheets
├── cmd/
│   ├── ps-api/            # API server
│   ├── ps-gen-projects/   # Project list generator
│   ├── ps-gen-starred/    # Starred repos generator
│   └── ps-proxy/          # Reverse proxy
├── content/
│   ├── articles/          # Blog posts
│   ├── notes/             # Short notes (Micropub)
│   ├── likes/             # Likes (Micropub)
│   ├── bookmarks/         # Bookmarks (Micropub)
│   └── reposts/           # Reposts (Micropub)
├── data/                  # YAML configuration
├── layouts/               # Hugo templates
│   ├── _default/          # Base templates
│   ├── articles/          # Article templates
│   ├── notes/             # Note templates
│   ├── likes/             # Like templates
│   ├── bookmarks/         # Bookmark templates
│   ├── reposts/           # Repost templates
│   └── partials/          # Reusable components
└── static/                # Static assets
```

## IndieWeb Setup Guide

### 1. Webmention.io

1. Sign up at [webmention.io](https://webmention.io)
2. Verify your domain via `rel="me"` link
3. Copy your API token to `WEBMENTION_IO_TOKEN`
4. Update `webmention.endpoint` in `data/integrations.yaml`

### 2. IndieAuth

1. Add `rel="me"` links to your social profiles
2. Ensure your social profiles link back to your site
3. The default IndieAuth.com endpoints work out of the box

### 3. Micropub Clients

Compatible clients include:
- [Quill](https://quill.p3k.io) - Web-based
- [Indigenous](https://indigenous.realize.be/) - iOS/Android
- [Omnibear](https://omnibear.com/) - Browser extension

### 4. Microsub Readers

Compatible readers include:
- [Monocle](https://monocle.p3k.io)
- [Together](https://together.tilde.cafe)
- [Indigenous](https://indigenous.realize.be/)

### 5. Bridgy (optional)

To syndicate to/from social networks:
1. Sign up at [brid.gy](https://brid.gy)
2. Connect your Mastodon/Bluesky accounts
3. Enable in `data/integrations.yaml`

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/forges` | GET | GitHub activity and projects |
| `/api/bluesky` | GET | Bluesky feed |
| `/api/mastodon` | GET | Mastodon feed |
| `/api/youtube` | GET | YouTube live status |
| `/api/webmentions?target=URL` | GET | Webmentions for a page |
| `/api/micropub` | GET/POST | Micropub endpoint |
| `/api/microsub` | GET/POST | Microsub endpoint |

## Acknowledgements

### Upstream

- [pojntfx/felicitas.pojtinger.com](https://github.com/pojntfx/felicitas.pojtinger.com) - Original project by Felicitas Pojtinger

### Libraries & Tools

- [gohugoio/hugo](https://github.com/gohugoio/hugo) - Static site generator
- [google/go-github](https://github.com/google/go-github) - GitHub API
- [giscus/giscus](https://github.com/giscus/giscus) - Comment system
- [PatternFly](https://www.patternfly.org/) - Design system

### IndieWeb

- [webmention.io](https://webmention.io) - Webmention receiving service
- [indieauth.com](https://indieauth.com) - IndieAuth provider
- [brid.gy](https://brid.gy) - Social network bridge

## License

Personal Site (c) 2025 Felicitas Pojtinger, Ricardo Mendes, and contributors

SPDX-License-Identifier: AGPL-3.0
