package parser

import (
	"os"
	"path/filepath"
	"testing"
)

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
