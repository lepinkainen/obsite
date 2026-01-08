package models

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestPost_IsDraft(t *testing.T) {
	tests := []struct {
		name string
		post Post
		want bool
	}{
		{
			name: "draft field true",
			post: Post{Draft: true},
			want: true,
		},
		{
			name: "draft tag lowercase",
			post: Post{Tags: []string{"blog", "draft"}},
			want: true,
		},
		{
			name: "draft tag uppercase",
			post: Post{Tags: []string{"blog", "Draft"}},
			want: true,
		},
		{
			name: "draft tag mixed case",
			post: Post{Tags: []string{"DRAFT"}},
			want: true,
		},
		{
			name: "both draft field and tag",
			post: Post{Draft: true, Tags: []string{"draft"}},
			want: true,
		},
		{
			name: "not a draft - no tags",
			post: Post{},
			want: false,
		},
		{
			name: "not a draft - other tags",
			post: Post{Tags: []string{"blog", "tutorial"}},
			want: false,
		},
		{
			name: "not a draft - draft false explicitly",
			post: Post{Draft: false, Tags: []string{"blog"}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.post.IsDraft(); got != tt.want {
				t.Errorf("Post.IsDraft() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlexibleTime_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, ft FlexibleTime)
	}{
		{
			name:    "date with time",
			input:   "created: 2022-03-02 08:21",
			wantErr: false,
			check: func(t *testing.T, ft FlexibleTime) {
				if ft.Year() != 2022 || ft.Month() != 3 || ft.Day() != 2 {
					t.Errorf("date = %v, want 2022-03-02", ft.Time)
				}
				if ft.Hour() != 8 || ft.Minute() != 21 {
					t.Errorf("time = %02d:%02d, want 08:21", ft.Hour(), ft.Minute())
				}
			},
		},
		{
			name:    "ISO 8601 UTC",
			input:   "created: 2024-01-15T10:30:00Z",
			wantErr: false,
			check: func(t *testing.T, ft FlexibleTime) {
				if ft.Year() != 2024 || ft.Month() != 1 || ft.Day() != 15 {
					t.Errorf("date = %v, want 2024-01-15", ft.Time)
				}
			},
		},
		{
			name:    "ISO 8601 with timezone",
			input:   "created: 2024-06-20T14:00:00+02:00",
			wantErr: false,
			check: func(t *testing.T, ft FlexibleTime) {
				if ft.Year() != 2024 || ft.Month() != 6 || ft.Day() != 20 {
					t.Errorf("date = %v, want 2024-06-20", ft.Time)
				}
			},
		},
		{
			name:    "date only",
			input:   "created: 2024-12-25",
			wantErr: false,
			check: func(t *testing.T, ft FlexibleTime) {
				if ft.Year() != 2024 || ft.Month() != 12 || ft.Day() != 25 {
					t.Errorf("date = %v, want 2024-12-25", ft.Time)
				}
			},
		},
		{
			name:    "invalid date format",
			input:   "created: not-a-date",
			wantErr: true,
			check:   nil,
		},
		{
			name:    "invalid date values",
			input:   "created: 2024-13-45",
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wrapper struct {
				Created FlexibleTime `yaml:"created"`
			}

			err := yaml.Unmarshal([]byte(tt.input), &wrapper)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, wrapper.Created)
			}
		})
	}
}

func TestPost_URLPath(t *testing.T) {
	post := Post{
		Slug:    "my-post",
		Created: FlexibleTime{Time: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
	}

	got := post.URLPath()
	want := "/2024/03/my-post/"
	if got != want {
		t.Errorf("URLPath() = %q, want %q", got, want)
	}
}

func TestPost_FullURL(t *testing.T) {
	// Save and restore original config
	origConfig := SiteConfig
	defer func() { SiteConfig = origConfig }()

	SiteConfig.BaseURL = "https://example.com"

	post := Post{
		Slug:    "test-post",
		Created: FlexibleTime{Time: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)},
	}

	got := post.FullURL()
	want := "https://example.com/2024/01/test-post/"
	if got != want {
		t.Errorf("FullURL() = %q, want %q", got, want)
	}
}

func TestPost_FormattedDate(t *testing.T) {
	post := Post{
		Created: FlexibleTime{Time: time.Date(2024, 7, 4, 0, 0, 0, 0, time.UTC)},
	}

	got := post.FormattedDate()
	want := "July 4, 2024"
	if got != want {
		t.Errorf("FormattedDate() = %q, want %q", got, want)
	}
}

func TestPost_RFCDate(t *testing.T) {
	post := Post{
		Created: FlexibleTime{Time: time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC)},
	}

	got := post.RFCDate()
	want := "2024-12-25T10:30:00Z"
	if got != want {
		t.Errorf("RFCDate() = %q, want %q", got, want)
	}
}
