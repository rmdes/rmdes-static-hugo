# IndieWeb Implementation Plan for rmdes-static-hugo

**Created**: 2026-01-16
**Status**: Ready for Implementation
**Estimated Total Effort**: 12-15 days

---

## Executive Summary

Implement full IndieWeb support for rmendes.net including microformats, webmentions (send/receive), Micropub (all content types), IndieAuth, Microsub feed exposure, and Bridgy backfeed. All features will be configurable via environment variables and follow existing codebase patterns.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         External Services                            │
├──────────────┬──────────────┬──────────────┬───────────────────────┤
│ webmention.io│ indieauth.com│    Bridgy    │   Target Sites        │
│ (receive)    │ (auth)       │ (backfeed)   │   (send webmentions)  │
└──────┬───────┴──────┬───────┴──────┬───────┴───────────┬───────────┘
       │              │              │                   │
       ▼              ▼              ▼                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        ps-api (Go Server)                            │
├─────────────┬─────────────┬─────────────┬─────────────┬─────────────┤
│ /api/       │ /api/       │ /api/       │ /api/       │ Existing    │
│ webmention  │ micropub    │ indieauth   │ webmention  │ Handlers    │
│ (receive)   │ (create)    │ (callback)  │ /send       │             │
└──────┬──────┴──────┬──────┴─────────────┴──────┬──────┴─────────────┘
       │             │                           │
       ▼             ▼                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      /app/data/ (Persistent Storage)                 │
├─────────────────┬───────────────────┬───────────────────────────────┤
│ webmentions/    │ site/content/     │ indieweb_state.json           │
│ cache.json      │ notes/            │ (tokens, queue)               │
│ (cached mentions)│ likes/           │                               │
│                 │ bookmarks/        │                               │
│                 │ reposts/          │                               │
└─────────────────┴───────────────────┴───────────────────────────────┘
       │                    │
       ▼                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Hugo Server (auto-rebuild)                       │
├─────────────────────────────────────────────────────────────────────┤
│ Templates with microformats (h-card, h-entry, h-feed)               │
│ Webmention display partial                                          │
│ New content type templates (notes, likes, bookmarks, reposts)       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Microformats & Giscus Fix (Foundation)

**Effort**: 2-3 days
**Dependencies**: None
**Goal**: Make site IndieWeb-readable with proper semantic markup

### Task 1.1: Add h-card to Homepage

**File**: `layouts/index.html`

Add author h-card in the profile section:

```html
<div class="h-card p-author">
  <img class="u-photo" src="{{ .Site.Data.person.img }}" alt="{{ .Site.Data.person.name }}" />
  <a class="p-name u-url" href="{{ .Site.Data.site.opengraph.baseUrl }}">{{ .Site.Data.person.name }}</a>
  <p class="p-note">{{ .Site.Data.person.description }}</p>
</div>
```

### Task 1.2: Add h-entry to Articles

**File**: `layouts/articles/single.html`

Wrap article content with h-entry microformat:

```html
<article class="h-entry">
  <h1 class="p-name">{{ .Title }}</h1>
  <time class="dt-published" datetime="{{ .Date.Format "2006-01-02T15:04:05Z07:00" }}">
    {{ .Date.Format "January 2, 2006" }}
  </time>
  <div class="e-content">
    {{ .Content }}
  </div>
  <a class="u-url" href="{{ .Permalink }}"></a>
  {{ partial "author-hcard" . }}
</article>
```

### Task 1.3: Add h-feed to Article Listings

**File**: `layouts/articles/list.html` or `layouts/_default/list.html`

```html
<div class="h-feed">
  <h1 class="p-name">{{ .Title }}</h1>
  {{ range .Pages }}
    <article class="h-entry">
      <h2><a class="p-name u-url" href="{{ .Permalink }}">{{ .Title }}</a></h2>
      <time class="dt-published" datetime="{{ .Date.Format "2006-01-02T15:04:05Z07:00" }}">
        {{ .Date.Format "January 2, 2006" }}
      </time>
      {{ with .Params.excerpt }}<p class="p-summary">{{ . }}</p>{{ end }}
    </article>
  {{ end }}
</div>
```

### Task 1.4: Create Author h-card Partial

**File**: `layouts/partials/author-hcard.html` (new)

```html
<div class="p-author h-card">
  <img class="u-photo" src="{{ .Site.Data.person.img }}" alt="" style="display:none" />
  <a class="p-name u-url" href="{{ .Site.Data.site.opengraph.baseUrl }}">
    {{ .Site.Data.person.name }}
  </a>
</div>
```

### Task 1.5: Fix Giscus Configuration

**File**: `data/integrations.yaml`

Add `enabled` flag and fix configuration:

```yaml
giscus:
  enabled: false  # NEW: toggle for Giscus
  repo: rmdes/rmdes-static-hugo  # Use the actual repo with discussions
  repoID: ""      # Get from https://giscus.app
  category: "Announcements"
  categoryID: ""  # Get from https://giscus.app
```

**File**: `layouts/articles/single.html`

Wrap Giscus in conditional:

```html
{{ if .Site.Data.integrations.giscus.enabled }}
  <script src="https://giscus.app/client.js"
    data-repo="{{ .Site.Data.integrations.giscus.repo }}"
    ...
  </script>
{{ end }}
```

### Task 1.6: Add IndieWeb Configuration

**File**: `data/integrations.yaml`

