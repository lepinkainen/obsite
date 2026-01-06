# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
go build -o obsite .          # Build the binary
go test ./...                  # Run all tests
./obsite <source> <target>    # Generate site from source markdown to target directory
```

## Architecture

**obsite** is a minimal static site generator for blogs. It converts markdown files with YAML frontmatter into a static HTML site.

### Core Flow

1. **main.go** - Entry point, embeds templates via `//go:embed`, passes source/target dirs to generator
2. **internal/generator/** - Orchestrates site generation: collects posts, renders templates, generates RSS/sitemap, copies assets
3. **internal/parser/** - Parses markdown files with YAML frontmatter using goldmark, extracts post metadata
4. **internal/models/** - Data structures for `Post` and `Site` config (edit `SiteConfig` in post.go to configure site metadata)
5. **templates/** - Go HTML/XML templates embedded at compile time

### Post Structure

Markdown files require YAML frontmatter with:
- `title`, `created` (date), `slug` (optional, derived from filename)
- `draft: true` to exclude from build
- Posts are output to `/<year>/<month>/<slug>/index.html`

### Key Design Decisions

- Templates are embedded in binary (no external template files needed at runtime)
- Site config is hardcoded in `internal/models/post.go` - change and rebuild to update
- Assets (non-.md files in source) are copied to `<target>/assets/`
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
