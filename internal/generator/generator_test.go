package generator

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"image"
	"image/png"
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

func TestGenerateStylesheet(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "output")

	gen := &Generator{
		Target:     target,
		templateFS: testTemplateFS,
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	if err := gen.generateStylesheet(); err != nil {
		t.Fatalf("generateStylesheet() error = %v", err)
	}

	// Check that styles.css was created
	cssPath := filepath.Join(target, "styles.css")
	content, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("styles.css was not created: %v", err)
	}

	// Verify it contains CSS content (test stub has "body")
	if !strings.Contains(string(content), "body") {
		t.Error("styles.css should contain CSS content")
	}
}

func TestGenerateThemeScript(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "output")

	gen := &Generator{
		Target:     target,
		templateFS: testTemplateFS,
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	if err := gen.generateThemeScript(); err != nil {
		t.Fatalf("generateThemeScript() error = %v", err)
	}

	// Check that theme.js was created
	jsPath := filepath.Join(target, "theme.js")
	content, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("theme.js was not created: %v", err)
	}

	// Verify it contains JavaScript content (test stub has "console")
	if !strings.Contains(string(content), "console") {
		t.Error("theme.js should contain JavaScript content")
	}
}

func setupTestSource(t *testing.T, files map[string]string) string {
	t.Helper()
	sourceDir := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		path := filepath.Join(sourceDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return sourceDir
}

func TestCollectPosts_SkipDrafts(t *testing.T) {
	source := setupTestSource(t, map[string]string{
		"published.md": `---
title: Published Post
created: 2024-01-15
---
Content`,
		"draft-field.md": `---
title: Draft via Field
created: 2024-01-16
draft: true
---
Content`,
		"draft-tag.md": `---
title: Draft via Tag
created: 2024-01-17
tags:
  - draft
---
Content`,
	})

	gen := &Generator{
		Source:        source,
		IncludeDrafts: false,
	}

	if err := gen.collectPosts(); err != nil {
		t.Fatalf("collectPosts() error = %v", err)
	}

	if len(gen.Posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(gen.Posts))
	}
	if gen.Posts[0].Title != "Published Post" {
		t.Errorf("expected 'Published Post', got %q", gen.Posts[0].Title)
	}
}

func TestCollectPosts_IncludeDrafts(t *testing.T) {
	source := setupTestSource(t, map[string]string{
		"published.md": `---
title: Published Post
created: 2024-01-15
---
Content`,
		"draft.md": `---
title: Draft Post
created: 2024-01-16
draft: true
---
Content`,
	})

	gen := &Generator{
		Source:        source,
		IncludeDrafts: true,
	}

	if err := gen.collectPosts(); err != nil {
		t.Fatalf("collectPosts() error = %v", err)
	}

	if len(gen.Posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(gen.Posts))
	}
}

func TestCollectPosts_TitleDefaultsToFilename(t *testing.T) {
	source := setupTestSource(t, map[string]string{
		"My Great Post.md": `---
created: 2024-01-15
---
Content without explicit title`,
	})

	gen := &Generator{Source: source}

	if err := gen.collectPosts(); err != nil {
		t.Fatalf("collectPosts() error = %v", err)
	}

	if len(gen.Posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(gen.Posts))
	}
	if gen.Posts[0].Title != "My Great Post" {
		t.Errorf("expected title 'My Great Post', got %q", gen.Posts[0].Title)
	}
}

func TestCollectPosts_SkipMissingCreated(t *testing.T) {
	source := setupTestSource(t, map[string]string{
		"no-date.md": `---
title: No Date Post
---
Content without date`,
		"has-date.md": `---
title: Has Date
created: 2024-01-16
---
Content`,
	})

	gen := &Generator{Source: source}

	if err := gen.collectPosts(); err != nil {
		t.Fatalf("collectPosts() error = %v", err)
	}

	if len(gen.Posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(gen.Posts))
	}
}

func TestProcessContent_BrokenLinksError(t *testing.T) {
	gen := &Generator{
		Posts: []*models.Post{
			{
				Title:    "Post A",
				Slug:     "post-a",
				Created:  models.FlexibleTime{Time: time.Now()},
				Content:  "Link to [[Missing Post]] and [[Another Missing]]",
				FilePath: "post-a.md",
			},
		},
	}

	err := gen.processContent()
	if err == nil {
		t.Fatal("expected error for broken links")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "missing-post") {
		t.Error("error should mention 'missing-post'")
	}
	if !strings.Contains(errStr, "another-missing") {
		t.Error("error should mention 'another-missing'")
	}
}