```yaml
indieweb:
  enabled: true
  # webmention.io configuration
  webmention:
    enabled: true
    endpoint: "https://webmention.io/rmendes.net/webmention"
    token: ""  # Set via WEBMENTION_IO_TOKEN env var
  # IndieAuth configuration
  indieauth:
    enabled: true
    endpoint: "https://indieauth.com/auth"
    tokenEndpoint: "https://tokens.indieauth.com/token"
  # Micropub configuration
  micropub:
    enabled: true
  # Bridgy for silo backfeed
  bridgy:
    enabled: true
    twitter: false
    mastodon: true
    bluesky: true
```

### Task 1.7: Add rel Links to Head

**File**: `layouts/_default/baseof.html`

Add to `<head>`:

```html
{{ if .Site.Data.integrations.indieweb.enabled }}
  <!-- IndieWeb endpoints -->
  <link rel="webmention" href="{{ .Site.Data.integrations.indieweb.webmention.endpoint }}" />
  <link rel="authorization_endpoint" href="{{ .Site.Data.integrations.indieweb.indieauth.endpoint }}" />
  <link rel="token_endpoint" href="{{ .Site.Data.integrations.indieweb.indieauth.tokenEndpoint }}" />
  <link rel="micropub" href="{{ .Site.Data.site.opengraph.baseUrl }}/api/micropub" />

  <!-- Identity verification (rel=me) -->
  {{ range .Site.Data.links.socials }}
    {{ if .link }}
      <link rel="me" href="{{ .link }}" />
    {{ end }}
  {{ end }}
{{ end }}
```

### Phase 1 Checklist
- [ ] h-card on homepage profile section
- [ ] h-entry on all article single pages
- [ ] h-feed on article list pages
- [ ] Author h-card partial created
- [ ] Giscus `enabled` flag added and conditional rendering
- [ ] IndieWeb config section in integrations.yaml
- [ ] rel="webmention", rel="authorization_endpoint", rel="me" links in head

---

## Phase 2: Webmention Receive & Display

**Effort**: 2-3 days
**Dependencies**: Phase 1 complete
**Goal**: Receive webmentions via webmention.io and display them on pages

### Task 2.1: Create Webmention API Handler

**File**: `api/webmention/index.go` (new)

```go
package webmention

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"
)

type Webmention struct {
    Source       string    `json:"source"`
    Target       string    `json:"target"`
    AuthorName   string    `json:"author_name"`
    AuthorPhoto  string    `json:"author_photo"`
    AuthorURL    string    `json:"author_url"`
    Content      string    `json:"content"`
    Published    string    `json:"published"`
    Type         string    `json:"type"` // like, reply, repost, mention
    WmID         int       `json:"wm-id"`
}

type WebmentionCache struct {
    Mentions  []Webmention `json:"mentions"`
    FetchedAt time.Time    `json:"fetchedAt"`
    mu        sync.RWMutex
}

var cache *WebmentionCache

func WebmentionHandler(w http.ResponseWriter, r *http.Request, cacheDir, token string, ttl int) {
    target := r.URL.Query().Get("target")

    // Load or refresh cache
    mentions, err := getCachedMentions(cacheDir, token, ttl)
    if err != nil {
        http.Error(w, "Failed to fetch webmentions", http.StatusInternalServerError)
        return
    }

    // Filter by target if specified
    if target != "" {
        mentions = filterByTarget(mentions, target)
    }

    // Group by type
    result := groupByType(mentions)

    json.NewEncoder(w).Encode(result)
}

func getCachedMentions(cacheDir, token string, ttl int) ([]Webmention, error) {
    cacheFile := filepath.Join(cacheDir, "webmentions_cache.json")

    // Check if cache is fresh
    if info, err := os.Stat(cacheFile); err == nil {
        if time.Since(info.ModTime()) < time.Duration(ttl)*time.Second {
            // Serve from cache
            data, _ := os.ReadFile(cacheFile)
            var cached WebmentionCache
            json.Unmarshal(data, &cached)
            return cached.Mentions, nil
        }
    }

    // Fetch from webmention.io
    mentions, err := fetchFromWebmentionIO(token)
    if err != nil {
        return nil, err
    }

    // Write cache
    cached := WebmentionCache{
        Mentions:  mentions,
        FetchedAt: time.Now(),
    }
    data, _ := json.MarshalIndent(cached, "", "  ")
    os.MkdirAll(cacheDir, 0755)
    os.WriteFile(cacheFile, data, 0644)

    return mentions, nil
}

func fetchFromWebmentionIO(token string) ([]Webmention, error) {
    url := fmt.Sprintf("https://webmention.io/api/mentions.jf2?token=%s&per-page=100", token)
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Parse JF2 response and convert to our format
    // ... implementation details
    return nil, nil
}
```

### Task 2.2: Register Webmention Handler in ps-api

**File**: `cmd/ps-api/main.go`

Add imports and handler registration:

```go
import (
    "github.com/pojntfx/felicitas.pojtinger.com/api/webmention"
)

// Add flags
webmentionToken := flag.String("webmention-token", "", "webmention.io API token")
cacheDir := flag.String("cache-dir", "/app/data", "Directory for cache files")

// Add handler
mux.HandleFunc("/api/webmentions", func(rw http.ResponseWriter, r *http.Request) {
    defer func() {
        if err := recover(); err != nil {
            log.Println("Error in webmention API:", err)
            http.Error(rw, "Error in webmention API", http.StatusInternalServerError)
        }
    }()
    rw.Header().Add("Cache-Control", fmt.Sprintf("s-maxage=%v", *ttl))
    webmention.WebmentionHandler(rw, r, *cacheDir, *webmentionToken, *ttl)
})
```

