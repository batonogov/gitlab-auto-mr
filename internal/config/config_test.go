package config

import (
	"flag"
	"os"
	"testing"
)

func TestParseUserIDs(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []int
		wantError bool
	}{
		{
			name:      "empty string",
			input:     "",
			expected:  nil,
			wantError: false,
		},
		{
			name:      "single user ID",
			input:     "123",
			expected:  []int{123},
			wantError: false,
		},
		{
			name:      "multiple user IDs",
			input:     "123,456,789",
			expected:  []int{123, 456, 789},
			wantError: false,
		},
		{
			name:      "user IDs with spaces",
			input:     "123, 456 , 789",
			expected:  []int{123, 456, 789},
			wantError: false,
		},
		{
			name:      "invalid user ID",
			input:     "123,abc,789",
			expected:  nil,
			wantError: true,
		},
		{
			name:      "empty values",
			input:     "123,,789",
			expected:  []int{123, 789},
			wantError: false,
		},
		{
			name:      "only commas",
			input:     ",,,",
			expected:  nil,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseUserIDs(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseUserIDs() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseUserIDs() unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("parseUserIDs() length = %d, want %d", len(result), len(tt.expected))
				return
			}

			for i, id := range result {
				if id != tt.expected[i] {
					t.Errorf("parseUserIDs()[%d] = %d, want %d", i, id, tt.expected[i])
				}
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				PrivateToken: "token",
				SourceBranch: "feature/test",
				ProjectID:    123,
				GitLabURL:    "https://gitlab.com",
			},
			wantError: false,
		},
		{
			name: "missing private token",
			config: Config{
				SourceBranch: "feature/test",
				ProjectID:    123,
				GitLabURL:    "https://gitlab.com",
			},
			wantError: true,
			errorMsg:  "private token is required (use --private-token or GITLAB_PRIVATE_TOKEN environment variable)",
		},
		{
			name: "missing source branch",
			config: Config{
				PrivateToken: "token",
				ProjectID:    123,
				GitLabURL:    "https://gitlab.com",
			},
			wantError: true,
			errorMsg:  "source branch is required (use --source-branch or CI_COMMIT_REF_NAME environment variable)",
		},
		{
			name: "missing project ID",
			config: Config{
				PrivateToken: "token",
				SourceBranch: "feature/test",
				GitLabURL:    "https://gitlab.com",
			},
			wantError: true,
			errorMsg:  "project ID is required (use --project-id or CI_PROJECT_ID environment variable)",
		},
		{
			name: "missing GitLab URL",
			config: Config{
				PrivateToken: "token",
				SourceBranch: "feature/test",
				ProjectID:    123,
			},
			wantError: true,
			errorMsg:  "GitLab URL is required (use --gitlab-url or CI_PROJECT_URL environment variable)",
		},
		{
			name: "URL without protocol gets https prefix",
			config: Config{
				PrivateToken: "token",
				SourceBranch: "feature/test",
				ProjectID:    123,
				GitLabURL:    "gitlab.com",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalURL := tt.config.GitLabURL
			err := tt.config.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("Config.Validate() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Config.Validate() error = %q, want %q", err.Error(), tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Config.Validate() unexpected error: %v", err)
				return
			}

			// Check if URL was prefixed with https://
			if originalURL == "gitlab.com" && tt.config.GitLabURL != "https://gitlab.com" {
				t.Errorf("Config.Validate() should prefix URL with https://, got %s", tt.config.GitLabURL)
			}
		})
	}
}

