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
			name:       "branch with feature prefix",
			branchName: "feature/123-description",
			want:       "123",
		},
		{
			name:       "branch with bug prefix",
			branchName: "bug/456-fix-something",
			want:       "456",
		},
		{
			name:       "branch without prefix",
			branchName: "789-some-work",
			want:       "789",
		},
		{
			name:       "branch without issue number",
			branchName: "feature/no-issue",
			want:       "",
		},
		{
			name:       "empty branch name",
			branchName: "",
			want:       "",
		},
		{
			name:       "multiple numbers",
			branchName: "feature/123-456",
			want:       "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIssueIID(tt.branchName)
			if got != tt.want {
				t.Errorf("extractIssueIID(%q) = %v, want %v", tt.branchName, got, tt.want)
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
			name: "zero",
			iid:  "0",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIssueIIDAsInt(tt.iid)
			if got != tt.want {
				t.Errorf("extractIssueIIDAsInt(%q) = %v, want %v", tt.iid, got, tt.want)
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
			name:   "with prefix",
			prefix: "feat",
			title:  "add new feature",
			want:   "feat: add new feature",
		},
		{
			name:   "empty prefix",
			prefix: "",
			title:  "add new feature",
			want:   "add new feature",
		},
		{
			name:   "title already has prefix",
			prefix: "feat",
			title:  "feat: add new feature",
			want:   "feat: add new feature",
		},
		{
			name:   "empty title",
			prefix: "feat",
			title:  "",
			want:   "feat: ",
		},
		{
			name:   "both empty",
			prefix: "",
			title:  "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTitle(tt.prefix, tt.title)
			if got != tt.want {
				t.Errorf("getTitle(%q, %q) = %v, want %v", tt.prefix, tt.title, got, tt.want)
			}
		})
	}
}