### Task 2.3: Create Webmention Display Partial

**File**: `layouts/partials/webmentions.html` (new)

```html
{{ if .Site.Data.integrations.indieweb.webmention.enabled }}
<div class="webmentions" id="webmentions-{{ .File.UniqueID | default "home" }}">
  <h3 class="pf-v6-c-title">Interactions</h3>

  <!-- Likes/Favorites -->
  <div class="webmentions__likes" style="display: none;">
    <h4>Likes</h4>
    <div class="webmentions__likes__faces"></div>
  </div>

  <!-- Reposts/Boosts -->
  <div class="webmentions__reposts" style="display: none;">
    <h4>Reposts</h4>
    <div class="webmentions__reposts__faces"></div>
  </div>

  <!-- Replies/Comments -->
  <div class="webmentions__replies" style="display: none;">
    <h4>Comments</h4>
    <ul class="webmentions__replies__list"></ul>
  </div>

  <!-- Mentions -->
  <div class="webmentions__mentions" style="display: none;">
    <h4>Mentions</h4>
    <ul class="webmentions__mentions__list"></ul>
  </div>
</div>

<template id="webmention-reply-template">
  <li class="webmention-reply h-cite">
    <a class="webmention-reply__author p-author h-card">
      <img class="webmention-reply__avatar u-photo" loading="lazy" />
      <span class="webmention-reply__name p-name"></span>
    </a>
    <div class="webmention-reply__content p-content"></div>
    <a class="webmention-reply__date u-url">
      <time class="dt-published"></time>
    </a>
  </li>
</template>

<script type="module">
  const baseUrl = "{{ .Site.Data.site.opengraph.baseUrl }}";
  const targetUrl = "{{ .Permalink | default .Site.Data.site.opengraph.baseUrl }}";
  const container = document.getElementById("webmentions-{{ .File.UniqueID | default "home" }}");

  try {
    const response = await fetch(`${baseUrl}/api/webmentions?target=${encodeURIComponent(targetUrl)}`);
    if (!response.ok) throw new Error("Failed to fetch");

    const data = await response.json();

    // Render likes
    if (data.likes?.length > 0) {
      const likesContainer = container.querySelector(".webmentions__likes");
      const facesContainer = likesContainer.querySelector(".webmentions__likes__faces");
      data.likes.forEach(like => {
        const img = document.createElement("img");
        img.src = like.author_photo || "/img/default-avatar.png";
        img.alt = like.author_name;
        img.title = like.author_name;
        img.className = "webmention-face";
        facesContainer.appendChild(img);
      });
      likesContainer.style.display = "";
    }

    // Render reposts
    if (data.reposts?.length > 0) {
      const repostsContainer = container.querySelector(".webmentions__reposts");
      const facesContainer = repostsContainer.querySelector(".webmentions__reposts__faces");
      data.reposts.forEach(repost => {
        const img = document.createElement("img");
        img.src = repost.author_photo || "/img/default-avatar.png";
        img.alt = repost.author_name;
        img.title = repost.author_name;
        img.className = "webmention-face";
        facesContainer.appendChild(img);
      });
      repostsContainer.style.display = "";
    }

    // Render replies
    if (data.replies?.length > 0) {
      const repliesContainer = container.querySelector(".webmentions__replies");
      const repliesList = repliesContainer.querySelector(".webmentions__replies__list");
      const template = document.getElementById("webmention-reply-template");

      data.replies.forEach(reply => {
        const clone = template.content.cloneNode(true);
        clone.querySelector(".webmention-reply__avatar").src = reply.author_photo || "/img/default-avatar.png";
        clone.querySelector(".webmention-reply__name").textContent = reply.author_name;
        clone.querySelector(".webmention-reply__content").textContent = reply.content;
        clone.querySelector(".webmention-reply__date").href = reply.source;
        clone.querySelector(".dt-published").textContent = new Date(reply.published).toLocaleDateString();
        clone.querySelector(".dt-published").setAttribute("datetime", reply.published);
        repliesList.appendChild(clone);
      });
      repliesContainer.style.display = "";
    }

    // Render mentions
    if (data.mentions?.length > 0) {
      const mentionsContainer = container.querySelector(".webmentions__mentions");
      const mentionsList = mentionsContainer.querySelector(".webmentions__mentions__list");
      data.mentions.forEach(mention => {
        const li = document.createElement("li");
        const a = document.createElement("a");
        a.href = mention.source;
        a.textContent = mention.author_name + " mentioned this";
        a.target = "_blank";
        li.appendChild(a);
        mentionsList.appendChild(li);
      });
      mentionsContainer.style.display = "";
    }

  } catch (error) {
    console.error("Failed to load webmentions:", error);
  }
</script>
{{ end }}
```

### Task 2.4: Add Webmention Styles

**File**: `assets/scss/main.scss`

```scss
// Webmentions
.webmentions {
  margin-top: var(--pf-t--global--spacer--xl);
  padding: var(--pf-t--global--spacer--lg);

  &__likes, &__reposts {
    margin-bottom: var(--pf-t--global--spacer--md);

    &__faces {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;
    }
  }

  &__replies, &__mentions {
    &__list {
      list-style: none;
      padding: 0;
    }
  }
}

.webmention-face {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
}

.webmention-reply {
  padding: var(--pf-t--global--spacer--md);
  margin-bottom: var(--pf-t--global--spacer--sm);
  border-left: 3px solid var(--pf-t--global--color--brand--200);

  &__author {
    display: flex;
    align-items: center;
    gap: var(--pf-t--global--spacer--sm);
    margin-bottom: var(--pf-t--global--spacer--xs);
  }

  &__avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
  }

  &__content {
    margin: var(--pf-t--global--spacer--sm) 0;
  }

  &__date {
    font-size: var(--pf-t--global--font--size--sm);
    color: var(--pf-t--global--color--status--info--100);
  }
}
```

