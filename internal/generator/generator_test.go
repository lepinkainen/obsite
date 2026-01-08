package generator

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"obsite/internal/models"
)

//go:embed testdata/templates/*
var testTemplateFS embed.FS

func createTestPost(slug string, daysAgo int) *models.Post {
	return &models.Post{
		Title:   "Post " + slug,
		Slug:    slug,
		Created: models.FlexibleTime{Time: time.Now().AddDate(0, 0, -daysAgo)},
		Content: "Test content",
	}
}

func TestGenerateIndex_ZeroPosts(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "output")

	gen := &Generator{
		Target: target,
		Posts:  []*models.Post{},
	}

	// Load templates
	var err error
	gen.templates, err = gen.templates.ParseFS(testTemplateFS, "testdata/templates/*.html")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	if err := gen.generateIndex(); err != nil {
		t.Fatalf("generateIndex() error = %v", err)
	}

	// Check that index.html was created
	indexPath := filepath.Join(target, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("index.html was not created for 0 posts")
	}

	// No pagination pages should exist
	page2Path := filepath.Join(target, "page", "2", "index.html")
	if _, err := os.Stat(page2Path); !os.IsNotExist(err) {
		t.Error("page/2/index.html should not exist for 0 posts")
	}
}

func TestGenerateIndex_Pagination(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "output")

	// Create 25 posts (should result in 3 pages: 10, 10, 5)
	posts := make([]*models.Post, 25)
	for i := 0; i < 25; i++ {
		posts[i] = createTestPost("post-"+string(rune('a'+i)), i)
	}

	gen := &Generator{
		Target: target,
		Posts:  posts,
	}

	// Load templates
	var err error
	gen.templates, err = gen.templates.ParseFS(testTemplateFS, "testdata/templates/*.html")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	if err := gen.generateIndex(); err != nil {
		t.Fatalf("generateIndex() error = %v", err)
	}

	// Check main index
	indexPath := filepath.Join(target, "index.html")
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}

	// Page 1 should have NextPage link to page 2
	if !strings.Contains(string(indexContent), "/page/2/") {
		t.Error("index.html should contain link to /page/2/")
	}

	// Page 1 should NOT have PrevPage link
	if strings.Contains(string(indexContent), "Newer Posts") {
		t.Error("index.html should not have 'Newer Posts' link")
	}

	// Check page 2
	page2Path := filepath.Join(target, "page", "2", "index.html")
	page2Content, err := os.ReadFile(page2Path)
	if err != nil {
		t.Fatalf("failed to read page/2/index.html: %v", err)
	}

	// Page 2 should have PrevPage link to /
	if !strings.Contains(string(page2Content), "Newer Posts") {
		t.Error("page 2 should have 'Newer Posts' link")
	}

	// Page 2 should have NextPage link to page 3
	if !strings.Contains(string(page2Content), "/page/3/") {
		t.Error("page 2 should contain link to /page/3/")
	}

	// Check page 3 (last page)
	page3Path := filepath.Join(target, "page", "3", "index.html")
	page3Content, err := os.ReadFile(page3Path)
	if err != nil {
		t.Fatalf("failed to read page/3/index.html: %v", err)
	}

	// Page 3 should have PrevPage link to page 2
	if !strings.Contains(string(page3Content), "/page/2/") {
		t.Error("page 3 should have link to /page/2/")
	}

	// Page 3 should NOT have NextPage link
	if strings.Contains(string(page3Content), "Older Posts") {
		t.Error("page 3 (last page) should not have 'Older Posts' link")
	}

	// Page 4 should NOT exist
	page4Path := filepath.Join(target, "page", "4", "index.html")
	if _, err := os.Stat(page4Path); !os.IsNotExist(err) {
		t.Error("page/4/index.html should not exist")
	}
}

func TestGenerateIndex_ExactlyTenPosts(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "output")

	// Create exactly 10 posts (should be 1 page, no pagination)
	posts := make([]*models.Post, 10)
	for i := 0; i < 10; i++ {
		posts[i] = createTestPost("post-"+string(rune('a'+i)), i)
	}

	gen := &Generator{
		Target: target,
		Posts:  posts,
	}

	var err error
	gen.templates, err = gen.templates.ParseFS(testTemplateFS, "testdata/templates/*.html")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	if err := gen.generateIndex(); err != nil {
		t.Fatalf("generateIndex() error = %v", err)
	}

	indexContent, err := os.ReadFile(filepath.Join(target, "index.html"))
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}

	// Should have no pagination links
	if strings.Contains(string(indexContent), "/page/2/") {
		t.Error("10 posts should not need pagination")
	}

	// Page 2 should not exist
	page2Path := filepath.Join(target, "page", "2", "index.html")
	if _, err := os.Stat(page2Path); !os.IsNotExist(err) {
		t.Error("page/2/index.html should not exist for exactly 10 posts")
	}
}

func TestGenerateIndex_ElevenPosts(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "output")

	// Create 11 posts (should be 2 pages: 10 + 1)
	posts := make([]*models.Post, 11)
	for i := 0; i < 11; i++ {
		posts[i] = createTestPost("post-"+string(rune('a'+i)), i)
	}

	gen := &Generator{
		Target: target,
		Posts:  posts,
	}

	var err error
	gen.templates, err = gen.templates.ParseFS(testTemplateFS, "testdata/templates/*.html")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	if err := gen.generateIndex(); err != nil {
		t.Fatalf("generateIndex() error = %v", err)
	}

	// Page 1 should have link to page 2
	indexContent, err := os.ReadFile(filepath.Join(target, "index.html"))
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	if !strings.Contains(string(indexContent), "/page/2/") {
		t.Error("11 posts should have pagination to page 2")
	}

	// Page 2 should exist with 1 post
	page2Path := filepath.Join(target, "page", "2", "index.html")
	if _, err := os.Stat(page2Path); os.IsNotExist(err) {
		t.Error("page/2/index.html should exist for 11 posts")
	}
}
