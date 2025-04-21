package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/xanzy/go-gitlab"
)

func TestCreateMergeRequest(t *testing.T) {
	// Setup test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/123/merge_requests":
			w.WriteHeader(http.StatusCreated)
			if _, err := w.Write([]byte(`{
				"id": 1,
				"iid": 1,
				"project_id": 123,
				"title": "Test MR",
				"description": "Test description",
				"state": "opened",
				"web_url": "http://gitlab.example.com/project/merge_requests/1"
			}`)); err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		case "/api/v4/projects/123/merge_requests/1/notes":
			w.WriteHeader(http.StatusCreated)
			if _, err := w.Write([]byte(`{"id": 1}`)); err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		}
	}))
	defer ts.Close()

	// Set test environment
	oldToken := os.Getenv("GITLAB_TOKEN")
	oldURL := os.Getenv("CI_SERVER_URL")
	oldProjectID := os.Getenv("CI_PROJECT_ID")
	oldBranch := os.Getenv("CI_COMMIT_REF_NAME")
	oldTitle := os.Getenv("CI_COMMIT_TITLE")
	defer func() {
		if err := os.Setenv("GITLAB_TOKEN", oldToken); err != nil {
			t.Fatalf("Failed to restore GITLAB_TOKEN: %v", err)
		}
		if err := os.Setenv("CI_SERVER_URL", oldURL); err != nil {
			t.Fatalf("Failed to restore CI_SERVER_URL: %v", err)
		}
		if err := os.Setenv("CI_PROJECT_ID", oldProjectID); err != nil {
			t.Fatalf("Failed to restore CI_PROJECT_ID: %v", err)
		}
		if err := os.Setenv("CI_COMMIT_REF_NAME", oldBranch); err != nil {
			t.Fatalf("Failed to restore CI_COMMIT_REF_NAME: %v", err)
		}
		if err := os.Setenv("CI_COMMIT_TITLE", oldTitle); err != nil {
			t.Fatalf("Failed to restore CI_COMMIT_TITLE: %v", err)
		}
	}()

	if err := os.Setenv("GITLAB_TOKEN", "test-token"); err != nil {
		t.Fatalf("Failed to set GITLAB_TOKEN: %v", err)
	}
	if err := os.Setenv("CI_SERVER_URL", ts.URL); err != nil {
		t.Fatalf("Failed to set CI_SERVER_URL: %v", err)
	}
	if err := os.Setenv("CI_PROJECT_ID", "123"); err != nil {
		t.Fatalf("Failed to set CI_PROJECT_ID: %v", err)
	}
	if err := os.Setenv("CI_COMMIT_REF_NAME", "feature/test"); err != nil {
		t.Fatalf("Failed to set CI_COMMIT_REF_NAME: %v", err)
	}
	if err := os.Setenv("CI_COMMIT_TITLE", "Test commit"); err != nil {
		t.Fatalf("Failed to set CI_COMMIT_TITLE: %v", err)
	}

	// Initialize GitLab client
	//nolint:staticcheck // TODO: migrate to new client when it's stable
	git, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(ts.URL+"/api/v4"))
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	// Test creating MR
	title := "Test MR"
	sourceBranch := "feature/test"
	targetBranch := "main"
	mrOpts := &gitlab.CreateMergeRequestOptions{
		Title:        &title,
		SourceBranch: &sourceBranch,
		TargetBranch: &targetBranch,
	}

	mr, _, err := git.MergeRequests.CreateMergeRequest("123", mrOpts)
	if err != nil {
		t.Fatalf("Failed to create merge request: %v", err)
	}

	if mr.Title != "Test MR" {
		t.Errorf("Expected MR title to be 'Test MR', got '%s'", mr.Title)
	}
}

func TestListProjectMergeRequests(t *testing.T) {
	// Setup test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/projects/123/merge_requests" {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`[{
				"id": 1,
				"iid": 1,
				"project_id": 123,
				"title": "Existing MR",
				"source_branch": "feature/test",
				"target_branch": "main",
				"state": "opened",
				"web_url": "http://gitlab.example.com/project/merge_requests/1"
			}]`)); err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		}
	}))
	defer ts.Close()

	// Initialize GitLab client
	//nolint:staticcheck // TODO: migrate to new client when it's stable
	git, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(ts.URL+"/api/v4"))
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	// Test listing MRs
	sourceBranch := "feature/test"
	targetBranch := "main"
	state := "opened"

	mrs, _, err := git.MergeRequests.ListProjectMergeRequests("123", &gitlab.ListProjectMergeRequestsOptions{
		SourceBranch: &sourceBranch,
		TargetBranch: &targetBranch,
		State:        &state,
	})

	if err != nil {
		t.Fatalf("Failed to list merge requests: %v", err)
	}

	if len(mrs) != 1 {
		t.Errorf("Expected 1 merge request, got %d", len(mrs))
	}

	if mrs[0].SourceBranch != sourceBranch {
		t.Errorf("Expected source branch '%s', got '%s'", sourceBranch, mrs[0].SourceBranch)
	}

	if mrs[0].TargetBranch != targetBranch {
		t.Errorf("Expected target branch '%s', got '%s'", targetBranch, mrs[0].TargetBranch)
	}
}