### Task 2.5: Include Webmentions in Templates

**File**: `layouts/articles/single.html`

Add before/after Giscus:

```html
{{ partial "webmentions" . }}

{{ if .Site.Data.integrations.giscus.enabled }}
  <!-- Giscus script -->
{{ end }}
```

**File**: `layouts/index.html`

Add in the social area (after social feeds, before starred repos):

```html
{{ if .Site.Data.integrations.indieweb.webmention.enabled }}
  <section id="webmentions" class="pf-v6-u-mt-xl">
    {{ partial "webmentions" . }}
  </section>
{{ end }}
```

### Task 2.6: Update Environment Configuration

**File**: `/home/rick/code/rmdes-cloudron-app/start.sh`

Add to env.sh template:

```bash
# IndieWeb - webmention.io token
# Get from: https://webmention.io/settings
export WEBMENTION_IO_TOKEN=""
```

### Phase 2 Checklist
- [ ] Webmention API handler created (`api/webmention/index.go`)
- [ ] Handler registered in ps-api with caching
- [ ] Webmention display partial created
- [ ] Styles added for webmention display
- [ ] Webmentions included in article template
- [ ] Webmentions included in homepage (social area)
- [ ] Environment variable for webmention.io token

---

## Phase 3: IndieAuth & Basic Micropub

**Effort**: 3-4 days
**Dependencies**: Phase 1, Phase 2
**Goal**: Enable posting via Micropub clients using IndieAuth

### Task 3.1: Create IndieAuth Callback Handler

**File**: `api/indieauth/index.go` (new)

```go
package indieauth

import (
    "encoding/json"
    "net/http"
    "net/url"
    "os"
)

type TokenResponse struct {
    Me          string `json:"me"`
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
    Scope       string `json:"scope"`
}

// VerifyToken verifies an IndieAuth token with the token endpoint
func VerifyToken(token string) (*TokenResponse, error) {
    tokenEndpoint := "https://tokens.indieauth.com/token"

    req, _ := http.NewRequest("GET", tokenEndpoint, nil)
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Accept", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("token verification failed: %d", resp.StatusCode)
    }

    var tokenResp TokenResponse
    json.NewDecoder(resp.Body).Decode(&tokenResp)

    // Verify the token is for our site
    expectedMe := os.Getenv("INDIEAUTH_ME")
    if tokenResp.Me != expectedMe && tokenResp.Me != expectedMe+"/" {
        return nil, fmt.Errorf("token not for this site")
    }

    return &tokenResp, nil
}
```

### Task 3.2: Create Micropub Handler

**File**: `api/micropub/index.go` (new)

