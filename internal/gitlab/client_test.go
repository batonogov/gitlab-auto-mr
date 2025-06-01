package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		token     string
		insecure  bool
		wantError bool
	}{
		{
			name:      "valid parameters",
			baseURL:   "https://gitlab.com",
			token:     "test-token",
			insecure:  false,
			wantError: false,
		},
		{
			name:      "valid parameters with trailing slash",
			baseURL:   "https://gitlab.com/",
			token:     "test-token",
			insecure:  false,
			wantError: false,
		},
		{
			name:      "empty baseURL",
			baseURL:   "",
			token:     "test-token",
			insecure:  false,
			wantError: true,
		},
		{
			name:      "empty token",
			baseURL:   "https://gitlab.com",
			token:     "",
			insecure:  false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.baseURL, tt.token, tt.insecure)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewClient() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewClient() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Errorf("NewClient() returned nil client")
				return
			}

			if !strings.HasSuffix(client.baseURL, "/") {
				t.Errorf("NewClient() baseURL should end with slash, got: %s", client.baseURL)
			}

			if client.token != tt.token {
				t.Errorf("NewClient() token = %s, want %s", client.token, tt.token)
			}
		})
	}
}

func TestCreateMergeRequest(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Check headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		token := r.Header.Get("PRIVATE-TOKEN")
		if token != "test-token" {
			t.Errorf("Expected PRIVATE-TOKEN: test-token, got %s", token)
		}

		// Check URL path
		expectedPath := "/api/v4/projects/123/merge_requests"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Parse request body
		var opts MergeRequestOptions
		if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Return mock response
		response := MergeRequest{
			ID:           1,
			IID:          1,
			ProjectID:    123,
			Title:        opts.Title,
			Description:  opts.Description,
			State:        "opened",
			SourceBranch: opts.SourceBranch,
			TargetBranch: opts.TargetBranch,
			WebURL:       "https://gitlab.com/group/project/-/merge_requests/1",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	opts := &MergeRequestOptions{
		SourceBranch: "feature/test",
		TargetBranch: "main",
		Title:        "Test MR",
		Description:  "Test description",
	}

	mr, err := client.CreateMergeRequest(123, opts)
	if err != nil {
		t.Fatalf("CreateMergeRequest() error: %v", err)
	}

	if mr == nil {
		t.Fatal("CreateMergeRequest() returned nil")
	}

	if mr.Title != opts.Title {
		t.Errorf("CreateMergeRequest() title = %s, want %s", mr.Title, opts.Title)
	}

	if mr.SourceBranch != opts.SourceBranch {
		t.Errorf("CreateMergeRequest() source_branch = %s, want %s", mr.SourceBranch, opts.SourceBranch)
	}
}

func TestCreateMergeRequestWithError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")

		errorResponse := GitLabError{
			Message: "Bad Request",
			Errors: map[string]string{
				"source_branch": "does not exist",
			},
		}
		json.NewEncoder(w).Encode(errorResponse)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	opts := &MergeRequestOptions{
		SourceBranch: "nonexistent",
		TargetBranch: "main",
		Title:        "Test MR",
	}

	_, err = client.CreateMergeRequest(123, opts)
	if err == nil {
		t.Fatal("CreateMergeRequest() expected error, got nil")
	}

	gitlabErr, ok := err.(*GitLabError)
	if !ok {
		t.Fatalf("Expected GitLabError, got %T", err)
	}

	if gitlabErr.Message != "Bad Request" {
		t.Errorf("Expected message 'Bad Request', got %s", gitlabErr.Message)
	}
}

func TestGetMergeRequests(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Check query parameters
		query := r.URL.Query()
		if query.Get("source_branch") != "feature/test" {
			t.Errorf("Expected source_branch=feature/test, got %s", query.Get("source_branch"))
		}
		if query.Get("target_branch") != "main" {
			t.Errorf("Expected target_branch=main, got %s", query.Get("target_branch"))
		}
		if query.Get("state") != "opened" {
			t.Errorf("Expected state=opened, got %s", query.Get("state"))
		}

		// Return mock response
		response := []MergeRequest{
			{
				ID:           1,
				IID:          1,
				ProjectID:    123,
				Title:        "Test MR",
				State:        "opened",
				SourceBranch: "feature/test",
				TargetBranch: "main",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mrs, err := client.GetMergeRequests(123, "feature/test", "main")
	if err != nil {
		t.Fatalf("GetMergeRequests() error: %v", err)
	}

	if len(mrs) != 1 {
		t.Fatalf("Expected 1 merge request, got %d", len(mrs))
	}

	mr := mrs[0]
	if mr.SourceBranch != "feature/test" {
		t.Errorf("Expected source_branch=feature/test, got %s", mr.SourceBranch)
	}
}

func TestCheckMergeRequestExists(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   []MergeRequest
		expectedExists bool
	}{
		{
			name:           "no merge requests",
			mockResponse:   []MergeRequest{},
			expectedExists: false,
		},
		{
			name: "merge request exists",
			mockResponse: []MergeRequest{
				{
					ID:           1,
					SourceBranch: "feature/test",
					TargetBranch: "main",
				},
			},
			expectedExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token", false)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			exists, err := client.CheckMergeRequestExists(123, "feature/test", "main")
			if err != nil {
				t.Fatalf("CheckMergeRequestExists() error: %v", err)
			}

			if exists != tt.expectedExists {
				t.Errorf("CheckMergeRequestExists() = %v, want %v", exists, tt.expectedExists)
			}
		})
	}
}

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseUserIDs(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("ParseUserIDs() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseUserIDs() unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("ParseUserIDs() length = %d, want %d", len(result), len(tt.expected))
				return
			}

			for i, id := range result {
				if id != tt.expected[i] {
					t.Errorf("ParseUserIDs()[%d] = %d, want %d", i, id, tt.expected[i])
				}
			}
		})
	}
}

func TestGitLabError(t *testing.T) {
	tests := []struct {
		name     string
		error    GitLabError
		expected string
	}{
		{
			name: "error with message",
			error: GitLabError{
				Message: "Bad Request",
			},
			expected: "Bad Request",
		},
		{
			name: "error with field errors",
			error: GitLabError{
				Errors: map[string]string{
					"source_branch": "does not exist",
					"target_branch": "is required",
				},
			},
			expected: "source_branch: does not exist, target_branch: is required",
		},
		{
			name:     "empty error",
			error:    GitLabError{},
			expected: "unknown GitLab API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.Error()

			// For map errors, we can't guarantee order, so check that all parts are present
			if strings.Contains(tt.expected, ":") && strings.Contains(tt.expected, ",") {
				// This is a field errors case - check that all parts are present
				for field, msg := range tt.error.Errors {
					expectedPart := fmt.Sprintf("%s: %s", field, msg)
					if !strings.Contains(result, expectedPart) {
						t.Errorf("GitLabError.Error() = %s, should contain %s", result, expectedPart)
					}
				}
			} else {
				if result != tt.expected {
					t.Errorf("GitLabError.Error() = %s, want %s", result, tt.expected)
				}
			}
		})
	}
}
