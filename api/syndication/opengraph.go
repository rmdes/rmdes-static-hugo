package syndication

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	ogTitlePattern       = regexp.MustCompile(`<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']+)["']`)
	ogTitlePatternAlt    = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:title["']`)
	ogDescPattern        = regexp.MustCompile(`<meta[^>]+property=["']og:description["'][^>]+content=["']([^"']+)["']`)
	ogDescPatternAlt     = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:description["']`)
	ogImagePattern       = regexp.MustCompile(`<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']`)
	ogImagePatternAlt    = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:image["']`)
	titlePattern         = regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	metaDescPattern      = regexp.MustCompile(`<meta[^>]+name=["']description["'][^>]+content=["']([^"']+)["']`)
	metaDescPatternAlt   = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+name=["']description["']`)
)

// FetchLinkCard fetches OpenGraph metadata from a URL
func FetchLinkCard(url string) (*LinkCard, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SyndicationBot/1.0)")
	req.Header.Set("Accept", "text/html")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Limit read to 1MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, err
	}

	html := string(body)
	card := &LinkCard{URL: url}

	// Extract title (prefer og:title)
	if match := ogTitlePattern.FindStringSubmatch(html); len(match) > 1 {
		card.Title = decodeHTMLEntities(match[1])
	} else if match := ogTitlePatternAlt.FindStringSubmatch(html); len(match) > 1 {
		card.Title = decodeHTMLEntities(match[1])
	} else if match := titlePattern.FindStringSubmatch(html); len(match) > 1 {
		card.Title = decodeHTMLEntities(match[1])
	}

	// Extract description (prefer og:description)
	if match := ogDescPattern.FindStringSubmatch(html); len(match) > 1 {
		card.Description = decodeHTMLEntities(match[1])
	} else if match := ogDescPatternAlt.FindStringSubmatch(html); len(match) > 1 {
		card.Description = decodeHTMLEntities(match[1])
	} else if match := metaDescPattern.FindStringSubmatch(html); len(match) > 1 {
		card.Description = decodeHTMLEntities(match[1])
	} else if match := metaDescPatternAlt.FindStringSubmatch(html); len(match) > 1 {
		card.Description = decodeHTMLEntities(match[1])
	}

	// Extract image URL
	var imageURL string
	if match := ogImagePattern.FindStringSubmatch(html); len(match) > 1 {
		imageURL = match[1]
	} else if match := ogImagePatternAlt.FindStringSubmatch(html); len(match) > 1 {
		imageURL = match[1]
	}

	// Fetch image if found
	if imageURL != "" {
		imageData, imageType, err := fetchImage(imageURL)
		if err == nil {
			card.Image = imageData
			card.ImageType = imageType
		}
	}

	return card, nil
}

// fetchImage downloads an image and returns its bytes and MIME type
func fetchImage(url string) ([]byte, string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SyndicationBot/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	// Limit to 5MB for images
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectImageType(data)
	}

	return data, contentType, nil
}

// detectImageType tries to detect image type from magic bytes
func detectImageType(data []byte) string {
	if len(data) < 4 {
		return "application/octet-stream"
	}

	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}

	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// GIF
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
		return "image/gif"
	}

	// WebP
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	return "application/octet-stream"
}

// decodeHTMLEntities decodes common HTML entities
func decodeHTMLEntities(s string) string {
	replacements := map[string]string{
		"&amp;":   "&",
		"&lt;":    "<",
		"&gt;":    ">",
		"&quot;":  "\"",
		"&#39;":   "'",
		"&apos;":  "'",
		"&nbsp;":  " ",
		"&#x27;":  "'",
		"&#x2F;":  "/",
		"&#x60;":  "`",
		"&#x3D;":  "=",
	}

	result := s
	for entity, char := range replacements {
		result = strings.ReplaceAll(result, entity, char)
	}

	return strings.TrimSpace(result)
}
