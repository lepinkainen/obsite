package generator

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/disintegration/gift"

	"obsite/internal/models"
	"obsite/internal/parser"
)

const postsPerPage = 10

type Generator struct {
	Source        string
	Target        string
	IncludeDrafts bool
	Posts         []*models.Post
	templates     *template.Template
	templateFS    embed.FS
}

func New(source, target string, templateFS embed.FS) *Generator {
	return &Generator{
		Source:     source,
		Target:     target,
		templateFS: templateFS,
	}
}

func (g *Generator) Generate() error {
	// Reset state for clean regeneration
	g.Posts = nil

	// Load templates
	if err := g.loadTemplates(); err != nil {
		return fmt.Errorf("loading templates: %w", err)
	}

	// Collect all posts (parses frontmatter, keeps markdown content)
	if err := g.collectPosts(); err != nil {
		return fmt.Errorf("collecting posts: %w", err)
	}

	// Resolve internal links and convert markdown to HTML
	if err := g.processContent(); err != nil {
		return err
	}

	// Sort posts by date (newest first)
	sort.Slice(g.Posts, func(i, j int) bool {
		return g.Posts[i].Created.After(g.Posts[j].Created.Time)
	})

	// Clean and create target directory
	if err := os.RemoveAll(g.Target); err != nil {
		return fmt.Errorf("cleaning target: %w", err)
	}
	if err := os.MkdirAll(g.Target, 0755); err != nil {
		return fmt.Errorf("creating target: %w", err)
	}

	// Generate stylesheet and theme script
	if err := g.generateStylesheet(); err != nil {
		return fmt.Errorf("generating stylesheet: %w", err)
	}
	if err := g.generateThemeScript(); err != nil {
		return fmt.Errorf("generating theme script: %w", err)
	}

	// Generate post pages
	for _, post := range g.Posts {
		if err := g.generatePost(post); err != nil {
			return fmt.Errorf("generating post %s: %w", post.Slug, err)
		}
	}

	// Generate index pages
	if err := g.generateIndex(); err != nil {
		return fmt.Errorf("generating index: %w", err)
	}

	// Generate RSS feed
	if err := g.generateFeed(); err != nil {
		return fmt.Errorf("generating feed: %w", err)
	}

	// Generate sitemap
	if err := g.generateSitemap(); err != nil {
		return fmt.Errorf("generating sitemap: %w", err)
	}

	// Copy assets
	if err := g.copyAssets(); err != nil {
		return fmt.Errorf("copying assets: %w", err)
	}

	fmt.Printf("Generated %d posts\n", len(g.Posts))
	return nil
}

func (g *Generator) loadTemplates() error {
	var err error
	g.templates, err = template.ParseFS(g.templateFS, "templates/*.html", "templates/*.xml")
	return err
}

func (g *Generator) collectPosts() error {
	return filepath.Walk(g.Source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if this is a page bundle directory
		if parser.IsBundleDir(path, info) {
			post, err := parser.ParseBundle(path)
			if err != nil {
				fmt.Printf("Warning: skipping bundle %s: %v\n", path, err)
				return filepath.SkipDir
			}

			// Validate and add post
			if err := g.validateAndAddPost(post, path); err != nil {
				fmt.Printf("Warning: skipping bundle %s: %v\n", path, err)
			}

			return filepath.SkipDir // Don't recurse into bundle directories
		}

		// Skip directories (non-bundles)
		if info.IsDir() {
			return nil
		}

		// Handle single .md files (skip if named index.md - already handled as bundle)
		if !strings.HasSuffix(path, ".md") || filepath.Base(path) == "index.md" {
			return nil
		}

		post, err := parser.ParseFile(path)
		if err != nil {
			fmt.Printf("Warning: skipping %s: %v\n", path, err)
			return nil
		}

		// Validate and add post
		if err := g.validateAndAddPost(post, path); err != nil {
			fmt.Printf("Warning: skipping %s: %v\n", path, err)
		}

		return nil
	})
}

// validateAndAddPost checks required fields and adds post to collection
func (g *Generator) validateAndAddPost(post *models.Post, _ string) error {
	// Skip drafts unless IncludeDrafts is set
	if post.IsDraft() && !g.IncludeDrafts {
		fmt.Printf("Skipping draft: %s\n", post.Title)
		return fmt.Errorf("draft post")
	}

	// Skip posts without required fields
	if post.Title == "" {
		return fmt.Errorf("no title")
	}
	if post.Created.IsZero() {
		return fmt.Errorf("no created date")
	}

	g.Posts = append(g.Posts, post)
	return nil
}

