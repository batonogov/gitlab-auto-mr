package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDescriptionFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "utils_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		filePath    string
		fileContent string
		createFile  bool
		expected    string
		wantError   bool
	}{
		{
			name:      "empty file path",
			filePath:  "",
			expected:  "",
			wantError: false,
		},
		{
			name:        "valid file",
			filePath:    filepath.Join(tempDir, "test.md"),
			fileContent: "# Test Description\n\nThis is a test description.",
			createFile:  true,
			expected:    "# Test Description\n\nThis is a test description.",
			wantError:   false,
		},
		{
			name:       "non-existent file",
			filePath:   filepath.Join(tempDir, "nonexistent.md"),
			createFile: false,
			expected:   "",
			wantError:  true,
		},
		{
			name:        "empty file",
			filePath:    filepath.Join(tempDir, "empty.md"),
			fileContent: "",
			createFile:  true,
			expected:    "",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createFile && tt.filePath != "" {
				err := os.WriteFile(tt.filePath, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			result, err := ReadDescriptionFile(tt.filePath)

			if tt.wantError {
				if err == nil {
					t.Errorf("ReadDescriptionFile() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ReadDescriptionFile() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ReadDescriptionFile() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractIssueNumber(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		expected   string
		wantError  bool
	}{
		{
			name:       "feature branch with issue",
			branchName: "feature/#123",
			expected:   "123",
			wantError:  false,
		},
		{
			name:       "bugfix branch with issue",
			branchName: "bugfix/#456-fix-login",
			expected:   "456",
			wantError:  false,
		},
		{
			name:       "branch with issue in middle",
			branchName: "feature/add-#789-support",
			expected:   "789",
			wantError:  false,
		},
		{
			name:       "branch without issue number",
			branchName: "feature/add-support",
			expected:   "",
			wantError:  true,
		},
		{
			name:       "branch with hash but no number",
			branchName: "feature/#",
			expected:   "",
			wantError:  true,
		},
		{
			name:       "empty branch name",
			branchName: "",
			expected:   "",
			wantError:  true,
		},
		{
			name:       "multiple issue numbers",
			branchName: "feature/#123-and-#456",
			expected:   "123", // Should return first match
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractIssueNumber(tt.branchName)

			if tt.wantError {
				if err == nil {
					t.Errorf("ExtractIssueNumber() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractIssueNumber() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ExtractIssueNumber() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateMRTitle(t *testing.T) {
	tests := []struct {
		name         string
		sourceBranch string
		commitPrefix string
		customTitle  string
		useIssueName bool
		expected     string
		wantError    bool
	}{
		{
			name:         "custom title with prefix",
			sourceBranch: "feature/test",
			commitPrefix: "Draft",
			customTitle:  "Add new feature",
			useIssueName: false,
			expected:     "Draft: Add new feature",
			wantError:    false,
		},
		{
			name:         "custom title without prefix",
			sourceBranch: "feature/test",
			commitPrefix: "",
			customTitle:  "Add new feature",
			useIssueName: false,
			expected:     "Add new feature",
			wantError:    false,
		},
		{
			name:         "use issue name with prefix",
			sourceBranch: "feature/#123",
			commitPrefix: "Draft",
			customTitle:  "",
			useIssueName: true,
			expected:     "Draft: Resolve #123",
			wantError:    false,
		},
		{
			name:         "use issue name without prefix",
			sourceBranch: "feature/#456",
			commitPrefix: "",
			customTitle:  "",
			useIssueName: true,
			expected:     "Resolve #456",
			wantError:    false,
		},
		{
			name:         "use issue name but no issue in branch",
			sourceBranch: "feature/test",
			commitPrefix: "Draft",
			customTitle:  "",
			useIssueName: true,
			expected:     "",
			wantError:    true,
		},
		{
			name:         "default title from branch name with prefix",
			sourceBranch: "feature/add-user-auth",
			commitPrefix: "Draft",
			customTitle:  "",
			useIssueName: false,
			expected:     "Draft: Feature/add User Auth",
			wantError:    false,
		},
		{
			name:         "default title from branch name without prefix",
			sourceBranch: "bugfix/fix_login_issue",
			commitPrefix: "",
			customTitle:  "",
			useIssueName: false,
			expected:     "Bugfix/fix Login Issue",
			wantError:    false,
		},
		{
			name:         "custom title takes precedence over issue name",
			sourceBranch: "feature/#123",
			commitPrefix: "Draft",
			customTitle:  "Custom Title",
			useIssueName: true,
			expected:     "Draft: Custom Title",
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateMRTitle(tt.sourceBranch, tt.commitPrefix, tt.customTitle, tt.useIssueName)

			if tt.wantError {
				if err == nil {
					t.Errorf("GenerateMRTitle() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GenerateMRTitle() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("GenerateMRTitle() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		expected   string
	}{
		{
			name:       "clean branch name",
			branchName: "feature/test",
			expected:   "feature/test",
		},
		{
			name:       "branch with origin prefix",
			branchName: "origin/feature/test",
			expected:   "feature/test",
		},
		{
			name:       "branch with refs/heads prefix",
			branchName: "refs/heads/feature/test",
			expected:   "feature/test",
		},
		{
			name:       "branch with whitespace",
			branchName: "  feature/test  ",
			expected:   "feature/test",
		},
		{
			name:       "branch with both prefixes",
			branchName: "origin/refs/heads/feature/test",
			expected:   "feature/test", // Removes both prefixes sequentially
		},
		{
			name:       "empty branch name",
			branchName: "",
			expected:   "",
		},
		{
			name:       "only whitespace",
			branchName: "   ",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeBranchName(tt.branchName)
			if result != tt.expected {
				t.Errorf("SanitizeBranchName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "valid https URL",
			url:      "https://gitlab.com",
			expected: true,
		},
		{
			name:     "valid http URL",
			url:      "http://gitlab.com",
			expected: true,
		},
		{
			name:     "URL without protocol",
			url:      "gitlab.com",
			expected: false,
		},
		{
			name:     "URL with path",
			url:      "https://gitlab.com/group/project",
			expected: true,
		},
		{
			name:     "empty URL",
			url:      "",
			expected: false,
		},
		{
			name:     "invalid URL",
			url:      "not-a-url",
			expected: false,
		},
		{
			name:     "URL with port",
			url:      "https://gitlab.com:8080",
			expected: true,
		},
		{
			name:     "ftp URL",
			url:      "ftp://example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidURL(tt.url)
			if result != tt.expected {
				t.Errorf("IsValidURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatUserIDs(t *testing.T) {
	tests := []struct {
		name     string
		userIDs  []int
		expected string
	}{
		{
			name:     "empty slice",
			userIDs:  []int{},
			expected: "",
		},
		{
			name:     "nil slice",
			userIDs:  nil,
			expected: "",
		},
		{
			name:     "single user ID",
			userIDs:  []int{123},
			expected: "123",
		},
		{
			name:     "multiple user IDs",
			userIDs:  []int{123, 456, 789},
			expected: "123, 456, 789",
		},
		{
			name:     "user IDs with zero",
			userIDs:  []int{0, 123, 456},
			expected: "0, 123, 456",
		},
		{
			name:     "negative user IDs",
			userIDs:  []int{-1, 123},
			expected: "-1, 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatUserIDs(tt.userIDs)
			if result != tt.expected {
				t.Errorf("FormatUserIDs() = %q, want %q", result, tt.expected)
			}
		})
	}
}
