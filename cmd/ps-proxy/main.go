package main

import (
	"compress/gzip"
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/fsnotify/fsnotify"
)

const (
	sh = "sh"
)

// HugoBuilder manages Hugo builds with debouncing
type HugoBuilder struct {
	siteDir   string
	publicDir string
	mu        sync.Mutex
	building  bool
	pending   bool
	lastBuild time.Time
	debounce  time.Duration
}

func NewHugoBuilder(siteDir, publicDir string) *HugoBuilder {
	return &HugoBuilder{
		siteDir:   siteDir,
		publicDir: publicDir,
		debounce:  2 * time.Second,
	}
}

func (h *HugoBuilder) Build() error {
	h.mu.Lock()
	if h.building {
		h.pending = true
		h.mu.Unlock()
		return nil
	}
	h.building = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.building = false
		pending := h.pending
		h.pending = false
		h.mu.Unlock()

		if pending {
			time.Sleep(h.debounce)
			h.Build()
		}
	}()

	log.Println("Building Hugo site...")
	start := time.Now()

	cmd := exec.Command("hugo", "--gc", "--minify")
	cmd.Dir = h.siteDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Hugo build failed: %v", err)
		return err
	}

	h.lastBuild = time.Now()
	log.Printf("Hugo build completed in %v", time.Since(start))
	return nil
}

func (h *HugoBuilder) TriggerRebuild() {
	time.AfterFunc(h.debounce, func() {
		h.Build()
	})
}

// compressedResponseWriter wraps http.ResponseWriter with compression
type compressedResponseWriter struct {
	io.Writer
	http.ResponseWriter
	statusCode int
}

func (w *compressedResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *compressedResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// compressionMiddleware adds gzip/brotli compression
func compressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip compression for API requests (they're proxied and handle their own compression)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip compression for already compressed formats
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		skipCompression := map[string]bool{
			".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
			".mp4": true, ".webm": true, ".mp3": true, ".ogg": true,
			".woff": true, ".woff2": true, ".ttf": true,
			".zip": true, ".gz": true, ".br": true,
		}
		if skipCompression[ext] {
			next.ServeHTTP(w, r)
			return
		}

		acceptEncoding := r.Header.Get("Accept-Encoding")

		// Prefer Brotli over Gzip
		if strings.Contains(acceptEncoding, "br") {
			w.Header().Set("Content-Encoding", "br")
			w.Header().Del("Content-Length")
			br := brotli.NewWriterLevel(w, brotli.BestSpeed)
			defer br.Close()
			next.ServeHTTP(&compressedResponseWriter{Writer: br, ResponseWriter: w}, r)
			return
		}

		if strings.Contains(acceptEncoding, "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length")
			gz, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
			defer gz.Close()
			next.ServeHTTP(&compressedResponseWriter{Writer: gz, ResponseWriter: w}, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// securityHeadersMiddleware adds security headers
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HTTPS enforcement
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// XSS protection (legacy browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy (disable unnecessary APIs)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Content Security Policy - adjust as needed for your site
		// This is a relatively permissive policy; tighten for better security
		csp := strings.Join([]string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline' https://giscus.app",
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data: https: blob:",
			"font-src 'self' data:",
			"connect-src 'self' https://webmention.io https://giscus.app wss://giscus.app",
			"frame-src https://giscus.app",
			"frame-ancestors 'self'",
			"base-uri 'self'",
			"form-action 'self'",
		}, "; ")
		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}

func main() {
	laddr := flag.String("laddr", "localhost:1312", "Listen address for the proxy")
	siteDir := flag.String("sitedir", ".", "Hugo site directory")
	publicDir := flag.String("publicdir", "public", "Hugo public output directory")
	contentDir := flag.String("contentdir", "content", "Hugo content directory to watch")

	aurl := flag.String("aurl", "http://localhost:1314/", "URL to proxy request on /api to")
	acmd := flag.String("acmd", "go run ./cmd/ps-api", "Command to run for the API server")

	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	devMode := flag.Bool("dev", false, "Development mode: proxy to Hugo server instead of serving static files")
	surl := flag.String("surl", "http://localhost:1313/", "URL to proxy to in dev mode")
	scmd := flag.String("scmd", "hugo server -D --baseUrl=/ --appendPort=false", "Command for Hugo server in dev mode")

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start API server
	api := exec.CommandContext(ctx, sh, "-c", *acmd)
	api.Stdout = os.Stdout
	api.Stderr = os.Stderr
	api.Stdin = os.Stdin
	api.Start()

	var handler http.Handler

	if *devMode {
		// Development mode: proxy to Hugo server
		site := exec.CommandContext(ctx, sh, "-c", *scmd)
		site.Stdout = os.Stdout
		site.Stderr = os.Stderr
		site.Stdin = os.Stdin
		site.Start()

		log.Println("Development mode: proxying to Hugo server at", *surl)
		handler = createProxyHandler(*surl, *aurl, *verbose)
	} else {
		// Production mode: serve static files with file watching
		builder := NewHugoBuilder(*siteDir, *publicDir)

		// Initial build
		if err := builder.Build(); err != nil {
			log.Fatalf("Initial Hugo build failed: %v", err)
		}

		// Set up file watcher for content directory
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatalf("Failed to create file watcher: %v", err)
		}
		defer watcher.Close()

		// Watch content directory recursively
		contentPath := filepath.Join(*siteDir, *contentDir)
		filepath.Walk(contentPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return watcher.Add(path)
			}
			return nil
		})

		// Also watch data, layouts, and assets directories
		for _, dir := range []string{"data", "layouts", "assets", "static"} {
			dirPath := filepath.Join(*siteDir, dir)
			if _, err := os.Stat(dirPath); err == nil {
				filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if info.IsDir() {
						watcher.Add(path)
					}
					return nil
				})
			}
		}

		// Start file watcher goroutine
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
						if strings.HasPrefix(filepath.Base(event.Name), ".") ||
							strings.HasSuffix(event.Name, "~") ||
							strings.HasSuffix(event.Name, ".swp") {
							continue
						}
						log.Printf("Content changed: %s", event.Name)

						if event.Op&fsnotify.Create != 0 {
							if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
								watcher.Add(event.Name)
							}
						}

						builder.TriggerRebuild()
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Printf("Watcher error: %v", err)
				}
			}
		}()

		log.Println("Production mode: serving static files from", filepath.Join(*siteDir, *publicDir))
		staticHandler := createStaticHandler(filepath.Join(*siteDir, *publicDir), *aurl, *verbose)

		// Apply middleware: security headers -> compression -> static serving
		handler = securityHeadersMiddleware(compressionMiddleware(staticHandler))
	}

	log.Println("Proxy listening on", *laddr)
	panic(http.ListenAndServe(*laddr, handler))
}

