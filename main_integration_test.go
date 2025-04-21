package main

import (
	"flag"
	"os"
	"testing"

	"github.com/xanzy/go-gitlab"
)

func setupGitlabClient(t *testing.T) *gitlab.Client {
	token := os.Getenv("GITLAB_TOKEN")
	url := os.Getenv("CI_SERVER_URL")
	if url == "" {
		url = "https://gitlab.com"
	}

	git, err := gitlab.NewClient(token, gitlab.WithBaseURL(url+"/api/v4"))
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}
	return git
}

func TestIntegrationMergeRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	requiredEnvVars := []string{
		"GITLAB_TOKEN",
		"CI_PROJECT_ID",
		"CI_COMMIT_REF_NAME",
		"CI_COMMIT_TITLE",
	}

	for _, env := range requiredEnvVars {
		if os.Getenv(env) == "" {
			t.Skipf("Skipping test: %s environment variable not set", env)
		}
	}

	// Prepare test data
	testBranch := "test-integration"
	testPrefix := "test"
	descContent := "Test merge request created by integration test"

	// Create temporary description file
	tmpDescFile, err := os.CreateTemp("", "mr-desc-*.md")
	if err != nil {
		t.Fatalf("Failed to create temporary description file: %v", err)
	}
	defer os.Remove(tmpDescFile.Name())

	if _, err := tmpDescFile.WriteString(descContent); err != nil {
		t.Fatalf("Failed to write description file: %v", err)
	}
	if err := tmpDescFile.Close(); err != nil {
		t.Fatalf("Failed to close description file: %v", err)
	}

	// Set up test arguments
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{
		"gitlab-auto-mr",
		"--target-branch", testBranch,
		"--commit-prefix", testPrefix,
		"--remove-branch",
		"--description", tmpDescFile.Name(),
	}

	// Reset flags for the test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Initialize GitLab client for verification
	git := setupGitlabClient(t)
	projectID := os.Getenv("CI_PROJECT_ID")
	sourceBranch := os.Getenv("CI_COMMIT_REF_NAME")

	// Run the main function (creates MR)
	main()

	// Verify MR was created
	mrs, _, err := git.MergeRequests.ListProjectMergeRequests(projectID, &gitlab.ListProjectMergeRequestsOptions{
		SourceBranch: &sourceBranch,
		TargetBranch: &testBranch,
		State:        gitlab.String("opened"),
	})
	if err != nil {
		t.Fatalf("Failed to list merge requests: %v", err)
	}

	if len(mrs) == 0 {
		t.Error("No merge request was created")
	}

	// Clean up - close the created MR
	if len(mrs) > 0 {
		_, _, err = git.MergeRequests.UpdateMergeRequest(projectID, mrs[0].IID, &gitlab.UpdateMergeRequestOptions{
			StateEvent: gitlab.String("close"),
		})
		if err != nil {
			t.Logf("Warning: Failed to close test merge request: %v", err)
		}
	}
}
