package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHandleStatic_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	s := &Server{target: tmpDir}

	req := httptest.NewRequest(http.MethodGet, "/nonexistent.html", nil)
	w := httptest.NewRecorder()

	s.handleStatic(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleStatic_DirectoryIndex(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "posts")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexContent := "<html><body>Hello</body></html>"
	if err := os.WriteFile(filepath.Join(subDir, "index.html"), []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{target: tmpDir}

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()

	s.handleStatic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Hello") {
		t.Error("expected body to contain index.html content")
	}
}

func TestHandleStatic_HTMLInjection(t *testing.T) {
	tmpDir := t.TempDir()
	htmlContent := "<html><body>Content</body></html>"
	if err := os.WriteFile(filepath.Join(tmpDir, "test.html"), []byte(htmlContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{target: tmpDir}

	req := httptest.NewRequest(http.MethodGet, "/test.html", nil)
	w := httptest.NewRecorder()

	s.handleStatic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "EventSource") {
		t.Error("expected live reload script to be injected")
	}
	if !strings.Contains(body, "__livereload") {
		t.Error("expected livereload endpoint in injected script")
	}
	if strings.Count(body, "</body>") != 1 {
		t.Errorf("expected exactly one </body> tag, got %d", strings.Count(body, "</body>"))
	}
}

func TestHandleStatic_NonHTMLNotInjected(t *testing.T) {
	tmpDir := t.TempDir()
	cssContent := "body { color: red; }"
	if err := os.WriteFile(filepath.Join(tmpDir, "styles.css"), []byte(cssContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{target: tmpDir}

	req := httptest.NewRequest(http.MethodGet, "/styles.css", nil)
	w := httptest.NewRecorder()

	s.handleStatic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleStatic_RootIndex(t *testing.T) {
	tmpDir := t.TempDir()
	indexContent := "<html><body>Home</body></html>"
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{target: tmpDir}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	s.handleStatic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Home") {
		t.Error("expected index.html content")
	}
}

func TestHandleSSE_Headers(t *testing.T) {
	s := &Server{
		clients: make(map[chan struct{}]struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/__livereload", nil)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	s.handleSSE(w, req)

	if got := w.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: connected") {
		t.Error("expected initial 'connected' event")
	}
}

func TestNotifyClients(t *testing.T) {
	s := &Server{
		clients: make(map[chan struct{}]struct{}),
	}

	ch := make(chan struct{}, 1)
	s.clients[ch] = struct{}{}

	s.notifyClients()

	select {
	case <-ch:
	default:
		t.Error("expected client channel to receive notification")
	}
}
