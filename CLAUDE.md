# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
task build                    # Build the project (runs linters, tests, and builds binary)
task lint                     # Run linters only
./obsite <source> <target>    # Generate site from source markdown to target directory
```

**Note**: `task build` already runs all linters and tests, so there's no need to manually run `go test` or `go build` for the whole project. Use `task build` before completing any task.

## Architecture

**obsite** is a minimal static site generator for blogs. It converts markdown files with YAML frontmatter into a static HTML site.

### Core Flow

1. **main.go** - Entry point, embeds templates via `//go:embed`, passes source/target dirs to generator
2. **internal/generator/** - Orchestrates site generation: collects posts, renders templates, generates RSS/sitemap, copies assets
3. **internal/parser/** - Parses markdown files with YAML frontmatter using goldmark, extracts post metadata. Supports both single files and page bundles (directories with index.md)
4. **internal/models/** - Data structures for `Post` and `Site` config (edit `SiteConfig` in post.go to configure site metadata). Post struct supports both obsite fields and Hugo-compatible fields (categories, summary, etc.) for migrated content
5. **templates/** - Go HTML/XML templates embedded at compile time

### Post Structure

Markdown files require YAML frontmatter with:
- `title`, `created` (date), `slug` (optional, derived from filename)
- `draft: true` to exclude from build
- Posts are output to `/<year>/<month>/<slug>/index.html`

### Page Bundles

obsite supports two post formats:

1. **Single files**: `post-name.md` in the source directory
2. **Page bundles**: `post-name/index.md` + assets (images, etc.) in a directory

For page bundles:
- Directory name becomes the slug (if not explicitly specified in frontmatter)
- All non-.md files in the bundle directory (except hidden files starting with `.`) are copied to the post's output directory
- Images referenced relatively in markdown work automatically: `![alt](image.jpg)`
- Bundle assets are output to `<target>/<year>/<month>/<slug>/image.jpg`

Example structure:
```
source/
  movie-review/
    index.md
    cover.jpg
    poster.png
  single-post.md

Output:
target/
  2024/06/movie-review/
    index.html
    cover.jpg
    poster.png
  2024/06/single-post/
    index.html
```

### Key Design Decisions

- Templates are embedded in binary (no external template files needed at runtime)
- Site config is hardcoded in `internal/models/post.go` - change and rebuild to update
- Assets are handled differently based on post type:
  - **Single-file posts**: non-.md files in source directory are copied to `<target>/assets/`
  - **Page bundles**: non-.md files in the bundle directory are copied alongside the post's index.html
- Pagination: 10 posts per page, pages at `/page/N/`

## Testing

All new features and bug fixes must include matching unit tests.

Tests use golden files stored in `testdata/` directories:
- `*.input.md` - test input files
- `*.golden.md` - expected output files

To update golden files after intentional changes:
```bash
UPDATE_GOLDEN=true go test ./...
```
