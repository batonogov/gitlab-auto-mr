package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xanzy/go-gitlab"
)

func TestWaitForPipeline(t *testing.T) {
	// Setup test server
	var requestCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// First request: pipeline is running
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{
				"id": 1,
				"status": "running",
				"sha": "abc123"
			}`)); err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		} else {
			// Second request: pipeline is successful
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{
				"id": 1,
				"status": "success",
				"sha": "abc123"
			}`)); err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		}
	}))
	defer ts.Close()

	//nolint:staticcheck // TODO: migrate to new client when it's stable
	git, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(ts.URL+"/api/v4"))
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	status, err := waitForPipeline(git, "123", 1, 10)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !status.Success {
		t.Error("Expected pipeline to be successful")
	}
	if !status.Completed {
		t.Error("Expected pipeline to be completed")
	}
	if status.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", status.Status)
	}
}

func TestGetPipelineID(t *testing.T) {
	// Setup test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`[{
			"id": 123,
			"status": "running",
			"sha": "abc123"
		}]`)); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	//nolint:staticcheck // TODO: migrate to new client when it's stable
	git, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(ts.URL+"/api/v4"))
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	pipelineID, err := getPipelineID(git, "123", "abc123")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if pipelineID != 123 {
		t.Errorf("Expected pipeline ID 123, got %d", pipelineID)
	}
}

func TestGetPipelineIDNoResults(t *testing.T) {
	// Setup test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`[]`)); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	//nolint:staticcheck // TODO: migrate to new client when it's stable
	git, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(ts.URL+"/api/v4"))
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	_, err = getPipelineID(git, "123", "abc123")
	if err == nil {
		t.Error("Expected error when no pipelines found")
	}
}
