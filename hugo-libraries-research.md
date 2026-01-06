# Hugo Static Site Generator - Library Dependencies Report

Research conducted: 2026-01-06

This report documents the key libraries used by the Hugo static site generator that could potentially enhance the obsite project.

## Executive Summary

Hugo uses a carefully selected set of Go libraries to handle various aspects of static site generation. The most relevant libraries for obsite are focused on markdown parsing, frontmatter handling, syntax highlighting, and image processing.

---

## Core Content Processing Libraries

### 1. Goldmark (Markdown Parser)
- **Package**: `github.com/yuin/goldmark` (v1.7.13)
- **Purpose**: Primary markdown parser (CommonMark compliant)
- **Why it matters**: Hugo migrated from Blackfriday to Goldmark for better performance and standards compliance
- **Features**:
  - Fast performance (comparable to cmark, the C reference implementation)
  - Clean, extensible AST structure
  - Full CommonMark compliance
  - Memory efficient
  - Depends only on standard libraries

**Current obsite status**: Already using Goldmark ✓

### 2. Goldmark Extensions

#### a. Core Extensions
- **Package**: `github.com/yuin/goldmark/extension`
- **Built-in extensions**:
  - Tables
  - Strikethrough
  - Task lists
  - Definition lists
  - Typographer (smart quotes, dashes, ellipses)
  - Footnotes

#### b. Emoji Support
- **Package**: `github.com/yuin/goldmark-emoji` (v1.0.6)
- **Purpose**: Add emoji support to markdown (e.g., `:smile:` → 😄)

#### c. Table of Contents
- **Package**: `go.abhg.dev/goldmark/toc` or `github.com/abhinav/goldmark-toc`
- **Purpose**: Automatically generate table of contents from headings
- **Features**:
  - Three modes: Extension (automatic), Transformer (more control), Manual (most control)
  - Configurable depth (MinDepth, MaxDepth)
  - Compact mode to remove empty items
  - Demo: https://abhinav.github.io/goldmark-toc/demo/

#### d. Hugo-Specific Extensions
- **Package**: `github.com/gohugoio/hugo-goldmark-extensions/extras` (v0.5.0)
- **Package**: `github.com/gohugoio/hugo-goldmark-extensions/passthrough` (v0.3.1)
- **Purpose**: Custom markdown features specific to Hugo

---

## Syntax Highlighting

### Chroma
- **Package**: `github.com/alecthomas/chroma/v2` (v2.21.1)
- **Purpose**: Syntax highlighting for code blocks
- **Features**:
  - Pure Go implementation
  - Supports 200+ languages
  - Multiple output formats (HTML, terminal)
  - Compatible with Pygments themes

**Current obsite status**: Mentioned as known library

---

## Configuration & Frontmatter

### 1. YAML Processing
- **Package**: `github.com/goccy/go-yaml` (v1.19.1)
- **Purpose**: YAML parsing for frontmatter and configuration
- **Why this one**: Hugo uses this instead of gopkg.in/yaml.v3
- **Features**:
  - Fast performance
  - Better error messages
  - Colored output support

### 2. TOML Support
- **Package**: `github.com/pelletier/go-toml/v2` (v2.2.4)
- **Purpose**: TOML configuration parsing
- **Use case**: Alternative frontmatter format

### 3. JSON/XML Conversion
- **Package**: `github.com/clbanning/mxj/v2` (v2.7.0)
- **Purpose**: XML to JSON conversion and manipulation
- **Use case**: Content format flexibility

---

## Image Processing Libraries

### 1. GIFT (Go Image Filtering Toolkit)
- **Package**: `github.com/disintegration/gift` (v1.2.1)
- **Purpose**: Comprehensive image transformation
- **Features**:
  - Resizing, rotation, cropping
  - Filters (blur, sharpen, brightness, contrast, etc.)
  - Color adjustments
  - Convolution filters

### 2. Image Resizing
- **Package**: `github.com/nfnt/resize`
- **Purpose**: Simple image resizing
- **Features**: Multiple interpolation algorithms

### 3. Image Metadata
- **Package**: `github.com/bep/imagemeta` (v0.12.1)
- **Purpose**: Extract EXIF and other image metadata
- **Use case**: Photo blogs, galleries

### 4. Smart Crop
- **Package**: `github.com/muesli/smartcrop` (v0.3.0)
- **Purpose**: Intelligent image cropping based on content analysis
- **Use case**: Automatic thumbnail generation

### 5. Color Extraction
- **Package**: `github.com/marekm4/color-extractor` (v1.2.1)
- **Purpose**: Extract dominant colors from images
- **Use case**: Theme generation, color schemes