func TestGenerateFeed(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "output")

	posts := make([]*models.Post, 25)
	for i := 0; i < 25; i++ {
		posts[i] = createTestPost("post-"+string(rune('a'+i)), i)
	}

	gen := &Generator{
		Target:     target,
		Posts:      posts,
		templateFS: testTemplateFS,
	}

	var err error
	gen.templates, err = gen.templates.ParseFS(testTemplateFS, "testdata/templates/*.html", "testdata/templates/*.xml")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}

	if err := gen.generateFeed(); err != nil {
		t.Fatalf("generateFeed() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(target, "feed.xml"))
	if err != nil {
		t.Fatal(err)
	}

	feedStr := string(content)

	if !strings.Contains(feedStr, "<rss") {
		t.Error("feed should contain RSS element")
	}

	itemCount := strings.Count(feedStr, "<item>")
	if itemCount > 20 {
		t.Errorf("feed should be limited to 20 posts, got %d", itemCount)
	}
}

func TestGenerateSitemap(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "output")

	posts := []*models.Post{
		createTestPost("post-a", 1),
		createTestPost("post-b", 2),
	}

	gen := &Generator{
		Target:     target,
		Posts:      posts,
		templateFS: testTemplateFS,
	}

	var err error
	gen.templates, err = gen.templates.ParseFS(testTemplateFS, "testdata/templates/*.html", "testdata/templates/*.xml")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}

	if err := gen.generateSitemap(); err != nil {
		t.Fatalf("generateSitemap() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(target, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}

	sitemapStr := string(content)

	if !strings.Contains(sitemapStr, "<urlset") {
		t.Error("sitemap should contain urlset element")
	}

	urlCount := strings.Count(sitemapStr, "<url>")
	if urlCount != 3 {
		t.Errorf("sitemap should have 3 URLs (1 home + 2 posts), got %d", urlCount)
	}
}

func TestGenerate_EndToEnd(t *testing.T) {
	source := setupTestSource(t, map[string]string{
		"post1.md": `---
title: First Post
created: 2024-01-15
---
Hello world`,
		"post2.md": `---
title: Second Post
created: 2024-01-16
---
Another post`,
		"image.png": "fake image data",
	})

	target := filepath.Join(t.TempDir(), "output")

	gen := &Generator{
		Source:     source,
		Target:     target,
		templateFS: testTemplateFS,
	}

	// Pre-load templates from test FS (mimic what loadTemplates does but with testdata path)
	var err error
	gen.templates, err = gen.templates.ParseFS(testTemplateFS, "testdata/templates/*.html", "testdata/templates/*.xml")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	// Run the partial Generate flow (skip loadTemplates since we already loaded)
	if err := gen.collectPosts(); err != nil {
		t.Fatalf("collectPosts() error = %v", err)
	}
	if err := gen.processContent(); err != nil {
		t.Fatalf("processContent() error = %v", err)
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}

	if err := gen.generateStylesheet(); err != nil {
		t.Fatalf("generateStylesheet() error = %v", err)
	}
	if err := gen.generateThemeScript(); err != nil {
		t.Fatalf("generateThemeScript() error = %v", err)
	}

	for _, post := range gen.Posts {
		if err := gen.generatePost(post); err != nil {
			t.Fatalf("generatePost() error = %v", err)
		}
	}

	if err := gen.generateIndex(); err != nil {
		t.Fatalf("generateIndex() error = %v", err)
	}
	if err := gen.generateFeed(); err != nil {
		t.Fatalf("generateFeed() error = %v", err)
	}
	if err := gen.generateSitemap(); err != nil {
		t.Fatalf("generateSitemap() error = %v", err)
	}
	if err := gen.copyAssets(); err != nil {
		t.Fatalf("copyAssets() error = %v", err)
	}

	checks := []string{
		"index.html",
		"feed.xml",
		"sitemap.xml",
		"styles.css",
		"theme.js",
	}
	for _, file := range checks {
		if _, err := os.Stat(filepath.Join(target, file)); os.IsNotExist(err) {
			t.Errorf("%s was not generated", file)
		}
	}

	if _, err := os.Stat(filepath.Join(target, "assets", "image.png")); os.IsNotExist(err) {
		t.Error("assets/image.png was not copied")
	}

	if len(gen.Posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(gen.Posts))
	}
}