```go
package micropub

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/pojntfx/felicitas.pojtinger.com/api/indieauth"
)

type MicropubRequest struct {
    Type       []string            `json:"type"`
    Properties map[string][]any    `json:"properties"`
}

type MicropubResponse struct {
    Location string `json:"location,omitempty"`
    Error    string `json:"error,omitempty"`
}

func MicropubHandler(w http.ResponseWriter, r *http.Request, contentDir string) {
    // Handle GET for config query
    if r.Method == "GET" {
        handleQuery(w, r)
        return
    }

    // POST requires authentication
    token := extractToken(r)
    if token == "" {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    tokenResp, err := indieauth.VerifyToken(token)
    if err != nil {
        http.Error(w, "forbidden: "+err.Error(), http.StatusForbidden)
        return
    }

    // Check scope
    if !strings.Contains(tokenResp.Scope, "create") {
        http.Error(w, "insufficient_scope", http.StatusForbidden)
        return
    }

    // Parse request
    var req MicropubRequest
    contentType := r.Header.Get("Content-Type")

    if strings.Contains(contentType, "application/json") {
        json.NewDecoder(r.Body).Decode(&req)
    } else {
        // Form-encoded
        r.ParseForm()
        req = formToMicropub(r.Form)
    }

    // Determine post type
    postType := determinePostType(req)

    // Create content file
    location, err := createPost(req, postType, contentDir)
    if err != nil {
        http.Error(w, "Error creating post: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Location", location)
    w.WriteHeader(http.StatusCreated)
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query().Get("q")

    switch q {
    case "config":
        json.NewEncoder(w).Encode(map[string]any{
            "media-endpoint": "/api/micropub/media",
            "post-types": []map[string]string{
                {"type": "note", "name": "Note"},
                {"type": "article", "name": "Article"},
                {"type": "like", "name": "Like"},
                {"type": "bookmark", "name": "Bookmark"},
                {"type": "repost", "name": "Repost"},
            },
        })
    case "syndicate-to":
        json.NewEncoder(w).Encode(map[string]any{
            "syndicate-to": []any{},
        })
    default:
        http.Error(w, "invalid query", http.StatusBadRequest)
    }
}

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
    if _, ok := props["name"]; ok {
        return "article"
    }
    return "note"
}

func createPost(req MicropubRequest, postType, contentDir string) (string, error) {
    now := time.Now()
    slug := generateSlug(req, now)

    // Determine content directory based on type
    typeDir := postType + "s" // notes, articles, likes, etc.
    postDir := filepath.Join(contentDir, typeDir, slug)

    if err := os.MkdirAll(postDir, 0755); err != nil {
        return "", err
    }

    // Generate frontmatter
    frontmatter := generateFrontmatter(req, postType, now)

    // Get content
    content := ""
    if c, ok := req.Properties["content"]; ok && len(c) > 0 {
        switch v := c[0].(type) {
        case string:
            content = v
        case map[string]any:
            if html, ok := v["html"].(string); ok {
                content = html
            }
        }
    }

    // Write file
    filePath := filepath.Join(postDir, "index.md")
    fileContent := fmt.Sprintf("---\n%s---\n\n%s\n", frontmatter, content)

    if err := os.WriteFile(filePath, []byte(fileContent), 0644); err != nil {
        return "", err
    }

    // Return URL
    baseURL := os.Getenv("SITE_BASE_URL")
    return fmt.Sprintf("%s/%s/%s/", baseURL, typeDir, slug), nil
}

func generateSlug(req MicropubRequest, t time.Time) string {
    if slugs, ok := req.Properties["mp-slug"]; ok && len(slugs) > 0 {
        return sanitizeSlug(slugs[0].(string))
    }
    if names, ok := req.Properties["name"]; ok && len(names) > 0 {
        return sanitizeSlug(names[0].(string))
    }
    return t.Format("2006-01-02-150405")
}

func generateFrontmatter(req MicropubRequest, postType string, t time.Time) string {
    var lines []string

    // Title
    if names, ok := req.Properties["name"]; ok && len(names) > 0 {
        lines = append(lines, fmt.Sprintf("title: %q", names[0]))
    }

    // Date
    lines = append(lines, fmt.Sprintf("date: %s", t.Format(time.RFC3339)))

    // Post type
    lines = append(lines, fmt.Sprintf("type: %s", postType))

    // Type-specific properties
    switch postType {
    case "like":
        if urls, ok := req.Properties["like-of"]; ok && len(urls) > 0 {
            lines = append(lines, fmt.Sprintf("like-of: %q", urls[0]))
        }
    case "bookmark":
        if urls, ok := req.Properties["bookmark-of"]; ok && len(urls) > 0 {
            lines = append(lines, fmt.Sprintf("bookmark-of: %q", urls[0]))
        }
    case "repost":
        if urls, ok := req.Properties["repost-of"]; ok && len(urls) > 0 {
            lines = append(lines, fmt.Sprintf("repost-of: %q", urls[0]))
        }
    }

    // Categories/tags
    if cats, ok := req.Properties["category"]; ok && len(cats) > 0 {
        var tags []string
        for _, c := range cats {
            tags = append(tags, fmt.Sprintf("%q", c))
        }
        lines = append(lines, fmt.Sprintf("tags: [%s]", strings.Join(tags, ", ")))
    }

    return strings.Join(lines, "\n") + "\n"
}

func extractToken(r *http.Request) string {
    auth := r.Header.Get("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return strings.TrimPrefix(auth, "Bearer ")
    }
    return r.FormValue("access_token")
}

func sanitizeSlug(s string) string {
    s = strings.ToLower(s)
    s = strings.ReplaceAll(s, " ", "-")
    // Remove non-alphanumeric except hyphens
    var result []rune
    for _, r := range s {
        if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
            result = append(result, r)
        }
    }
    return string(result)
}
```

### Task 3.3: Register Micropub Handler

**File**: `cmd/ps-api/main.go`

```go
import (
    "github.com/pojntfx/felicitas.pojtinger.com/api/micropub"
)

// Add flag
contentDir := flag.String("content-dir", "/app/data/site/content", "Hugo content directory")

// Add handler
mux.HandleFunc("/api/micropub", func(rw http.ResponseWriter, r *http.Request) {
    defer func() {
        if err := recover(); err != nil {
            log.Println("Error in Micropub API:", err)
            http.Error(rw, "Error in Micropub API", http.StatusInternalServerError)
        }
    }()
    micropub.MicropubHandler(rw, r, *contentDir)
})
```

### Task 3.4: Create Content Type Templates

**File**: `layouts/notes/single.html` (new)

```html
{{ define "main" }}
<main class="pf-v6-c-page__main">
  <article class="h-entry pf-v6-u-p-xl">
    <div class="e-content">
      {{ .Content }}
    </div>
    <footer class="pf-v6-u-mt-md">
      <time class="dt-published" datetime="{{ .Date.Format "2006-01-02T15:04:05Z07:00" }}">
        {{ .Date.Format "January 2, 2006 at 3:04 PM" }}
      </time>
      <a class="u-url" href="{{ .Permalink }}">Permalink</a>
    </footer>
    {{ partial "author-hcard" . }}
  </article>

  {{ partial "webmentions" . }}
</main>
{{ end }}
```

**File**: `layouts/notes/list.html` (new)

```html
{{ define "main" }}
<main class="pf-v6-c-page__main">
  <div class="h-feed pf-v6-u-p-xl">
    <h1 class="p-name pf-v6-c-title">Notes</h1>
    {{ range .Pages }}
      <article class="h-entry pf-v6-u-mb-lg">
        <div class="e-content">{{ .Content }}</div>
        <footer>
          <time class="dt-published" datetime="{{ .Date.Format "2006-01-02T15:04:05Z07:00" }}">
            {{ .Date.Format "Jan 2, 2006" }}
          </time>
          <a class="u-url" href="{{ .Permalink }}">Permalink</a>
        </footer>
      </article>
    {{ end }}
  </div>
</main>
{{ end }}
```

