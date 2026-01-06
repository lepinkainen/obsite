package models

import "testing"

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
