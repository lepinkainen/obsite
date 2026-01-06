package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"obsite/internal/generator"

	"github.com/fsnotify/fsnotify"
)

const (
	defaultAddr   = "127.0.0.1:8000"
	debounceDelay = 100 * time.Millisecond
)

// liveReloadScript is injected into HTML pages for automatic browser refresh
const liveReloadScript = `<script>
(function() {
    const es = new EventSource('/__livereload');
    es.addEventListener('reload', function() {
        location.reload();
    });
    es.onerror = function() {
        console.log('[obsite] Live reload disconnected, retrying...');
    };
})();
</script></body>`

// Server handles development serving with live reload
type Server struct {
	gen       *generator.Generator
	source    string
	target    string
	clients   map[chan struct{}]struct{}
	clientsMu sync.Mutex
}

// New creates a new development server
func New(gen *generator.Generator, source, target string) *Server {
	return &Server{
		gen:     gen,
		source:  source,
		target:  target,
		clients: make(map[chan struct{}]struct{}),
	}
}

// Run starts the development server
func (s *Server) Run() error {
	// Initial build
	fmt.Println("Building site...")
	if err := s.gen.Generate(); err != nil {
		return fmt.Errorf("initial build: %w", err)
	}

	// Start file watcher
	go s.watchFiles()

	// Set up HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/__livereload", s.handleSSE)
	mux.HandleFunc("/", s.handleStatic)

	fmt.Printf("Serving at http://%s\n", defaultAddr)
	fmt.Println("Press Ctrl+C to stop")

	return http.ListenAndServe(defaultAddr, mux)
}

func (s *Server) watchFiles() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not start file watcher: %v\n", err)
		return
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not stop file watcher: %v\n", err)
		}
	}()

	// Add source directory recursively
	if err := s.addWatchRecursive(watcher, s.source); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not watch source: %v\n", err)
		return
	}

	var (
		timer *time.Timer
		mu    sync.Mutex
	)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Skip non-relevant events
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			// Handle new directories
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := watcher.Add(event.Name); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not watch new dir %s: %v\n", event.Name, err)
					}
				}
			}

			// Debounce: reset timer on each event
			mu.Lock()
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounceDelay, func() {
				s.rebuild()
			})
			mu.Unlock()

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}

func (s *Server) addWatchRecursive(watcher *fsnotify.Watcher, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}

func (s *Server) rebuild() {
	fmt.Println("Rebuilding site...")
	start := time.Now()
	if err := s.gen.Generate(); err != nil {
		fmt.Fprintf(os.Stderr, "Rebuild error: %v\n", err)
		return
	}
	fmt.Printf("Rebuilt in %v\n", time.Since(start).Round(time.Millisecond))
	s.notifyClients()
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	ch := make(chan struct{}, 1)
	s.clientsMu.Lock()
	s.clients[ch] = struct{}{}
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, ch)
		s.clientsMu.Unlock()
		close(ch)
	}()

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	if _, err := fmt.Fprintf(w, "event: connected\ndata: ok\n\n"); err != nil {
		return
	}
	flusher.Flush()

	// Wait for reload events or client disconnect
	for {
		select {
		case <-ch:
			if _, err := fmt.Fprintf(w, "event: reload\ndata: reload\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) notifyClients() {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	for ch := range s.clients {
		select {
		case ch <- struct{}{}:
		default:
			// Channel full, skip (client will get next event)
		}
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Clean and resolve path
	path := filepath.Clean(r.URL.Path)
	if path == "/" {
		path = "/index.html"
	}

	// Check if requesting a directory (add index.html)
	fullPath := filepath.Join(s.target, path)
	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		fullPath = filepath.Join(fullPath, "index.html")
	}

	// Try to open the file
	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not close %s: %v\n", fullPath, err)
		}
	}()

	// For HTML files, inject live reload script
	if strings.HasSuffix(fullPath, ".html") {
		content, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Inject live reload script before </body>
		modified := bytes.Replace(content, []byte("</body>"), []byte(liveReloadScript), 1)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if _, err := w.Write(modified); err != nil {
			return
		}
		return
	}

	// For non-HTML files, serve directly with appropriate content type
	http.ServeFile(w, r, fullPath)
}
