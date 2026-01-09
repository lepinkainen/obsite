package generator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"obsite/internal/models"
	"obsite/internal/parser"
)

var (
	// ![[image.jpg]] or ![[image.jpg|250]] - Obsidian wiki-style image syntax with optional width
	obsidianImageRegex = regexp.MustCompile(`!\[\[([^\]|]+)(?:\|(\d+))?\]\]`)
	// ![alt](image.jpg) or ![alt](path/to/image.jpg) - markdown images
	mdImageRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	// [[Page Name]] or [[Page Name|Display Text]]
	wikiLinkRegex = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)
	// [text](path/to/file.md) or [text](./file.md)
	mdLinkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+\.md)\)`)
)

// LinkError represents a broken internal link
type LinkError struct {
	SourceFile string
	LinkText   string
	TargetSlug string
}

func (e LinkError) Error() string {
	return fmt.Sprintf("%s: %s (no post with slug %q)", filepath.Base(e.SourceFile), e.LinkText, e.TargetSlug)
}

// LinkResolver resolves internal links in post content
type LinkResolver struct {
	slugMap map[string]*models.Post
}

// NewLinkResolver creates a resolver from a slice of posts
func NewLinkResolver(posts []*models.Post) *LinkResolver {
	slugMap := make(map[string]*models.Post, len(posts))
	for _, p := range posts {
		slugMap[p.Slug] = p
	}
	return &LinkResolver{slugMap: slugMap}
}

// Resolve processes a post's content and resolves all internal links.
// Returns the modified content and any link errors found.
func (r *LinkResolver) Resolve(post *models.Post) (string, []LinkError) {
	content := post.Content
	var errors []LinkError

	// Convert Obsidian image syntax to markdown: ![[image.jpg]] -> ![](image.jpg)
	// With optional resize: ![[image.jpg|250]] -> stores width and outputs HTML img tag
	// This must be done before processing wiki links to avoid treating images as links
	content = obsidianImageRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := obsidianImageRegex.FindStringSubmatch(match)
		imagePath := parts[1]
		widthStr := parts[2]

		if widthStr != "" {
			width, err := strconv.Atoi(widthStr)
			if err != nil {
				fmt.Printf("[WARN] Invalid image resize width %q in %s: %v\n", widthStr, imagePath, err)
			} else if width <= 0 {
				fmt.Printf("[WARN] Invalid image resize width %q in %s: must be positive\n", widthStr, imagePath)
			} else {
				if post.ImageResizes == nil {
					post.ImageResizes = make(map[string]int)
				}
				post.ImageResizes[imagePath] = width
				return fmt.Sprintf("![%s](%s)", widthStr, imagePath)
			}
		}
		return fmt.Sprintf("![](%s)", imagePath)
	})

	// Resolve relative image paths for page bundles
	// For bundles: ![alt](cover.jpg) -> ![alt](/2025/08/post-slug/cover.jpg)
	// For resized images: output HTML img tag with width attribute
	if post.BundleDir != "" {
		content = mdImageRegex.ReplaceAllStringFunc(content, func(match string) string {
			parts := mdImageRegex.FindStringSubmatch(match)
			altText := parts[1]
			imagePath := parts[2]

			// Only convert relative paths (those not starting with / or http)
			if !strings.HasPrefix(imagePath, "/") && !strings.HasPrefix(imagePath, "http") {
				absolutePath := fmt.Sprintf("%s%s", post.URLPath(), imagePath)

				if width, hasResize := post.ImageResizes[imagePath]; hasResize {
					return fmt.Sprintf(`<img src="%s" alt="%s" width="%d">`, absolutePath, altText, width)
				}
				return fmt.Sprintf("![%s](%s)", altText, absolutePath)
			}
			return match
		})
	}

	// Process wiki links: [[Page Name]] -> [Page Name](/year/month/slug/)
	content = wikiLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := wikiLinkRegex.FindStringSubmatch(match)
		pageName := parts[1]
		displayText := parts[2]
		if displayText == "" {
			displayText = pageName
		}

		slug := parser.Slugify(pageName)
		target, ok := r.slugMap[slug]
		if !ok {
			errors = append(errors, LinkError{
				SourceFile: post.FilePath,
				LinkText:   match,
				TargetSlug: slug,
			})
			return match // Keep original if not found
		}

		return fmt.Sprintf("[%s](%s)", displayText, target.URLPath())
	})

	// Process markdown links to .md files: [text](file.md) -> [text](/year/month/slug/)
	content = mdLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := mdLinkRegex.FindStringSubmatch(match)
		linkText := parts[1]
		linkPath := parts[2]

		// Extract filename without path and extension
		filename := filepath.Base(linkPath)
		filename = strings.TrimSuffix(filename, ".md")
		slug := parser.Slugify(filename)

		target, ok := r.slugMap[slug]
		if !ok {
			errors = append(errors, LinkError{
				SourceFile: post.FilePath,
				LinkText:   match,
				TargetSlug: slug,
			})
			return match // Keep original if not found
		}

		return fmt.Sprintf("[%s](%s)", linkText, target.URLPath())
	})

	return content, errors
}