### 6. Image Dithering
- **Package**: `github.com/makeworld-the-better-one/dither/v2` (v2.4.0)
- **Purpose**: Apply dithering effects to images
- **Use case**: Artistic effects, retro aesthetics

---

## Content Management & Processing

### 1. HTML Sanitization
- **Package**: `github.com/microcosm-cc/bluemonday` (v1.0.27)
- **Purpose**: HTML sanitizer (XSS protection)
- **Use case**: When accepting user-generated HTML or processing untrusted content

### 2. HTML to Markdown
- **Package**: `github.com/JohannesKaufmann/html-to-markdown/v2` (v2.5.0)
- **Purpose**: Convert HTML back to markdown
- **Use case**: Content migration, import tools

### 3. Org-mode Support
- **Package**: `github.com/niklasfasching/go-org` (v1.9.1)
- **Purpose**: Parse Emacs Org-mode files
- **Use case**: Alternative input format for content

### 4. Natural Language Processing
- **Package**: `github.com/jdkato/prose` (v1.2.1)
- **Purpose**: Text processing, tokenization, part-of-speech tagging
- **Use case**: Content analysis, reading time estimation, keyword extraction

### 5. Content Hashing
- **Package**: `github.com/gohugoio/hashstructure` (v0.6.0)
- **Purpose**: Hash Go data structures for cache keys
- **Use case**: Content change detection, cache invalidation

---

## CLI & Infrastructure

### 1. Cobra
- **Package**: `github.com/spf13/cobra` (v1.10.2)
- **Purpose**: CLI framework
- **Features**: Command structure, flags, help generation
- **Use case**: Enhanced CLI interface for obsite

---

## Priority Recommendations for obsite

Based on the current obsite architecture, here are the most valuable additions:

### High Priority
1. **Goldmark Extensions** (`github.com/yuin/goldmark/extension`)
   - Tables, strikethrough, footnotes already built-in
   - Easy to add, significant value

2. **Syntax Highlighting** (`github.com/alecthomas/chroma/v2`)
   - Essential for code-heavy blogs
   - Pure Go, no external dependencies

3. **Table of Contents** (`go.abhg.dev/goldmark/toc`)
   - Improves navigation for long posts
   - Simple integration with existing Goldmark setup

### Medium Priority
4. **Emoji Support** (`github.com/yuin/goldmark-emoji`)
   - Easy to add, nice quality-of-life feature

5. **Better YAML Parser** (`github.com/goccy/go-yaml`)
   - Better error messages than gopkg.in/yaml.v3
   - Would improve debugging frontmatter issues

6. **TOML Support** (`github.com/pelletier/go-toml/v2`)
   - Alternative frontmatter format
   - Some users prefer TOML over YAML

### Lower Priority (Feature-Specific)
7. **Image Processing** (GIFT, smartcrop, etc.)
   - Only if adding image manipulation features
   - Significant scope increase

8. **Prose NLP** (`github.com/jdkato/prose`)
   - Reading time estimation
   - Keyword extraction
   - Tag suggestions

9. **CLI Framework** (`github.com/spf13/cobra`)
   - Current simple approach works fine
   - Only if expanding CLI features significantly

---

## Implementation Considerations

### Minimal Dependencies Philosophy
- obsite currently has minimal dependencies
- Each addition increases binary size and complexity
- Consider which features truly add value vs. bloat

### Testing Requirements
- All new features require tests with golden files
- Ensure additions follow existing testing patterns

### Backward Compatibility
- Ensure new features don't break existing sites
- Make extensions optional/configurable

---

## References & Sources

- [Hugo Official Website](https://gohugo.io/)
- [Hugo GitHub Repository](https://github.com/gohugoio/hugo)
- [Hugo go.mod File](https://github.com/gohugoio/hugo/blob/master/go.mod)
- [Goldmark GitHub](https://github.com/yuin/goldmark)
- [Goldmark TOC Extension](https://github.com/abhinav/goldmark-toc)
- [Hugo Content Formats Documentation](https://gohugo.io/content-management/formats/)
- [Hugo Markup Configuration](https://gohugo.io/configuration/markup/)
- [Complete Guide to Using Hugo](https://strapi.io/blog/guide-to-using-hugo-site-generator)
- [Goldmark TOC Demo](https://abhinav.github.io/goldmark-toc/demo/)

---

**Report compiled**: 2026-01-06
**Hugo version reference**: Latest (master branch)
**Obsite version**: Current development version