**File**: `layouts/likes/single.html` (new)

```html
{{ define "main" }}
<main class="pf-v6-c-page__main">
  <article class="h-entry pf-v6-u-p-xl">
    <p class="p-summary">
      Liked <a class="u-like-of" href="{{ .Params.likeOf }}">{{ .Params.likeOf }}</a>
    </p>
    {{ with .Content }}<div class="e-content">{{ . }}</div>{{ end }}
    <footer class="pf-v6-u-mt-md">
      <time class="dt-published" datetime="{{ .Date.Format "2006-01-02T15:04:05Z07:00" }}">
        {{ .Date.Format "January 2, 2006" }}
      </time>
    </footer>
    {{ partial "author-hcard" . }}
  </article>
</main>
{{ end }}
```

Similar templates for `bookmarks` and `reposts`.

### Task 3.5: Create Content Directories

**File**: `content/notes/_index.md` (new)

```markdown
---
title: "Notes"
---
```

Similar for `content/likes/_index.md`, `content/bookmarks/_index.md`, `content/reposts/_index.md`.

### Task 3.6: Update start.sh for Content Preservation

**File**: `/home/rick/code/rmdes-cloudron-app/start.sh`

Update rsync to preserve Micropub-created content:

```bash
rsync -a \
  --exclude='data/projects_gen.yaml' \
  --exclude='data/starred_gen.yaml' \
  --exclude='content/notes/*' \
  --exclude='content/likes/*' \
  --exclude='content/bookmarks/*' \
  --exclude='content/reposts/*' \
  /app/code/ /app/data/site/
```

### Task 3.7: Add Environment Variables

**File**: `/home/rick/code/rmdes-cloudron-app/start.sh`

Add to env.sh template:

```bash
# IndieAuth - your site URL (with trailing slash)
export INDIEAUTH_ME="https://rmendes.net/"

# Site base URL for Micropub responses
export SITE_BASE_URL="https://rmendes.net"
```

### Phase 3 Checklist
- [ ] IndieAuth token verification implemented
- [ ] Micropub handler with config query support
- [ ] Post type detection (note, article, like, bookmark, repost)
- [ ] Content file generation with proper frontmatter
- [ ] Templates for notes, likes, bookmarks, reposts (single + list)
- [ ] Content directories created with _index.md
- [ ] start.sh updated to preserve Micropub content
- [ ] Environment variables added

---

## Phase 4: Webmention Send & Full Micropub

**Effort**: 3-4 days
**Dependencies**: Phase 3
**Goal**: Automatically send webmentions and complete Micropub features

### Task 4.1: Create Webmention Send Handler

**File**: `api/webmention/send.go` (new)

```go
package webmention

import (
    "io"
    "net/http"
    "net/url"
    "strings"

    "golang.org/x/net/html"
)

// SendWebmentions discovers and sends webmentions for all links in content
func SendWebmentions(sourceURL, content string) []error {
    var errors []error

    // Parse HTML to find links
    links := extractLinks(content)

    for _, targetURL := range links {
        // Discover webmention endpoint
        endpoint, err := discoverEndpoint(targetURL)
        if err != nil {
            continue // No endpoint found, skip
        }

        // Send webmention
        if err := sendWebmention(endpoint, sourceURL, targetURL); err != nil {
            errors = append(errors, err)
        }
    }

    return errors
}

func discoverEndpoint(targetURL string) (string, error) {
    resp, err := http.Get(targetURL)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // Check Link header first
    for _, link := range resp.Header.Values("Link") {
        if strings.Contains(link, `rel="webmention"`) || strings.Contains(link, `rel=webmention`) {
            // Parse endpoint from header
            parts := strings.Split(link, ";")
            if len(parts) > 0 {
                endpoint := strings.Trim(parts[0], "<> ")
                return resolveURL(targetURL, endpoint)
            }
        }
    }

    // Parse HTML for <link rel="webmention">
    doc, err := html.Parse(resp.Body)
    if err != nil {
        return "", err
    }

    endpoint := findWebmentionLink(doc)
    if endpoint != "" {
        return resolveURL(targetURL, endpoint)
    }

    return "", fmt.Errorf("no webmention endpoint found")
}

func sendWebmention(endpoint, source, target string) error {
    data := url.Values{}
    data.Set("source", source)
    data.Set("target", target)

    resp, err := http.PostForm(endpoint, data)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("webmention failed: %d", resp.StatusCode)
    }

    return nil
}
```

### Task 4.2: Integrate Send with Micropub

**File**: `api/micropub/index.go`

Add to `createPost` function after writing file:

```go
// After successful post creation, queue webmention sending
go func() {
    time.Sleep(5 * time.Second) // Give Hugo time to rebuild
    webmention.SendWebmentions(location, content)
}()
```

### Task 4.3: Add Media Endpoint

**File**: `api/micropub/media.go` (new)