func TestEnvironmentVariableHandling(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"GITLAB_PRIVATE_TOKEN": os.Getenv("GITLAB_PRIVATE_TOKEN"),
		"CI_COMMIT_REF_NAME":   os.Getenv("CI_COMMIT_REF_NAME"),
		"CI_PROJECT_ID":        os.Getenv("CI_PROJECT_ID"),
		"CI_PROJECT_URL":       os.Getenv("CI_PROJECT_URL"),
		"GITLAB_USER_ID":       os.Getenv("GITLAB_USER_ID"),
	}

	// Clean environment
	for key := range originalEnv {
		os.Unsetenv(key)
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected Config
	}{
		{
			name: "all environment variables set",
			envVars: map[string]string{
				"GITLAB_PRIVATE_TOKEN": "env-token",
				"CI_COMMIT_REF_NAME":   "env-branch",
				"CI_PROJECT_ID":        "456",
				"CI_PROJECT_URL":       "https://gitlab.example.com/group/project",
				"GITLAB_USER_ID":       "789",
			},
			expected: Config{
				PrivateToken: "env-token",
				SourceBranch: "env-branch",
				ProjectID:    456,
				GitLabURL:    "https://gitlab.example.com",
				UserIDs:      []int{789},
				TargetBranch: "main",  // default value
				CommitPrefix: "Draft", // default value
			},
		},
		{
			name: "project URL extraction",
			envVars: map[string]string{
				"CI_PROJECT_URL": "https://gitlab.com/namespace/project",
			},
			expected: Config{
				GitLabURL:    "https://gitlab.com",
				TargetBranch: "main",
				CommitPrefix: "Draft",
			},
		},
		{
			name: "invalid project ID ignored",
			envVars: map[string]string{
				"CI_PROJECT_ID": "invalid",
			},
			expected: Config{
				ProjectID:    0, // should remain 0 for invalid input
				TargetBranch: "main",
				CommitPrefix: "Draft",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Clear command line for this test
			oldArgs := os.Args
			os.Args = []string{"cmd"}
			defer func() { os.Args = oldArgs }()

			// Reset flag package state
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			config, err := ParseFlags()
			if err != nil {
				t.Fatalf("ParseFlags() error: %v", err)
			}

			// Check specific fields
			if config.PrivateToken != tt.expected.PrivateToken {
				t.Errorf("PrivateToken = %q, want %q", config.PrivateToken, tt.expected.PrivateToken)
			}

			if config.SourceBranch != tt.expected.SourceBranch {
				t.Errorf("SourceBranch = %q, want %q", config.SourceBranch, tt.expected.SourceBranch)
			}

			if config.ProjectID != tt.expected.ProjectID {
				t.Errorf("ProjectID = %d, want %d", config.ProjectID, tt.expected.ProjectID)
			}

			if config.GitLabURL != tt.expected.GitLabURL {
				t.Errorf("GitLabURL = %q, want %q", config.GitLabURL, tt.expected.GitLabURL)
			}

			if len(config.UserIDs) != len(tt.expected.UserIDs) {
				t.Errorf("UserIDs length = %d, want %d", len(config.UserIDs), len(tt.expected.UserIDs))
			} else {
				for i, id := range config.UserIDs {
					if id != tt.expected.UserIDs[i] {
						t.Errorf("UserIDs[%d] = %d, want %d", i, id, tt.expected.UserIDs[i])
					}
				}
			}

			// Clean up environment for next test
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	// Clear environment
	envVars := []string{
		"GITLAB_PRIVATE_TOKEN",
		"CI_COMMIT_REF_NAME",
		"CI_PROJECT_ID",
		"CI_PROJECT_URL",
		"GITLAB_USER_ID",
	}

	originalValues := make(map[string]string)
	for _, env := range envVars {
		originalValues[env] = os.Getenv(env)
		os.Unsetenv(env)
	}

	defer func() {
		for env, value := range originalValues {
			if value != "" {
				os.Setenv(env, value)
			}
		}
	}()

	// Clear command line
	oldArgs := os.Args
	os.Args = []string{"cmd"}
	defer func() { os.Args = oldArgs }()

	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	config, err := ParseFlags()
	if err != nil {
		t.Fatalf("ParseFlags() error: %v", err)
	}

	// Check defaults
	if config.TargetBranch != "main" {
		t.Errorf("Default TargetBranch = %q, want %q", config.TargetBranch, "main")
	}

	if config.CommitPrefix != "Draft" {
		t.Errorf("Default CommitPrefix = %q, want %q", config.CommitPrefix, "Draft")
	}

	if config.Insecure != false {
		t.Errorf("Default Insecure = %v, want %v", config.Insecure, false)
	}

	if config.RemoveBranch != false {
		t.Errorf("Default RemoveBranch = %v, want %v", config.RemoveBranch, false)
	}

	if config.SquashCommits != false {
		t.Errorf("Default SquashCommits = %v, want %v", config.SquashCommits, false)
	}
}
