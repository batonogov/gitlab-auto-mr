package main

import (
	"testing"
)

func TestExtractIssueIID(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		want       string
	}{
		{
			name:       "feature branch with issue number",
			branchName: "feature/123-description",
			want:       "123",
		},
		{
			name:       "simple branch with issue number",
			branchName: "123-description",
			want:       "123",
		},
		{
			name:       "branch without issue number",
			branchName: "feature-description",
			want:       "",
		},
		{
			name:       "empty branch name",
			branchName: "",
			want:       "",
		},
		{
			name:       "multiple slashes with number",
			branchName: "feature/123/description",
			want:       "123",
		},
		{
			name:       "non-numeric after slash",
			branchName: "feature/abc-description",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractIssueIID(tt.branchName); got != tt.want {
				t.Errorf("extractIssueIID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractIssueIIDAsInt(t *testing.T) {
	tests := []struct {
		name string
		iid  string
		want int
	}{
		{
			name: "valid number",
			iid:  "123",
			want: 123,
		},
		{
			name: "invalid number",
			iid:  "abc",
			want: 0,
		},
		{
			name: "empty string",
			iid:  "",
			want: 0,
		},
		{
			name: "mixed content",
			iid:  "123abc",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractIssueIIDAsInt(tt.iid); got != tt.want {
				t.Errorf("extractIssueIIDAsInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTitle(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		title  string
		want   string
	}{
		{
			name:   "empty prefix",
			prefix: "",
			title:  "Test Title",
			want:   "Test Title",
		},
		{
			name:   "with prefix",
			prefix: "feat",
			title:  "Test Title",
			want:   "feat: Test Title",
		},
		{
			name:   "title already has prefix",
			prefix: "feat",
			title:  "feat: Test Title",
			want:   "feat: Test Title",
		},
		{
			name:   "empty title",
			prefix: "feat",
			title:  "",
			want:   "feat: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTitle(tt.prefix, tt.title); got != tt.want {
				t.Errorf("getTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}
