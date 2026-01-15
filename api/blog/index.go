package blog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type Output struct {
	Title       string `json:"title"`
	HomePageURL string `json:"homePageURL"`
	Posts       []Post `json:"posts"`
}

type Post struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Timestamp string `json:"timestamp"`
}

func BlogFeedHandler(w http.ResponseWriter, r *http.Request, feedURL string) {
	if feedURL == "" {
		feedURL = r.URL.Query().Get("feedUrl")
	}

	if feedURL == "" {
		http.Error(w, "missing feedUrl", http.StatusBadRequest)
		return
	}

	// Use gofeed to parse any feed format (RSS, Atom, JSON Feed)
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		panic(fmt.Errorf("failed to parse feed: %w", err))
	}

	output := Output{
		Title:       feed.Title,
		HomePageURL: feed.Link,
		Posts:       []Post{},
	}

	// Limit to 4 most recent posts
	limit := 4
	if len(feed.Items) < limit {
		limit = len(feed.Items)
	}

	for _, item := range feed.Items[:limit] {
		post := Post{
			URL:   item.Link,
			Title: item.Title,
		}

		// Use description/summary, strip HTML if needed
		summary := item.Description
		if summary == "" {
			summary = item.Content
		}
		// Simple HTML tag stripping
		summary = stripHTML(summary)
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		post.Summary = summary

		// Parse timestamp
		if item.PublishedParsed != nil {
			post.Timestamp = item.PublishedParsed.Format(time.RFC3339)
		} else if item.Published != "" {
			post.Timestamp = item.Published
		}

		output.Posts = append(output.Posts, post)
	}

	j, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%v", string(j))
}

// stripHTML removes HTML tags from a string
func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}