// processContent resolves internal links and converts markdown to HTML
func (g *Generator) processContent() error {
	resolver := NewLinkResolver(g.Posts)
	var allErrors []LinkError

	// Resolve links in all posts
	for _, post := range g.Posts {
		resolved, errors := resolver.Resolve(post)
		post.Content = resolved
		allErrors = append(allErrors, errors...)
	}

	// Fail build if there are broken links
	if len(allErrors) > 0 {
		var sb strings.Builder
		sb.WriteString("broken internal links found:\n")
		for _, e := range allErrors {
			sb.WriteString("  - ")
			sb.WriteString(e.Error())
			sb.WriteString("\n")
		}
		return errors.New(sb.String())
	}

	// Convert markdown to HTML
	for _, post := range g.Posts {
		if err := parser.ConvertMarkdown(post); err != nil {
			return fmt.Errorf("converting markdown for %s: %w", post.Slug, err)
		}
	}

	return nil
}

func (g *Generator) generatePost(post *models.Post) error {
	// Create output directory
	outDir := filepath.Join(g.Target, post.URLPath())
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Copy bundle assets if this is a page bundle
	if post.BundleDir != "" {
		if err := g.copyBundleAssets(post, outDir); err != nil {
			return fmt.Errorf("copying bundle assets: %w", err)
		}
	}

	// Create output file
	outPath := filepath.Join(outDir, "index.html")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}

	data := struct {
		Site *models.Site
		Post *models.Post
	}{
		Site: &models.SiteConfig,
		Post: post,
	}

	if err := g.templates.ExecuteTemplate(f, "post.html", data); err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return fmt.Errorf("rendering %s: %w (close error: %v)", outPath, err, closeErr)
		}
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return nil
}

func (g *Generator) generateIndex() error {
	totalPages := (len(g.Posts) + postsPerPage - 1) / postsPerPage
	if totalPages == 0 {
		totalPages = 1
	}

	for page := 0; page < totalPages; page++ {
		start := page * postsPerPage
		end := start + postsPerPage
		if end > len(g.Posts) {
			end = len(g.Posts)
		}

		var outPath string
		if page == 0 {
			outPath = filepath.Join(g.Target, "index.html")
		} else {
			outDir := filepath.Join(g.Target, "page", fmt.Sprintf("%d", page+1))
			if err := os.MkdirAll(outDir, 0755); err != nil {
				return err
			}
			outPath = filepath.Join(outDir, "index.html")
		}

		f, err := os.Create(outPath)
		if err != nil {
			return err
		}

		var prevPage, nextPage string
		if page > 0 {
			if page == 1 {
				prevPage = "/"
			} else {
				prevPage = fmt.Sprintf("/page/%d/", page)
			}
		}
		if page < totalPages-1 {
			nextPage = fmt.Sprintf("/page/%d/", page+2)
		}

		data := struct {
			Site     *models.Site
			Posts    []*models.Post
			PrevPage string
			NextPage string
		}{
			Site:     &models.SiteConfig,
			Posts:    g.Posts[start:end],
			PrevPage: prevPage,
			NextPage: nextPage,
		}

		if err := g.templates.ExecuteTemplate(f, "index.html", data); err != nil {
			if closeErr := f.Close(); closeErr != nil {
				return fmt.Errorf("rendering %s: %w (close error: %v)", outPath, err, closeErr)
			}
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) generateFeed() error {
	outPath := filepath.Join(g.Target, "feed.xml")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}

	// Limit feed to latest 20 posts
	posts := g.Posts
	if len(posts) > 20 {
		posts = posts[:20]
	}

	data := struct {
		Site      *models.Site
		Posts     []*models.Post
		BuildDate string
	}{
		Site:      &models.SiteConfig,
		Posts:     posts,
		BuildDate: time.Now().Format(time.RFC1123Z),
	}

	if err := g.templates.ExecuteTemplate(f, "feed.xml", data); err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return fmt.Errorf("rendering %s: %w (close error: %v)", outPath, err, closeErr)
		}
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return nil
}

func (g *Generator) generateSitemap() error {
	outPath := filepath.Join(g.Target, "sitemap.xml")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}

	data := struct {
		Site      *models.Site
		Posts     []*models.Post
		BuildDate string
	}{
		Site:      &models.SiteConfig,
		Posts:     g.Posts,
		BuildDate: time.Now().Format("2006-01-02"),
	}

	if err := g.templates.ExecuteTemplate(f, "sitemap.xml", data); err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return fmt.Errorf("rendering %s: %w (close error: %v)", outPath, err, closeErr)
		}
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return nil
}

func (g *Generator) generateStylesheet() error {
	// Try production path first, then test path
	cssPath := "templates/styles.css"
	cssContent, err := g.templateFS.ReadFile(cssPath)
	if err != nil {
		cssPath = "testdata/templates/styles.css"
		cssContent, err = g.templateFS.ReadFile(cssPath)
		if err != nil {
			return fmt.Errorf("reading styles.css: %w", err)
		}
	}

	outPath := filepath.Join(g.Target, "styles.css")
	if err := os.WriteFile(outPath, cssContent, 0644); err != nil {
		return fmt.Errorf("writing styles.css: %w", err)
	}

	return nil
}

