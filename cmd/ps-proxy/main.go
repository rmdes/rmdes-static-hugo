package main

import (
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

	"github.com/fsnotify/fsnotify"
)

const (
	sh = "sh"
)

// HugoBuilder manages Hugo builds with debouncing
type HugoBuilder struct {
	siteDir    string
	publicDir  string
	mu         sync.Mutex
	building   bool
	pending    bool
	lastBuild  time.Time
	debounce   time.Duration
}

func NewHugoBuilder(siteDir, publicDir string) *HugoBuilder {
	return &HugoBuilder{
		siteDir:   siteDir,
		publicDir: publicDir,
		debounce:  2 * time.Second, // Wait 2 seconds after last change before rebuilding
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
			// Another build was requested while we were building
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
	// Debounce: wait a bit before building to batch rapid changes
	time.AfterFunc(h.debounce, func() {
		h.Build()
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
		err = filepath.Walk(contentPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return watcher.Add(path)
			}
			return nil
		})
		if err != nil {
			log.Printf("Warning: Failed to watch content directory: %v", err)
		}

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
					// Trigger rebuild on write, create, or remove events
					if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
						// Skip temporary files and hidden files
						if strings.HasPrefix(filepath.Base(event.Name), ".") ||
							strings.HasSuffix(event.Name, "~") ||
							strings.HasSuffix(event.Name, ".swp") {
							continue
						}
						log.Printf("Content changed: %s", event.Name)

						// If a new directory was created, watch it too
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
		handler = createStaticHandler(filepath.Join(*siteDir, *publicDir), *aurl, *verbose)
	}

	log.Println("Proxy listening on", *laddr)
	panic(http.ListenAndServe(*laddr, handler))
}

// createProxyHandler creates a handler that proxies all requests (dev mode)
func createProxyHandler(siteURL, apiURL string, verbose bool) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var upstream *url.URL

		if strings.HasPrefix(r.URL.Path, "/api/") {
			up, err := url.Parse(apiURL)
			if err != nil {
				log.Println("Could not parse API URL", err)
			}
			upstream = up

			if verbose {
				log.Println("Proxying request to API", r.URL)
			}
		} else {
			up, err := url.Parse(siteURL)
			if err != nil {
				log.Println("Could not parse site URL", err)
			}
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
			return
		}
	})
}

// createStaticHandler creates a handler that serves static files and proxies /api
func createStaticHandler(publicDir, apiURL string, verbose bool) http.Handler {
	fileServer := http.FileServer(http.Dir(publicDir))

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Proxy /api requests to the API server
		if strings.HasPrefix(r.URL.Path, "/api/") {
			upstream, err := url.Parse(apiURL)
			if err != nil {
				log.Println("Could not parse API URL", err)
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}

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

		// Add cache headers for static assets
		if strings.HasPrefix(r.URL.Path, "/assets/") ||
			strings.HasPrefix(r.URL.Path, "/uploads/") ||
			strings.HasSuffix(r.URL.Path, ".css") ||
			strings.HasSuffix(r.URL.Path, ".js") ||
			strings.HasSuffix(r.URL.Path, ".woff2") ||
			strings.HasSuffix(r.URL.Path, ".png") ||
			strings.HasSuffix(r.URL.Path, ".jpg") ||
			strings.HasSuffix(r.URL.Path, ".svg") {
			rw.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			// HTML pages - short cache for faster updates
			rw.Header().Set("Cache-Control", "public, max-age=60")
		}

		fileServer.ServeHTTP(rw, r)
	})
}
