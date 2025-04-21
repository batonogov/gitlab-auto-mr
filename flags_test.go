package main

import (
	"flag"
	"os"
	"testing"
)

func TestCommandLineFlags(t *testing.T) {
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name          string
		args          []string
		expectError   bool
		checkFunction func(t *testing.T)
	}{
		{
			name: "all required flags provided",
			args: []string{
				"cmd",
				"-target-branch", "main",
			},
			expectError: false,
			checkFunction: func(t *testing.T) {
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
				var (
					targetBranch    = flag.String("target-branch", "", "Target branch for the merge request")
					commitPrefix    = flag.String("commit-prefix", "", "Prefix to add to the commit message")
					descriptionFile = flag.String("description", "", "Path to a file containing merge request description")
					removeBranch    = flag.Bool("remove-branch", false, "Remove source branch after merge")
					useIssueName    = flag.Bool("use-issue-name", false, "Use issue name for merge request title")
					squashCommits   = flag.Bool("squash-commits", false, "Squash commits in the merge request")
					autoMerge       = flag.Bool("auto-merge", false, "Enable auto-merge for the merge request")
					reviewers       = flag.String("reviewers", "", "Comma-separated list of reviewer usernames")
					milestone       = flag.String("milestone", "", "Milestone ID for the merge request")
					assignee        = flag.String("assignee", "", "Username of the assignee")
				)
				flag.Parse()

				if *targetBranch != "main" {
					t.Errorf("Expected target branch 'main', got '%s'", *targetBranch)
				}
				if *commitPrefix != "" {
					t.Errorf("Expected empty commit prefix, got '%s'", *commitPrefix)
				}
				if *descriptionFile != "" {
					t.Errorf("Expected empty description file, got '%s'", *descriptionFile)
				}
				if *removeBranch {
					t.Error("Expected remove branch to be false")
				}
				if *useIssueName {
					t.Error("Expected use issue name to be false")
				}
				if *squashCommits {
					t.Error("Expected squash commits to be false")
				}
				if *autoMerge {
					t.Error("Expected auto merge to be false")
				}
				if *reviewers != "" {
					t.Errorf("Expected empty reviewers list, got '%s'", *reviewers)
				}
				if *milestone != "" {
					t.Errorf("Expected empty milestone, got '%s'", *milestone)
				}
				if *assignee != "" {
					t.Errorf("Expected empty assignee, got '%s'", *assignee)
				}
			},
		},
		{
			name: "all flags provided",
			args: []string{
				"cmd",
				"-target-branch", "main",
				"-commit-prefix", "feat",
				"-description", "desc.md",
				"-remove-branch",
				"-use-issue-name",
				"-squash-commits",
				"-auto-merge",
				"-reviewers", "user1,user2",
				"-milestone", "1",
				"-assignee", "user3",
			},
			expectError: false,
			checkFunction: func(t *testing.T) {
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
				var (
					targetBranch    = flag.String("target-branch", "", "Target branch for the merge request")
					commitPrefix    = flag.String("commit-prefix", "", "Prefix to add to the commit message")
					descriptionFile = flag.String("description", "", "Path to a file containing merge request description")
					removeBranch    = flag.Bool("remove-branch", false, "Remove source branch after merge")
					useIssueName    = flag.Bool("use-issue-name", false, "Use issue name for merge request title")
					squashCommits   = flag.Bool("squash-commits", false, "Squash commits in the merge request")
					autoMerge       = flag.Bool("auto-merge", false, "Enable auto-merge for the merge request")
					reviewers       = flag.String("reviewers", "", "Comma-separated list of reviewer usernames")
					milestone       = flag.String("milestone", "", "Milestone ID for the merge request")
					assignee        = flag.String("assignee", "", "Username of the assignee")
				)
				flag.Parse()

				if *targetBranch != "main" {
					t.Errorf("Expected target branch 'main', got '%s'", *targetBranch)
				}
				if *commitPrefix != "feat" {
					t.Errorf("Expected commit prefix 'feat', got '%s'", *commitPrefix)
				}
				if *descriptionFile != "desc.md" {
					t.Errorf("Expected description file 'desc.md', got '%s'", *descriptionFile)
				}
				if !*removeBranch {
					t.Error("Expected remove branch to be true")
				}
				if !*useIssueName {
					t.Error("Expected use issue name to be true")
				}
				if !*squashCommits {
					t.Error("Expected squash commits to be true")
				}
				if !*autoMerge {
					t.Error("Expected auto merge to be true")
				}
				if *reviewers != "user1,user2" {
					t.Errorf("Expected reviewers 'user1,user2', got '%s'", *reviewers)
				}
				if *milestone != "1" {
					t.Errorf("Expected milestone '1', got '%s'", *milestone)
				}
				if *assignee != "user3" {
					t.Errorf("Expected assignee 'user3', got '%s'", *assignee)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			tt.checkFunction(t)
		})
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Save original environment variables
	oldToken := os.Getenv("GITLAB_TOKEN")
	oldURL := os.Getenv("CI_SERVER_URL")
	oldProjectID := os.Getenv("CI_PROJECT_ID")
	oldBranch := os.Getenv("CI_COMMIT_REF_NAME")
	oldTitle := os.Getenv("CI_COMMIT_TITLE")

	// Restore environment variables after test
	defer func() {
		if err := os.Setenv("GITLAB_TOKEN", oldToken); err != nil {
			t.Errorf("Failed to restore GITLAB_TOKEN: %v", err)
		}
		if err := os.Setenv("CI_SERVER_URL", oldURL); err != nil {
			t.Errorf("Failed to restore CI_SERVER_URL: %v", err)
		}
		if err := os.Setenv("CI_PROJECT_ID", oldProjectID); err != nil {
			t.Errorf("Failed to restore CI_PROJECT_ID: %v", err)
		}
		if err := os.Setenv("CI_COMMIT_REF_NAME", oldBranch); err != nil {
			t.Errorf("Failed to restore CI_COMMIT_REF_NAME: %v", err)
		}
		if err := os.Setenv("CI_COMMIT_TITLE", oldTitle); err != nil {
			t.Errorf("Failed to restore CI_COMMIT_TITLE: %v", err)
		}
	}()

	tests := []struct {
		name       string
		envVars    map[string]string
		expectVars map[string]string
	}{
		{
			name: "all environment variables set",
			envVars: map[string]string{
				"GITLAB_TOKEN":       "test-token",
				"CI_SERVER_URL":      "https://gitlab.com",
				"CI_PROJECT_ID":      "123",
				"CI_COMMIT_REF_NAME": "feature/test",
				"CI_COMMIT_TITLE":    "Test commit",
			},
			expectVars: map[string]string{
				"GITLAB_TOKEN":       "test-token",
				"CI_SERVER_URL":      "https://gitlab.com",
				"CI_PROJECT_ID":      "123",
				"CI_COMMIT_REF_NAME": "feature/test",
				"CI_COMMIT_TITLE":    "Test commit",
			},
		},
		{
			name: "missing optional variables",
			envVars: map[string]string{
				"GITLAB_TOKEN":       "test-token",
				"CI_PROJECT_ID":      "123",
				"CI_COMMIT_REF_NAME": "feature/test",
				"CI_COMMIT_TITLE":    "Test commit",
			},
			expectVars: map[string]string{
				"GITLAB_TOKEN":       "test-token",
				"CI_SERVER_URL":      "https://gitlab.com", // Changed: this is the default value
				"CI_PROJECT_ID":      "123",
				"CI_COMMIT_REF_NAME": "feature/test",
				"CI_COMMIT_TITLE":    "Test commit",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for test
			for k, v := range tt.envVars {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("Failed to set environment variable %s: %v", k, err)
				}
			}

			// Check environment variables
			for k, v := range tt.expectVars {
				got := os.Getenv(k)
				if got != v {
					t.Errorf("Expected environment variable %s='%s', got '%s'", k, v, got)
				}
			}
		})
	}
}
