package generator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"obsite/internal/models"
	"obsite/internal/parser"
)

var (
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