```go
package micropub

import (
    "crypto/rand"
    "encoding/hex"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"

    "github.com/pojntfx/felicitas.pojtinger.com/api/indieauth"
)

func MediaHandler(w http.ResponseWriter, r *http.Request, mediaDir string) {
    if r.Method != "POST" {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Authenticate
    token := extractToken(r)
    if token == "" {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    tokenResp, err := indieauth.VerifyToken(token)
    if err != nil || !strings.Contains(tokenResp.Scope, "media") {
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    }

    // Parse multipart form
    if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
        http.Error(w, "invalid form", http.StatusBadRequest)
        return
    }

    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "no file provided", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Generate unique filename
    ext := filepath.Ext(header.Filename)
    randomBytes := make([]byte, 16)
    rand.Read(randomBytes)
    filename := hex.EncodeToString(randomBytes) + ext

    // Save file
    filePath := filepath.Join(mediaDir, filename)
    if err := os.MkdirAll(mediaDir, 0755); err != nil {
        http.Error(w, "failed to create directory", http.StatusInternalServerError)
        return
    }

    dst, err := os.Create(filePath)
    if err != nil {
        http.Error(w, "failed to save file", http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    if _, err := io.Copy(dst, file); err != nil {
        http.Error(w, "failed to save file", http.StatusInternalServerError)
        return
    }

    // Return URL
    baseURL := os.Getenv("SITE_BASE_URL")
    mediaURL := fmt.Sprintf("%s/media/%s", baseURL, filename)

    w.Header().Set("Location", mediaURL)
    w.WriteHeader(http.StatusCreated)
}
```

### Task 4.4: Register Media Handler

**File**: `cmd/ps-api/main.go`

```go
mediaDir := flag.String("media-dir", "/app/data/site/static/media", "Media upload directory")

mux.HandleFunc("/api/micropub/media", func(rw http.ResponseWriter, r *http.Request) {
    defer func() {
        if err := recover(); err != nil {
            log.Println("Error in Media API:", err)
            http.Error(rw, "Error in Media API", http.StatusInternalServerError)
        }
    }()
    micropub.MediaHandler(rw, r, *mediaDir)
})
```

### Task 4.5: Add Bridgy Links

**File**: `layouts/_default/baseof.html`

Add Bridgy webmention links if enabled:

```html
{{ if .Site.Data.integrations.indieweb.bridgy.enabled }}
  {{ if .Site.Data.integrations.indieweb.bridgy.mastodon }}
    <link rel="webmention" href="https://brid.gy/webmention/mastodon" />
  {{ end }}
  {{ if .Site.Data.integrations.indieweb.bridgy.bluesky }}
    <link rel="webmention" href="https://brid.gy/webmention/bluesky" />
  {{ end }}
{{ end }}
```

### Task 4.6: Update Micropub Config Response

**File**: `api/micropub/index.go`

Update `handleQuery` to include media endpoint:

```go
case "config":
    baseURL := os.Getenv("SITE_BASE_URL")
    json.NewEncoder(w).Encode(map[string]any{
        "media-endpoint": baseURL + "/api/micropub/media",
        "post-types": []map[string]string{
            {"type": "note", "name": "Note"},
            {"type": "article", "name": "Article"},
            {"type": "like", "name": "Like"},
            {"type": "bookmark", "name": "Bookmark"},
            {"type": "repost", "name": "Repost"},
        },
        "syndicate-to": []any{},
    })
```

### Phase 4 Checklist
- [ ] Webmention endpoint discovery implemented
- [ ] Webmention sending implemented
- [ ] Automatic webmention sending after Micropub post
- [ ] Media upload endpoint created
- [ ] Media handler registered
- [ ] Bridgy links added for backfeed
- [ ] Micropub config returns media endpoint

---

## Phase 5: Microsub & Polish

**Effort**: 1-2 days
**Dependencies**: Phase 4
**Goal**: Expose Microsub feed and polish the implementation

### Task 5.1: Create Microsub Feed Endpoint

**File**: `api/microsub/index.go` (new)

```go
package microsub

import (
    "encoding/json"
    "net/http"
    "os"
    "path/filepath"
    "sort"
    "time"

    "gopkg.in/yaml.v3"
)

type JF2Feed struct {
    Type     string     `json:"type"`
    Name     string     `json:"name"`
    Children []JF2Entry `json:"children"`
}

type JF2Entry struct {
    Type      string   `json:"type"`
    Name      string   `json:"name,omitempty"`
    Content   JF2Text  `json:"content,omitempty"`
    Published string   `json:"published"`
    URL       string   `json:"url"`
    Author    JF2Card  `json:"author"`
}

type JF2Text struct {
    Text string `json:"text,omitempty"`
    HTML string `json:"html,omitempty"`
}

type JF2Card struct {
    Type  string `json:"type"`
    Name  string `json:"name"`
    URL   string `json:"url"`
    Photo string `json:"photo,omitempty"`
}

func MicrosubHandler(w http.ResponseWriter, r *http.Request, contentDir string) {
    w.Header().Set("Content-Type", "application/jf2feed+json")

    baseURL := os.Getenv("SITE_BASE_URL")

    // Collect all posts
    var entries []JF2Entry

    // Read from different content types
    for _, contentType := range []string{"articles", "notes", "likes", "bookmarks", "reposts"} {
        typeDir := filepath.Join(contentDir, contentType)
        dirs, _ := os.ReadDir(typeDir)

        for _, dir := range dirs {
            if !dir.IsDir() || dir.Name() == "_index.md" {
                continue
            }

            indexPath := filepath.Join(typeDir, dir.Name(), "index.md")
            entry, err := parseContentFile(indexPath, baseURL, contentType, dir.Name())
            if err == nil {
                entries = append(entries, entry)
            }
        }
    }

    // Sort by date descending
    sort.Slice(entries, func(i, j int) bool {
        ti, _ := time.Parse(time.RFC3339, entries[i].Published)
        tj, _ := time.Parse(time.RFC3339, entries[j].Published)
        return ti.After(tj)
    })

    // Limit to 50 entries
    if len(entries) > 50 {
        entries = entries[:50]
    }

    feed := JF2Feed{
        Type:     "feed",
        Name:     "Ricardo Mendes",
        Children: entries,
    }

    json.NewEncoder(w).Encode(feed)
}
```