func TestGenerateIndex_Pagination_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		postCount    int
		wantPages    int
		page1HasNext bool
		page1HasPrev bool
		lastHasNext  bool
		lastHasPrev  bool
	}{
		{
			name:         "zero posts",
			postCount:    0,
			wantPages:    1,
			page1HasNext: false,
			page1HasPrev: false,
		},
		{
			name:         "exactly 10 posts",
			postCount:    10,
			wantPages:    1,
			page1HasNext: false,
			page1HasPrev: false,
		},
		{
			name:         "11 posts",
			postCount:    11,
			wantPages:    2,
			page1HasNext: true,
			page1HasPrev: false,
			lastHasNext:  false,
			lastHasPrev:  true,
		},
		{
			name:         "25 posts",
			postCount:    25,
			wantPages:    3,
			page1HasNext: true,
			page1HasPrev: false,
			lastHasNext:  false,
			lastHasPrev:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			target := filepath.Join(tmpDir, "output")

			posts := make([]*models.Post, tt.postCount)
			for i := 0; i < tt.postCount; i++ {
				posts[i] = createTestPost("post-"+string(rune('a'+i%26)), i)
			}

			gen := &Generator{Target: target, Posts: posts}

			var err error
			gen.templates, err = gen.templates.ParseFS(testTemplateFS, "testdata/templates/*.html")
			if err != nil {
				t.Fatalf("failed to parse templates: %v", err)
			}

			if err := os.MkdirAll(target, 0755); err != nil {
				t.Fatal(err)
			}

			if err := gen.generateIndex(); err != nil {
				t.Fatalf("generateIndex() error = %v", err)
			}

			indexContent, err := os.ReadFile(filepath.Join(target, "index.html"))
			if err != nil {
				t.Fatal(err)
			}

			hasNext := strings.Contains(string(indexContent), "Older Posts")
			hasPrev := strings.Contains(string(indexContent), "Newer Posts")

			if hasNext != tt.page1HasNext {
				t.Errorf("page 1 next link: got %v, want %v", hasNext, tt.page1HasNext)
			}
			if hasPrev != tt.page1HasPrev {
				t.Errorf("page 1 prev link: got %v, want %v", hasPrev, tt.page1HasPrev)
			}

			if tt.wantPages > 1 {
				lastPagePath := filepath.Join(target, "page", fmt.Sprintf("%d", tt.wantPages), "index.html")
				lastContent, err := os.ReadFile(lastPagePath)
				if err != nil {
					t.Fatalf("failed to read last page: %v", err)
				}

				lastHasNext := strings.Contains(string(lastContent), "Older Posts")
				lastHasPrev := strings.Contains(string(lastContent), "Newer Posts")

				if lastHasNext != tt.lastHasNext {
					t.Errorf("last page next link: got %v, want %v", lastHasNext, tt.lastHasNext)
				}
				if lastHasPrev != tt.lastHasPrev {
					t.Errorf("last page prev link: got %v, want %v", lastHasPrev, tt.lastHasPrev)
				}
			}

			nonExistentPage := filepath.Join(target, "page", fmt.Sprintf("%d", tt.wantPages+1), "index.html")
			if _, err := os.Stat(nonExistentPage); !os.IsNotExist(err) {
				t.Errorf("page %d should not exist", tt.wantPages+1)
			}
		})
	}
}

func TestCollectPosts_WithBundle(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")

	// Create a page bundle
	bundleDir := filepath.Join(sourceDir, "test-bundle")
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}

	indexContent := `---
title: Bundle Post
created: 2024-01-15
---
Bundle content`

	if err := os.WriteFile(filepath.Join(bundleDir, "index.md"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("failed to create index.md: %v", err)
	}

	if err := os.WriteFile(filepath.Join(bundleDir, "cover.jpg"), []byte("fake image"), 0644); err != nil {
		t.Fatalf("failed to create image: %v", err)
	}

	// Create a single post
	singleContent := `---
title: Single Post
created: 2024-01-16
---
Single content`

	if err := os.WriteFile(filepath.Join(sourceDir, "single.md"), []byte(singleContent), 0644); err != nil {
		t.Fatalf("failed to create single post: %v", err)
	}

	gen := New(sourceDir, targetDir, testTemplateFS)

	if err := gen.collectPosts(); err != nil {
		t.Fatalf("collectPosts() error = %v", err)
	}

	if len(gen.Posts) != 2 {
		t.Errorf("collectPosts() got %d posts, want 2", len(gen.Posts))
	}

	// Find bundle post
	var bundlePost *models.Post
	var singlePost *models.Post
	for _, p := range gen.Posts {
		switch p.Title {
		case "Bundle Post":
			bundlePost = p
		case "Single Post":
			singlePost = p
		}
	}

	if bundlePost == nil {
		t.Fatal("bundle post not found")
	}
	if singlePost == nil {
		t.Fatal("single post not found")
	}

	if bundlePost.BundleDir != bundleDir {
		t.Errorf("bundle post BundleDir = %q, want %q", bundlePost.BundleDir, bundleDir)
	}

	if singlePost.BundleDir != "" {
		t.Errorf("single post BundleDir = %q, want empty", singlePost.BundleDir)
	}
}

