package models

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// Site configuration - edit these values and rebuild
var SiteConfig = Site{
	Title:       "My Blog",
	Description: "Personal blog",
	BaseURL:     "https://example.com",
	Author:      "Your Name",
}

type Site struct {
	Title       string
	Description string
	BaseURL     string
	Author      string
}

// FlexibleTime handles multiple date formats in YAML
type FlexibleTime struct {
	time.Time
}

var dateFormats = []string{
	"2006-01-02 15:04",     // 2022-03-02 08:21
	"2006-01-02T15:04:05Z", // ISO 8601
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02", // Just date
	time.RFC3339,
}

func (ft *FlexibleTime) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var dateStr string
	if err := unmarshal(&dateStr); err != nil {
		return err
	}

	for _, format := range dateFormats {
		if t, err := time.Parse(format, dateStr); err == nil {
			ft.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse date: %s", dateStr)
}

type Post struct {
	Title       string       `yaml:"title"`
	Created     FlexibleTime `yaml:"created"`
	Slug        string       `yaml:"slug"`
	Draft       bool         `yaml:"draft"`
	Tags        []string     `yaml:"tags"`
	Description string       `yaml:"description"`

	// Computed fields
	Content  string        // Raw markdown content
	HTML     template.HTML // Rendered HTML (safe for template output)
	URL      string        // Full URL path (e.g., /2024/01/my-post/)
	FilePath string        // Source file path
}

// URLPath returns the URL path for this post (e.g., /2024/01/my-slug/)
func (p *Post) URLPath() string {
	return fmt.Sprintf("/%d/%02d/%s/", p.Created.Year(), p.Created.Month(), p.Slug)
}

// FullURL returns the complete URL including base URL
func (p *Post) FullURL() string {
	return SiteConfig.BaseURL + p.URLPath()
}

// FormattedDate returns the created date in a human-readable format
func (p *Post) FormattedDate() string {
	return p.Created.Format("January 2, 2006")
}

// RFCDate returns the date in RFC 3339 format for RSS/sitemap
func (p *Post) RFCDate() string {
	return p.Created.Format(time.RFC3339)
}

// IsDraft returns true if the post is a draft (either via draft field or "draft" tag)
func (p *Post) IsDraft() bool {
	if p.Draft {
		return true
	}
	for _, tag := range p.Tags {
		if strings.EqualFold(tag, "draft") {
			return true
		}
	}
	return false
}
