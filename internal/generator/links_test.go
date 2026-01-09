package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"obsite/internal/models"
)

var updateGolden = os.Getenv("UPDATE_GOLDEN") == "true"

// fixedTime creates a FlexibleTime for testing
func fixedTime(year int, month time.Month, day int) models.FlexibleTime {
	return models.FlexibleTime{Time: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// testPosts returns a set of fake posts for link resolution testing
func testPosts() []*models.Post {
	return []*models.Post{
		{
			Slug:    "target-post",
			Created: fixedTime(2024, 1, 15),
		},
		{
			Slug:    "another-post",
			Created: fixedTime(2024, 2, 20),
		},
	}
}

func TestLinkResolver_Resolve(t *testing.T) {
	resolver := NewLinkResolver(testPosts())

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
				Content:  string(input),
				FilePath: "test.md",
			}
			got, linkErrors := resolver.Resolve(post)

			// Handle broken link test case
			if strings.HasPrefix(name, "broken") {
				if len(linkErrors) == 0 {
					t.Error("expected link errors for broken link test, got none")
				}
				return
			}

			// For non-broken tests, expect no errors
			if len(linkErrors) > 0 {
				t.Errorf("unexpected link errors: %v", linkErrors)
			}

			goldenFile := strings.Replace(inputFile, ".input.md", ".golden.md", 1)

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

func TestLinkResolver_BrokenLinkError(t *testing.T) {
	resolver := NewLinkResolver(testPosts())

	post := &models.Post{
		Content:  "Link to [[Missing Page]] here",
		FilePath: "source.md",
	}

	_, errors := resolver.Resolve(post)

	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}

	err := errors[0]
	if err.TargetSlug != "missing-page" {
		t.Errorf("expected target slug 'missing-page', got %q", err.TargetSlug)
	}
	if err.LinkText != "[[Missing Page]]" {
		t.Errorf("expected link text '[[Missing Page]]', got %q", err.LinkText)
	}
}

func TestResolve_ObsidianImageWithResize(t *testing.T) {
	resolver := NewLinkResolver(testPosts())

	post := &models.Post{
		Content:   "Check this: ![[photo.jpg|300]] and ![[banner.png|150]]",
		FilePath:  "test.md",
		BundleDir: "/fake/bundle",
		Slug:      "test-post",
		Created:   fixedTime(2024, 1, 15),
	}

	content, errors := resolver.Resolve(post)
	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	if post.ImageResizes == nil {
		t.Fatal("ImageResizes map should be initialized")
	}

	if post.ImageResizes["photo.jpg"] != 300 {
		t.Errorf("expected photo.jpg resize to 300, got %d", post.ImageResizes["photo.jpg"])
	}

	if post.ImageResizes["banner.png"] != 150 {
		t.Errorf("expected banner.png resize to 150, got %d", post.ImageResizes["banner.png"])
	}

	if !strings.Contains(content, `width="300"`) {
		t.Errorf("expected HTML with width=300, got: %s", content)
	}

	if !strings.Contains(content, `width="150"`) {
		t.Errorf("expected HTML with width=150, got: %s", content)
	}
}

func TestResolve_ObsidianImageWithoutResize(t *testing.T) {
	resolver := NewLinkResolver(testPosts())

	post := &models.Post{
		Content:   "Normal image: ![[photo.jpg]]",
		FilePath:  "test.md",
		BundleDir: "/fake/bundle",
		Slug:      "test-post",
		Created:   fixedTime(2024, 1, 15),
	}

	content, errors := resolver.Resolve(post)
	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	if len(post.ImageResizes) > 0 {
		t.Errorf("ImageResizes should be empty for images without resize, got %v", post.ImageResizes)
	}

	if !strings.Contains(content, "![](") {
		t.Errorf("expected markdown image syntax, got: %s", content)
	}
}

func TestLinkError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      LinkError
		wantPart []string
	}{
		{
			name: "formats source file basename",
			err: LinkError{
				SourceFile: "/path/to/my-post.md",
				LinkText:   "[[Missing]]",
				TargetSlug: "missing",
			},
			wantPart: []string{"my-post.md", "[[Missing]]", "missing"},
		},
		{
			name: "includes all fields",
			err: LinkError{
				SourceFile: "test.md",
				LinkText:   "[link](broken.md)",
				TargetSlug: "broken",
			},
			wantPart: []string{"test.md", "[link](broken.md)", "broken"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			for _, part := range tt.wantPart {
				if !strings.Contains(got, part) {
					t.Errorf("Error() = %q, want to contain %q", got, part)
				}
			}
		})
	}
}

func TestResolve_ObsidianImageWithInvalidWidth(t *testing.T) {
	resolver := NewLinkResolver(testPosts())

	tests := []struct {
		name    string
		content string
		wantImg string
	}{
		{
			name:    "zero width",
			content: "Image: ![[photo.jpg|0]]",
			wantImg: "![](/2024/01/test-post/photo.jpg)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := &models.Post{
				Content:   tt.content,
				FilePath:  "test.md",
				BundleDir: "/fake/bundle",
				Slug:      "test-post",
				Created:   fixedTime(2024, 1, 15),
			}

			content, errors := resolver.Resolve(post)
			if len(errors) > 0 {
				t.Errorf("unexpected link errors: %v", errors)
			}

			if !strings.Contains(content, tt.wantImg) {
				t.Errorf("expected %q in output, got: %s", tt.wantImg, content)
			}

			// Invalid widths should not create resize entries
			if len(post.ImageResizes) > 0 {
				t.Errorf("ImageResizes should be empty for invalid widths, got %v", post.ImageResizes)
			}
		})
	}
}
