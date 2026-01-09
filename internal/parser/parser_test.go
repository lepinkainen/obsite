package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"obsite/internal/models"
)

var updateGolden = os.Getenv("UPDATE_GOLDEN") == "true"

func TestParse_WithFrontmatter(t *testing.T) {
	input := `---
title: My Post
slug: my-post
created: 2024-01-15
tags:
  - blog
  - test
---
This is the content.`

	post, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if post.Title != "My Post" {
		t.Errorf("Title = %q, want %q", post.Title, "My Post")
	}
	if post.Slug != "my-post" {
		t.Errorf("Slug = %q, want %q", post.Slug, "my-post")
	}
	if post.Content != "This is the content." {
		t.Errorf("Content = %q, want %q", post.Content, "This is the content.")
	}
	if len(post.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(post.Tags))
	}
}

func TestParse_WithoutFrontmatter(t *testing.T) {
	input := `# Just Markdown

No frontmatter here, just content.`

	post, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if post.Title != "" {
		t.Errorf("Title = %q, want empty", post.Title)
	}
	if post.Content != input {
		t.Errorf("Content = %q, want %q", post.Content, input)
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	input := `---
title: [invalid yaml
slug: broken
---
Content here.`

	_, err := Parse(input)
	if err == nil {
		t.Error("Parse() expected error for invalid YAML, got nil")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple lowercase",
			input: "hello world",
			want:  "hello-world",
		},
		{
			name:  "mixed case",
			input: "Hello World",
			want:  "hello-world",
		},
		{
			name:  "punctuation",
			input: "Hello, World!",
			want:  "hello-world",
		},
		{
			name:  "multiple spaces",
			input: "hello    world",
			want:  "hello-world",
		},
		{
			name:  "leading separator",
			input: "  hello world",
			want:  "hello-world",
		},
		{
			name:  "trailing separator",
			input: "hello world  ",
			want:  "hello-world",
		},
		{
			name:  "unicode letters",
			input: "café résumé",
			want:  "café-résumé",
		},
		{
			name:  "digits",
			input: "post 123 title",
			want:  "post-123-title",
		},
		{
			name:  "special characters",
			input: "hello@world#2024",
			want:  "hello-world-2024",
		},
		{
			name:  "apostrophe",
			input: "it's a test",
			want:  "it-s-a-test",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only special chars",
			input: "!!!",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFile_Defaults(t *testing.T) {
	// Create a temp file with minimal frontmatter
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "My Test Post.md")

	content := `---
created: 2024-03-15
---
Some content here.`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	post, err := ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	// Title defaults to filename
	if post.Title != "My Test Post" {
		t.Errorf("Title = %q, want %q", post.Title, "My Test Post")
	}

	// Slug defaults to slugified filename
	if post.Slug != "my-test-post" {
		t.Errorf("Slug = %q, want %q", post.Slug, "my-test-post")
	}

	// URL should be set
	expectedURL := "/2024/03/my-test-post/"
	if post.URL != expectedURL {
		t.Errorf("URL = %q, want %q", post.URL, expectedURL)
	}

	// FilePath should be set
	if post.FilePath != testFile {
		t.Errorf("FilePath = %q, want %q", post.FilePath, testFile)
	}
}

func TestParseFile_FrontmatterOverridesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "filename.md")

	content := `---
title: Custom Title
slug: custom-slug
created: 2024-06-20
---
Content.`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	post, err := ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	// Frontmatter values should override defaults
	if post.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", post.Title, "Custom Title")
	}
	if post.Slug != "custom-slug" {
		t.Errorf("Slug = %q, want %q", post.Slug, "custom-slug")
	}
}