### Task 5.2: Add Microsub Link

**File**: `layouts/_default/baseof.html`

```html
{{ if .Site.Data.integrations.indieweb.enabled }}
  <link rel="microsub" href="{{ .Site.Data.site.opengraph.baseUrl }}/api/microsub" />
{{ end }}
```

### Task 5.3: Register Microsub Handler

**File**: `cmd/ps-api/main.go`

```go
import (
    "github.com/pojntfx/felicitas.pojtinger.com/api/microsub"
)

mux.HandleFunc("/api/microsub", func(rw http.ResponseWriter, r *http.Request) {
    defer func() {
        if err := recover(); err != nil {
            log.Println("Error in Microsub API:", err)
            http.Error(rw, "Error in Microsub API", http.StatusInternalServerError)
        }
    }()
    rw.Header().Add("Cache-Control", fmt.Sprintf("s-maxage=%v", *ttl))
    microsub.MicrosubHandler(rw, r, *contentDir)
})
```

### Task 5.4: Add Navigation for New Content Types

**File**: `layouts/index.html`

Add links to notes, likes, etc. in appropriate section:

```html
{{ if .Site.Data.integrations.indieweb.micropub.enabled }}
<section id="activity" class="pf-v6-u-mt-xl">
  <div class="pf-v6-l-flex pf-m-space-items-md">
    <a href="/notes/" class="pf-v6-c-button pf-m-tertiary">Notes</a>
    <a href="/likes/" class="pf-v6-c-button pf-m-tertiary">Likes</a>
    <a href="/bookmarks/" class="pf-v6-c-button pf-m-tertiary">Bookmarks</a>
  </div>
</section>
{{ end }}
```

### Task 5.5: Final Configuration Updates

**File**: `data/integrations.yaml`

Ensure complete configuration:

```yaml
indieweb:
  enabled: true
  webmention:
    enabled: true
    endpoint: "https://webmention.io/rmendes.net/webmention"
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

### Task 5.6: Update Dockerfile

**File**: `/home/rick/code/rmdes-cloudron-app/Dockerfile`

Ensure Go dependencies are fetched:

```dockerfile
# After cloning, fetch any new dependencies
RUN go mod download
RUN go mod tidy
```

### Task 5.7: Documentation

Create a brief README section or INDIEWEB.md documenting:
- How to configure webmention.io
- How to use Micropub clients (Quill, Indigenous, etc.)
- Environment variables needed
- How to verify rel=me links for IndieAuth

### Phase 5 Checklist
- [ ] Microsub JF2 feed endpoint created
- [ ] Microsub link added to head
- [ ] Handler registered in ps-api
- [ ] Navigation links for new content types
- [ ] Complete integrations.yaml configuration
- [ ] Dockerfile updated for dependencies
- [ ] Documentation created

---

## Environment Variables Summary

Add all these to `/home/rick/code/rmdes-cloudron-app/start.sh` env.sh template:

```bash
# ============================================
# IndieWeb Configuration
# ============================================

# webmention.io API token (get from https://webmention.io/settings)
export WEBMENTION_IO_TOKEN=""

# Your site URL for IndieAuth verification (with trailing slash)
export INDIEAUTH_ME="https://rmendes.net/"

# Site base URL for generating permalinks
export SITE_BASE_URL="https://rmendes.net"
```

---

## Testing Checklist

### Phase 1 Testing
- [ ] Validate microformats with https://indiewebify.me/
- [ ] Check h-card parsed correctly with https://php.microformats.io/
- [ ] Verify rel=me links resolve bidirectionally
- [ ] Test Giscus toggle on/off

### Phase 2 Testing
- [ ] Send test webmention from webmention.rocks
- [ ] Verify webmentions display on article pages
- [ ] Check webmention caching works (check /app/data/webmentions_cache.json)

### Phase 3 Testing
- [ ] Test IndieAuth login flow at https://indieauth.com/
- [ ] Use Quill (https://quill.p3k.io/) to create a note
- [ ] Verify note appears in Hugo after creation
- [ ] Test all post types (note, like, bookmark, repost)

### Phase 4 Testing
- [ ] Create post with link, verify webmention sent
- [ ] Upload media via Micropub client
- [ ] Verify Bridgy backfeed (may take time)

### Phase 5 Testing
- [ ] Validate Microsub feed with a reader (Monocle, Together)
- [ ] Full end-to-end test: create post, send mention, receive reply

---

## Rollback Plan

If issues arise:
1. Set `indieweb.enabled: false` in `data/integrations.yaml`
2. All features are conditionally rendered
3. API endpoints will still exist but can be disabled by removing handlers

---

## Future Enhancements (Out of Scope)

- Full Microsub server for reading feeds
- Webmention moderation UI
- Syndication to silos (POSSE)
- Private posts with IndieAuth
- Webmention analytics

---

## References

- [IndieWeb.org](https://indieweb.org/)
- [Webmention Spec](https://www.w3.org/TR/webmention/)
- [Micropub Spec](https://www.w3.org/TR/micropub/)
- [IndieAuth Spec](https://indieauth.spec.indieweb.org/)
- [Microformats2](https://microformats.org/wiki/microformats2)
- [go.hacdias.com/indielib](https://pkg.go.dev/go.hacdias.com/indielib)
- [willnorris.com/go/webmention](https://pkg.go.dev/willnorris.com/go/webmention)
