package parser

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"obsite/internal/models"

	attributes "github.com/mdigger/goldmark-attributes"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

var frontmatterRegex = regexp.MustCompile(`(?s)^---\n(.+?)\n---\n(.*)$`)

// ParseFile reads a markdown file and returns a Post
func ParseFile(path string) (*models.Post, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	post, err := Parse(string(data))
	if err != nil {
		return nil, err
	}

	post.FilePath = path

	// Title defaults to filename (frontmatter title overrides this)
	if post.Title == "" {
		post.Title = titleFromFilename(path)
	}

	// If no slug specified, derive from filename
	if post.Slug == "" {
		post.Slug = slugFromFilename(path)
	}

	// Set the URL path
	post.URL = post.URLPath()

	return post, nil
}

// ParseBundle reads a page bundle (directory with index.md + assets)
// and returns a Post with BundleDir set to the directory path.
func ParseBundle(bundleDir string) (*models.Post, error) {
	indexPath := filepath.Join(bundleDir, "index.md")

	// Check if index.md exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("index.md not found in bundle: %s", bundleDir)
	}

	post, err := ParseFile(indexPath)
	if err != nil {
		return nil, err
	}

	// Mark this as a bundle and store the directory
	post.BundleDir = bundleDir

	// Always derive slug from directory name (overrides filename-based slug)
	post.Slug = slugFromFilename(bundleDir)

	// Update the URL path with the correct slug
	post.URL = post.URLPath()

	return post, nil
}

// IsBundleDir checks if a directory is a page bundle (contains index.md)
func IsBundleDir(path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}
	indexPath := filepath.Join(path, "index.md")
	if _, err := os.Stat(indexPath); err == nil {
		return true
	}
	return false
}

// titleFromFilename extracts the title from filename (without extension)
func titleFromFilename(path string) string {
	name := filepath.Base(path)
	return strings.TrimSuffix(name, filepath.Ext(name))
}

// Parse parses markdown content with YAML frontmatter.
// Note: This does NOT convert markdown to HTML. Call ConvertMarkdown() after
// resolving internal links.
func Parse(content string) (*models.Post, error) {
	post := &models.Post{}

	matches := frontmatterRegex.FindStringSubmatch(content)
	if matches == nil {
		// No frontmatter, treat entire content as markdown
		post.Content = content
	} else {
		// Parse YAML frontmatter
		if err := yaml.Unmarshal([]byte(matches[1]), post); err != nil {
			return nil, err
		}
		post.Content = strings.TrimSpace(matches[2])
	}

	return post, nil
}

// ConvertMarkdown converts the post's markdown Content to HTML.
// Call this after resolving internal links.
func ConvertMarkdown(post *models.Post) error {
	var buf bytes.Buffer
	md := goldmark.New(
		attributes.Enable,
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	if err := md.Convert([]byte(post.Content), &buf); err != nil {
		return err
	}
	post.HTML = template.HTML(buf.String())
	return nil
}

// slugFromFilename generates a URL-safe slug from a filename
func slugFromFilename(path string) string {
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return Slugify(name)
}

// Slugify converts a string to a URL-safe slug
func Slugify(s string) string {
	var result strings.Builder
	prevDash := false

	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
			prevDash = false
		} else if !prevDash && result.Len() > 0 {
			result.WriteRune('-')
			prevDash = true
		}
	}

	slug := result.String()
	return strings.TrimSuffix(slug, "-")
}