func TestConvertMarkdown_GoldenFiles(t *testing.T) {
	inputFiles, err := filepath.Glob("testdata/*.input.md")
	if err != nil {
		t.Fatalf("failed to glob input files: %v", err)
	}

	if len(inputFiles) == 0 {
		t.Fatal("no input files found in testdata/")
	}

	for _, inputFile := range inputFiles {
		name := strings.TrimSuffix(filepath.Base(inputFile), ".input.md")

		t.Run(name, func(t *testing.T) {
			input, err := os.ReadFile(inputFile)
			if err != nil {
				t.Fatalf("failed to read input file: %v", err)
			}

			post := &models.Post{
				Content: string(input),
			}

			if err := ConvertMarkdown(post); err != nil {
				t.Fatalf("ConvertMarkdown() error = %v", err)
			}

			got := string(post.HTML)
			goldenFile := strings.Replace(inputFile, ".input.md", ".golden.html", 1)

			if updateGolden {
				if err := os.WriteFile(goldenFile, []byte(got), 0644); err != nil {
					t.Fatalf("failed to update golden file: %v", err)
				}
				return
			}

			expected, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("failed to read golden file %s: %v", goldenFile, err)
			}

			if got != string(expected) {
				t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", got, expected)
			}
		})
	}
}

func TestParseBundle(t *testing.T) {
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "my-bundle")

	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}

	// Create index.md
	indexContent := `---
title: Bundle Post
created: 2024-01-15
categories:
  - test
summary: A test bundle
---
Content with ![image](cover.jpg)`

	indexPath := filepath.Join(bundleDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		t.Fatalf("failed to create index.md: %v", err)
	}

	// Create an image file
	imagePath := filepath.Join(bundleDir, "cover.jpg")
	if err := os.WriteFile(imagePath, []byte("fake image"), 0644); err != nil {
		t.Fatalf("failed to create image: %v", err)
	}

	post, err := ParseBundle(bundleDir)
	if err != nil {
		t.Fatalf("ParseBundle() error = %v", err)
	}

	if post.Title != "Bundle Post" {
		t.Errorf("Title = %q, want %q", post.Title, "Bundle Post")
	}

	if post.BundleDir != bundleDir {
		t.Errorf("BundleDir = %q, want %q", post.BundleDir, bundleDir)
	}

	if post.Slug != "my-bundle" {
		t.Errorf("Slug = %q, want %q", post.Slug, "my-bundle")
	}

	if post.Summary != "A test bundle" {
		t.Errorf("Summary = %q, want %q", post.Summary, "A test bundle")
	}

	if len(post.Categories) != 1 || post.Categories[0] != "test" {
		t.Errorf("Categories = %v, want [test]", post.Categories)
	}
}

func TestParseBundle_MissingIndexMd(t *testing.T) {
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "no-index")

	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}

	_, err := ParseBundle(bundleDir)
	if err == nil {
		t.Error("ParseBundle() expected error for missing index.md, got nil")
	}
}

func TestIsBundleDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a bundle directory
	bundleDir := filepath.Join(tmpDir, "bundle")
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "index.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write index.md: %v", err)
	}

	// Create a non-bundle directory
	nonBundleDir := filepath.Join(tmpDir, "not-bundle")
	if err := os.MkdirAll(nonBundleDir, 0755); err != nil {
		t.Fatalf("failed to create non-bundle dir: %v", err)
	}

	bundleInfo, err := os.Stat(bundleDir)
	if err != nil {
		t.Fatalf("failed to stat bundle dir: %v", err)
	}

	if !IsBundleDir(bundleDir, bundleInfo) {
		t.Error("IsBundleDir() = false for bundle directory, want true")
	}

	nonBundleInfo, err := os.Stat(nonBundleDir)
	if err != nil {
		t.Fatalf("failed to stat non-bundle dir: %v", err)
	}

	if IsBundleDir(nonBundleDir, nonBundleInfo) {
		t.Error("IsBundleDir() = true for non-bundle directory, want false")
	}

	// Test with a file instead of directory
	fileInfo, err := os.Stat(filepath.Join(bundleDir, "index.md"))
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	if IsBundleDir(filepath.Join(bundleDir, "index.md"), fileInfo) {
		t.Error("IsBundleDir() = true for file, want false")
	}
}