func TestGeneratePost_WithBundle(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")
	bundleDir := filepath.Join(sourceDir, "test-bundle")

	// Create bundle structure
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}

	indexContent := `---
title: Bundle Test
created: 2024-01-15
---
Post content`

	if err := os.WriteFile(filepath.Join(bundleDir, "index.md"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("failed to create index.md: %v", err)
	}

	if err := os.WriteFile(filepath.Join(bundleDir, "cover.jpg"), []byte("fake image data"), 0644); err != nil {
		t.Fatalf("failed to create image: %v", err)
	}

	post := &models.Post{
		Title:     "Bundle Test",
		Slug:      "test-bundle",
		Created:   models.FlexibleTime{Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
		Content:   "Post content",
		HTML:      template.HTML("<p>Post content</p>"),
		BundleDir: bundleDir,
	}
	post.URL = post.URLPath()

	gen := New(sourceDir, targetDir, testTemplateFS)

	var err error
	gen.templates, err = gen.templates.ParseFS(testTemplateFS, "testdata/templates/*.html", "testdata/templates/*.xml")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	if err := gen.generatePost(post); err != nil {
		t.Fatalf("generatePost() error = %v", err)
	}

	// Check HTML file exists
	htmlPath := filepath.Join(targetDir, "2024/01/test-bundle/index.html")
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		t.Error("index.html was not created")
	}

	// Check image was copied
	imagePath := filepath.Join(targetDir, "2024/01/test-bundle/cover.jpg")
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		t.Error("cover.jpg was not copied to output directory")
	}
}

func TestCopyBundleAssets_WithResize(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")
	bundleDir := filepath.Join(sourceDir, "resize-bundle")
	outDir := filepath.Join(targetDir, "2024/01/resize-bundle")

	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	pngData := createTestPNG(t, 400, 300)
	if err := os.WriteFile(filepath.Join(bundleDir, "test.png"), pngData, 0644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	post := &models.Post{
		Title:        "Resize Test",
		Slug:         "resize-bundle",
		Created:      models.FlexibleTime{Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
		BundleDir:    bundleDir,
		ImageResizes: map[string]int{"test.png": 200},
	}

	gen := New(sourceDir, targetDir, testTemplateFS)

	if err := gen.copyBundleAssets(post, outDir); err != nil {
		t.Fatalf("copyBundleAssets() error = %v", err)
	}

	dstPath := filepath.Join(outDir, "test.png")
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Fatal("test.png was not created in output directory")
	}
}

func TestCopyBundleAssets_NoUpscale(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")
	bundleDir := filepath.Join(sourceDir, "noupscale-bundle")
	outDir := filepath.Join(targetDir, "2024/01/noupscale-bundle")

	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	pngData := createTestPNG(t, 100, 75)
	if err := os.WriteFile(filepath.Join(bundleDir, "small.png"), pngData, 0644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	post := &models.Post{
		Title:        "No Upscale Test",
		Slug:         "noupscale-bundle",
		Created:      models.FlexibleTime{Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
		BundleDir:    bundleDir,
		ImageResizes: map[string]int{"small.png": 500},
	}

	gen := New(sourceDir, targetDir, testTemplateFS)

	if err := gen.copyBundleAssets(post, outDir); err != nil {
		t.Fatalf("copyBundleAssets() error = %v", err)
	}

	dstPath := filepath.Join(outDir, "small.png")
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Fatal("small.png was not created in output directory")
	}
}

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"photo.jpg", true},
		{"photo.jpeg", true},
		{"photo.JPG", true},
		{"image.png", true},
		{"animation.gif", true},
		{"modern.webp", true},
		{"document.pdf", false},
		{"video.mp4", false},
		{"readme.txt", false},
		{"noext", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := isImageFile(tt.filename); got != tt.want {
				t.Errorf("isImageFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func createTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, image.White)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}
	return buf.Bytes()
}