func (g *Generator) generateThemeScript() error {
	// Try production path first, then test path
	jsPath := "templates/theme.js"
	jsContent, err := g.templateFS.ReadFile(jsPath)
	if err != nil {
		jsPath = "testdata/templates/theme.js"
		jsContent, err = g.templateFS.ReadFile(jsPath)
		if err != nil {
			return fmt.Errorf("reading theme.js: %w", err)
		}
	}

	outPath := filepath.Join(g.Target, "theme.js")
	if err := os.WriteFile(outPath, jsContent, 0644); err != nil {
		return fmt.Errorf("writing theme.js: %w", err)
	}

	return nil
}

func (g *Generator) copyAssets() error {
	return filepath.Walk(g.Source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the source directory itself (handles symlinks)
		if path == g.Source {
			return nil
		}

		// Skip markdown files and directories
		if info.IsDir() || strings.HasSuffix(path, ".md") {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(g.Source, path)
		if err != nil {
			return err
		}

		// Create destination path
		destPath := filepath.Join(g.Target, "assets", relPath)
		destDir := filepath.Dir(destPath)

		// Create destination directory
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		// Copy file
		return copyFile(path, destPath)
	})
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		if closeErr := srcFile.Close(); closeErr != nil {
			return fmt.Errorf("closing %s: %v (after create error: %w)", src, closeErr, err)
		}
		return err
	}

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		closeDstErr := dstFile.Close()
		closeSrcErr := srcFile.Close()
		if closeDstErr != nil && closeSrcErr != nil {
			return fmt.Errorf("copy %s: %w; close dst: %v; close src: %v", src, err, closeDstErr, closeSrcErr)
		}
		if closeDstErr != nil {
			return fmt.Errorf("copy %s: %w; close dst: %v", src, err, closeDstErr)
		}
		if closeSrcErr != nil {
			return fmt.Errorf("copy %s: %w; close src: %v", src, err, closeSrcErr)
		}
		return err
	}

	if err := dstFile.Close(); err != nil {
		return err
	}
	if err := srcFile.Close(); err != nil {
		return err
	}

	return nil
}

// copyBundleAssets copies all non-.md files from a bundle directory
// to the post's output directory, resizing images as needed
func (g *Generator) copyBundleAssets(post *models.Post, outDir string) error {
	entries, err := os.ReadDir(post.BundleDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Skip markdown files
		if strings.HasSuffix(filename, ".md") {
			continue
		}

		// Skip hidden files
		if strings.HasPrefix(filename, ".") {
			continue
		}

		srcPath := filepath.Join(post.BundleDir, filename)
		dstPath := filepath.Join(outDir, filename)

		if width, needsResize := post.ImageResizes[filename]; needsResize && isImageFile(filename) {
			if err := resizeImage(srcPath, dstPath, width); err != nil {
				return fmt.Errorf("resizing %s: %w", filename, err)
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("copying %s: %w", filename, err)
			}
		}
	}

	return nil
}

// isImageFile checks if a filename has an image extension
func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}
	return false
}

// resizeImage resizes an image to the specified width, maintaining aspect ratio.
// If the requested width is larger than the original, the image is copied as-is (no upscaling).
func resizeImage(srcPath, dstPath string, width int) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}

	srcImg, format, err := image.Decode(srcFile)
	if err != nil {
		if closeErr := srcFile.Close(); closeErr != nil {
			return fmt.Errorf("decoding image: %w (close error: %v)", err, closeErr)
		}
		return fmt.Errorf("decoding image: %w", err)
	}

	origWidth := srcImg.Bounds().Dx()
	if width >= origWidth {
		if err := srcFile.Close(); err != nil {
			return err
		}
		return copyFile(srcPath, dstPath)
	}

	if err := srcFile.Close(); err != nil {
		return err
	}

	// Check if we support encoding this format
	switch format {
	case "jpeg", "png", "gif":
		// Supported formats, proceed with resize
	default:
		// Unsupported format (e.g., webp) - copy original instead
		fmt.Printf("[WARN] Cannot encode %s format, copying original image\n", format)
		return copyFile(srcPath, dstPath)
	}

	g := gift.New(gift.Resize(width, 0, gift.LanczosResampling))
	dst := image.NewRGBA(g.Bounds(srcImg.Bounds()))
	g.Draw(dst, srcImg)

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}

	var encodeErr error
	switch format {
	case "jpeg":
		encodeErr = jpeg.Encode(dstFile, dst, &jpeg.Options{Quality: 85})
	case "png":
		encodeErr = png.Encode(dstFile, dst)
	case "gif":
		encodeErr = gif.Encode(dstFile, dst, nil)
	}

	if closeErr := dstFile.Close(); closeErr != nil {
		if encodeErr != nil {
			return fmt.Errorf("encoding: %w (close error: %v)", encodeErr, closeErr)
		}
		return closeErr
	}

	return encodeErr
}
