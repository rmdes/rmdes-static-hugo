# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A personal website combining Hugo static site generation with Go backend services for dynamic data from external APIs (GitHub, Bluesky, Mastodon, Spotify, Twitch, YouTube). Built as a reusable template with PatternFly UI components.

## Commands

```bash
# Install dependencies (npm + Go) and copy assets
make depend

# Development server (proxy combines Hugo + API on localhost:1312)
make dev

# Production build (creates out/ directory with binaries and tarball)
make build

# Run tests
make test

# Build individual CLI tools
make build-cli/ps-api           # API server
make build-cli/ps-gen-projects  # Project list generator
make build-cli/ps-proxy         # Development proxy
make build-cli/ps-spotify-get-refresh-token

# Build CV from markdown (requires pandoc)
make build-cv

# Clean build artifacts
make clean
```

## Architecture

### Data Flow

```
External APIs (GitHub, Bluesky, etc.)
        ↓
    ps-api (port 1314) → JSON responses with 15-min cache
        ↓
    ps-proxy (port 1312) → routes /api/* to API, else to Hugo
        ↓
    Hugo server (port 1313) → renders templates with data/*.yaml
        ↓
    PatternFly UI components
```

### Key Components

**cmd/ps-proxy/** - Development proxy that starts both Hugo and API servers, routing requests by path prefix. Single entry point for local development.

**cmd/ps-api/** - HTTP API aggregating data from 7 external services. Each endpoint has panic recovery and 15-minute cache headers. All credentials come from flags or environment variables.

**cmd/ps-gen-projects/** - Reads `data/projects.yaml` and `data/forges.yaml`, fetches live metadata from GitHub/Forgejo, outputs `data/projects_gen.yaml` for Hugo templates.

**pkg/forges/** - Unified client interface for GitHub and Forgejo APIs. Types `InputProject` → `OutputProject` with full repo metadata (stars, forks, license, language, topics).

**api/*/** - Individual API handlers for Bluesky, Mastodon, Spotify, Twitch, YouTube, Twitter (via Nitter), and Git forges.

### Configuration

**data/*.yaml** - Hugo data files read by templates:
- `person.yaml` - Name, pronouns, description, image
- `integrations.yaml` - Enable/disable each service with usernames
- `projects.yaml` - Project list input (repo references)
- `projects_gen.yaml` - Generated project metadata (don't edit directly)
- `forges.yaml` - Git forge definitions (domain, type, API URL, CDN)
- `links.yaml` - Navigation and social links
- `effects.yaml` - Visual effects toggles

**Environment Variables** (required for full functionality):
- `FORGE_TOKENS` - JSON object: `{"github.com": "token", "codeberg.org": "token"}`
- Bluesky: `BLUESKY_SERVER`, `BLUESKY_PASSWORD`
- Mastodon: `MASTODON_SERVER`, `MASTODON_CLIENT_ID`, `MASTODON_CLIENT_SECRET`, `MASTODON_ACCESS_TOKEN`
- Spotify: `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET`, `SPOTIFY_REFRESH_TOKEN`
- Twitch: `TWITCH_CLIENT_ID`, `TWITCH_CLIENT_SECRET`
- YouTube: `YOUTUBE_TOKEN`

### Hugo Templates

- `layouts/index.html` - Homepage with grid layout
- `layouts/partials/*.html` - Social feed widgets (blueskyfeed, mastodonfeed, blogfeed, spotifystatus, streamstatus, lastcommit)
- `layouts/articles/single.html` - Article pages with Giscus comments
- `assets/scss/main.scss` - PatternFly customizations

### Project Format

Projects in `data/projects.yaml` use format: `domain/owner/repo`
```yaml
- title: Category Name
  projects:
    - repo: github.com/owner/repo
      background: var(--pf-t--color--black)
      icon: docs/icon.svg  # path within repo
```

## API Endpoints

All endpoints return JSON with `Cache-Control: s-maxage=900`:
- `/api/forges?username=xxx` - GitHub/Forgejo activity
- `/api/bluesky?username=xxx` - Bluesky posts
- `/api/mastodon?username=xxx` - Mastodon toots
- `/api/spotify?username=xxx` - Currently playing
- `/api/twitch?username=xxx` - Stream status
- `/api/youtube?username=xxx` - Channel status
- `/api/twitter?url=nitter-instance` - Twitter via Nitter
- `/api/blog?feedUrl=xxx` - External blog JSON feed