// createProxyHandler creates a handler that proxies all requests (dev mode)
func createProxyHandler(siteURL, apiURL string, verbose bool) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var upstream *url.URL

		if strings.HasPrefix(r.URL.Path, "/api/") {
			up, _ := url.Parse(apiURL)
			upstream = up
			if verbose {
				log.Println("Proxying request to API", r.URL)
			}
		} else {
			up, _ := url.Parse(siteURL)
			upstream = up
			if verbose {
				log.Println("Proxying request to site", r.URL)
			}
		}

		r.URL.Scheme = upstream.Scheme
		r.URL.Host = upstream.Host

		previousPath := r.URL.Path
		r.URL.Path = path.Join(upstream.Path, r.URL.Path)
		if strings.HasSuffix(previousPath, "/") && !strings.HasSuffix(r.URL.Path, "/") {
			r.URL.Path += "/"
		}

		res, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			log.Println("Could not proxy request:", err)
			http.Error(rw, err.Error(), http.StatusBadGateway)
			return
		}

		for key, values := range res.Header {
			for _, value := range values {
				rw.Header().Add(key, value)
			}
		}

		rw.WriteHeader(res.StatusCode)

		if _, err := io.Copy(rw, res.Body); err != nil {
			log.Println("Could not send result to client:", err)
		}
	})
}

// createStaticHandler creates a handler that serves static files and proxies /api
func createStaticHandler(publicDir, apiURL string, verbose bool) http.Handler {
	fileServer := http.FileServer(http.Dir(publicDir))

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Proxy /api requests to the API server
		if strings.HasPrefix(r.URL.Path, "/api/") {
			upstream, _ := url.Parse(apiURL)

			if verbose {
				log.Println("Proxying request to API", r.URL)
			}

			r.URL.Scheme = upstream.Scheme
			r.URL.Host = upstream.Host

			previousPath := r.URL.Path
			r.URL.Path = path.Join(upstream.Path, r.URL.Path)
			if strings.HasSuffix(previousPath, "/") && !strings.HasSuffix(r.URL.Path, "/") {
				r.URL.Path += "/"
			}

			res, err := http.DefaultTransport.RoundTrip(r)
			if err != nil {
				log.Println("Could not proxy request:", err)
				http.Error(rw, err.Error(), http.StatusBadGateway)
				return
			}

			for key, values := range res.Header {
				for _, value := range values {
					rw.Header().Add(key, value)
				}
			}

			rw.WriteHeader(res.StatusCode)

			if _, err := io.Copy(rw, res.Body); err != nil {
				log.Println("Could not send result to client:", err)
			}
			return
		}

		// Serve static files
		if verbose {
			log.Println("Serving static file", r.URL.Path)
		}

		// Cache headers based on content type
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		switch {
		case ext == ".html" || ext == "":
			// HTML pages - short cache for faster updates
			rw.Header().Set("Cache-Control", "public, max-age=60")
		case ext == ".css" || ext == ".js":
			// Fingerprinted assets - long cache
			rw.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case ext == ".woff" || ext == ".woff2" || ext == ".ttf":
			// Fonts - long cache
			rw.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" || ext == ".svg" || ext == ".ico":
			// Images - long cache
			rw.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case ext == ".xml" || ext == ".json":
			// Feeds and data - medium cache
			rw.Header().Set("Cache-Control", "public, max-age=3600")
		default:
			rw.Header().Set("Cache-Control", "public, max-age=86400")
		}

		// Add Vary header for compression
		rw.Header().Set("Vary", "Accept-Encoding")

		// Check if file exists, serve 404.html for missing files
		requestPath := r.URL.Path
		if requestPath == "/" {
			requestPath = "/index.html"
		}

		// Try the exact path first
		filePath := filepath.Join(publicDir, filepath.Clean(requestPath))
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// Try adding index.html for directory-like paths
			if !strings.Contains(filepath.Base(requestPath), ".") {
				indexPath := filepath.Join(filePath, "index.html")
				if _, err := os.Stat(indexPath); os.IsNotExist(err) {
					// Serve 404 page
					rw.WriteHeader(http.StatusNotFound)
					http.ServeFile(rw, r, filepath.Join(publicDir, "404.html"))
					return
				}
			} else {
				// File with extension doesn't exist
				rw.WriteHeader(http.StatusNotFound)
				http.ServeFile(rw, r, filepath.Join(publicDir, "404.html"))
				return
			}
		}

		fileServer.ServeHTTP(rw, r)
	})
}
