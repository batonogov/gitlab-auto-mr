package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestGetEnv(t *testing.T) {
	// Test with existing environment variable
	os.Setenv("TEST_VAR", "test_value")
	result := getEnv("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}

	// Test with non-existing environment variable
	result = getEnv("NON_EXISTING_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}

	// Cleanup
	os.Unsetenv("TEST_VAR")
}

func TestGetEnvInt(t *testing.T) {
	// Test with valid integer
	os.Setenv("TEST_INT", "123")
	result := getEnvInt("TEST_INT", 0)
	if result != 123 {
		t.Errorf("Expected 123, got %d", result)
	}

	// Test with invalid integer
	os.Setenv("TEST_INT", "invalid")
	result = getEnvInt("TEST_INT", 456)
	if result != 456 {
		t.Errorf("Expected 456, got %d", result)
	}

	// Test with non-existing environment variable
	result = getEnvInt("NON_EXISTING_INT", 789)
	if result != 789 {
		t.Errorf("Expected 789, got %d", result)
	}

	// Cleanup
	os.Unsetenv("TEST_INT")
}

func TestParseIntSlice(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"", nil},
		{"123", []int{123}},
		{"123,456", []int{123, 456}},
		{"123, 456, 789", []int{123, 456, 789}},
		{"123,invalid,456", []int{123, 456}},
		{" 123 , 456 ", []int{123, 456}},
		{"invalid", []int{}},
	}

	for _, test := range tests {
		result := parseIntSlice(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("Input '%s': expected length %d, got %d", test.input, len(test.expected), len(result))
			continue
		}

		for i, v := range result {
			if i >= len(test.expected) || v != test.expected[i] {
				t.Errorf("Input '%s': expected %v, got %v", test.input, test.expected, result)
				break
			}
		}
	}
}

func TestGetMRTitle(t *testing.T) {
	tests := []struct {
		prefix   string
		title    string
		branch   string
		expected string
	}{
		{"Draft", "", "feature/test", "Draft: feature/test"},
		{"", "Custom Title", "feature/test", "Custom Title"},
		{"Draft", "Custom Title", "feature/test", "Draft: Custom Title"},
		{"", "", "feature/test", "feature/test"},
	}

	for _, test := range tests {
		result := getMRTitle(test.prefix, test.title, test.branch)
		if result != test.expected {
			t.Errorf("prefix='%s', title='%s', branch='%s': expected '%s', got '%s'",
				test.prefix, test.title, test.branch, test.expected, result)
		}
	}
}

func TestValidateMR(t *testing.T) {
	// Test valid branches
	err := validateMR("feature/test", "main")
	if err != nil {
		t.Errorf("Expected no error for different branches, got: %v", err)
	}

	// Test same branches
	err = validateMR("main", "main")
	if err == nil {
		t.Error("Expected error for same branches, got none")
	}
}

func TestGetDescriptionData(t *testing.T) {
	// Test with empty path
	result := getDescriptionData("")
	if result != "" {
		t.Errorf("Expected empty string for empty path, got '%s'", result)
	}

	// Test with non-existing file
	result = getDescriptionData("/non/existing/file.txt")
	if result != "" {
		t.Errorf("Expected empty string for non-existing file, got '%s'", result)
	}
}

func TestGetMRTitleWithUpdate(t *testing.T) {
	tests := []struct {
		prefix   string
		title    string
		branch   string
		expected string
		desc     string
	}{
		{"", "Updated Title", "feature/test", "Updated Title", "Custom title without prefix"},
		{"WIP", "Updated Title", "feature/test", "WIP: Updated Title", "Custom title with prefix"},
		{"", "", "feature/updated", "feature/updated", "Branch name as title"},
		{"Update", "", "feature/updated", "Update: feature/updated", "Branch name with prefix"},
	}

	for _, test := range tests {
		result := getMRTitle(test.prefix, test.title, test.branch)
		if result != test.expected {
			t.Errorf("%s: prefix='%s', title='%s', branch='%s': expected '%s', got '%s'",
				test.desc, test.prefix, test.title, test.branch, test.expected, result)
		}
	}
}

func TestMRUpdateRequest(t *testing.T) {
	// Test MRUpdateRequest struct initialization
	updateReq := MRUpdateRequest{
		Title:              "Updated Title",
		Description:        "Updated Description",
		AssigneeIDs:        []int{123, 456},
		ReviewerIDs:        []int{789},
		RemoveSourceBranch: true,
		Squash:             true,
		AllowCollaboration: false,
		MilestoneID:        999,
		Labels:             []string{"bug", "urgent"},
	}

	if updateReq.Title != "Updated Title" {
		t.Errorf("Expected 'Updated Title', got '%s'", updateReq.Title)
	}
	if len(updateReq.AssigneeIDs) != 2 {
		t.Errorf("Expected 2 assignees, got %d", len(updateReq.AssigneeIDs))
	}
	if len(updateReq.ReviewerIDs) != 1 {
		t.Errorf("Expected 1 reviewer, got %d", len(updateReq.ReviewerIDs))
	}
	if updateReq.MilestoneID != 999 {
		t.Errorf("Expected milestone ID 999, got %d", updateReq.MilestoneID)
	}
	if len(updateReq.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(updateReq.Labels))
	}
}

func TestConfigUpdateMRFlag(t *testing.T) {
	// Test that Config struct has UpdateMR field
	config := Config{
		UpdateMR: true,
	}

	if config.UpdateMR != true {
		t.Errorf("Expected UpdateMR to be true, got %v", config.UpdateMR)
	}

	config.UpdateMR = false
	if config.UpdateMR != false {
		t.Errorf("Expected UpdateMR to be false, got %v", config.UpdateMR)
	}
}

func TestConfigCreateOnlyFlag(t *testing.T) {
	// Test that Config struct has CreateOnly field
	config := Config{
		CreateOnly: true,
	}

	if config.CreateOnly != true {
		t.Errorf("Expected CreateOnly to be true, got %v", config.CreateOnly)
	}

	config.CreateOnly = false
	if config.CreateOnly != false {
		t.Errorf("Expected CreateOnly to be false, got %v", config.CreateOnly)
	}
}

func TestSmartMRManagement(t *testing.T) {
	// Test smart MR management behavior
	tests := []struct {
		name       string
		updateMR   bool
		createOnly bool
		expected   string
	}{
		{"Default smart mode", false, false, "smart"},
		{"Force update mode", true, false, "update"},
		{"Force create mode", false, true, "create"},
	}

	for _, test := range tests {
		config := Config{
			UpdateMR:   test.updateMR,
			CreateOnly: test.createOnly,
		}

		var mode string
		if config.UpdateMR {
			mode = "update"
		} else if config.CreateOnly {
			mode = "create"
		} else {
			mode = "smart"
		}

		if mode != test.expected {
			t.Errorf("%s: expected '%s', got '%s'", test.name, test.expected, mode)
		}
	}
}

func TestCreateHTTPClient(t *testing.T) {
	// Test secure client
	client := createHTTPClient(false)
	if client == nil {
		t.Error("Expected non-nil client")
	}
	if client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.Timeout)
	}

	// Test insecure client
	insecureClient := createHTTPClient(true)
	if insecureClient == nil {
		t.Error("Expected non-nil insecure client")
	}
	if insecureClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", insecureClient.Timeout)
	}
}

func TestGetProject(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if strings.Contains(r.URL.Path, "/projects/123") {
			project := Project{
				ID:            123,
				Name:          "test-project",
				DefaultBranch: "main",
			}
			json.NewEncoder(w).Encode(project)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
	}

	// Test successful request
	project, err := getProject(client, config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if project.ID != 123 {
		t.Errorf("Expected project ID 123, got %d", project.ID)
	}
	if project.Name != "test-project" {
		t.Errorf("Expected project name 'test-project', got '%s'", project.Name)
	}
	if project.DefaultBranch != "main" {
		t.Errorf("Expected default branch 'main', got '%s'", project.DefaultBranch)
	}

	// Test unauthorized request
	config.PrivateToken = ""
	_, err = getProject(client, config)
	if err == nil {
		t.Error("Expected error for unauthorized request")
	}
	if !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("Expected unauthorized error, got %v", err)
	}
}

func TestGetExistingMR(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mrs := []MergeRequest{
			{
				ID:           1,
				IID:          1,
				Title:        "Test MR",
				SourceBranch: "feature/test",
				TargetBranch: "main",
				State:        "opened",
			},
			{
				ID:           2,
				IID:          2,
				Title:        "Another MR",
				SourceBranch: "feature/other",
				TargetBranch: "main",
				State:        "opened",
			},
		}
		json.NewEncoder(w).Encode(mrs)
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
		SourceBranch: "feature/test",
		TargetBranch: "main",
	}

	// Test finding existing MR
	mr, err := getExistingMR(client, config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if mr == nil {
		t.Error("Expected to find MR, got nil")
	}
	if mr.SourceBranch != "feature/test" {
		t.Errorf("Expected source branch 'feature/test', got '%s'", mr.SourceBranch)
	}

	// Test not finding MR
	config.SourceBranch = "feature/nonexistent"
	mr, err = getExistingMR(client, config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if mr != nil {
		t.Error("Expected not to find MR, got one")
	}
}

func TestGetIssueData(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/issues/123") {
			issue := Issue{
				ID:     123,
				IID:    123,
				Title:  "Test Issue",
				Labels: []string{"bug", "urgent"},
				Milestone: struct {
					ID int `json:"id"`
				}{ID: 1},
			}
			json.NewEncoder(w).Encode(issue)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    456,
		PrivateToken: "test-token",
		SourceBranch: "feature/fix-#123",
	}

	// Test successful request
	issue, err := getIssueData(client, config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if issue.ID != 123 {
		t.Errorf("Expected issue ID 123, got %d", issue.ID)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("Expected issue title 'Test Issue', got '%s'", issue.Title)
	}
	if len(issue.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(issue.Labels))
	}
	if issue.Milestone.ID != 1 {
		t.Errorf("Expected milestone ID 1, got %d", issue.Milestone.ID)
	}

	// Test invalid branch name
	config.SourceBranch = "feature/no-issue"
	_, err = getIssueData(client, config)
	if err == nil {
		t.Error("Expected error for invalid branch name")
	}

	// Test invalid issue number
	config.SourceBranch = "feature/fix-#invalid"
	_, err = getIssueData(client, config)
	if err == nil {
		t.Error("Expected error for invalid issue number")
	}
}

func TestCreateMR(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var mrRequest MRCreateRequest
		err := json.NewDecoder(r.Body).Decode(&mrRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if mrRequest.SourceBranch == "" || mrRequest.TargetBranch == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing required fields"))
			return
		}

		w.WriteHeader(http.StatusCreated)
		mr := MergeRequest{
			ID:           1,
			IID:          1,
			Title:        mrRequest.Title,
			SourceBranch: mrRequest.SourceBranch,
			TargetBranch: mrRequest.TargetBranch,
			State:        "opened",
		}
		json.NewEncoder(w).Encode(mr)
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
	}

	// Test successful creation
	mrRequest := &MRCreateRequest{
		SourceBranch: "feature/test",
		TargetBranch: "main",
		Title:        "Test MR",
		Description:  "Test description",
	}

	err := createMR(client, config, mrRequest)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test failed creation
	mrRequest.SourceBranch = ""
	err = createMR(client, config, mrRequest)
	if err == nil {
		t.Error("Expected error for invalid request")
	}
}

func TestUpdateMR(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var updateRequest MRUpdateRequest
		err := json.NewDecoder(r.Body).Decode(&updateRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		mr := MergeRequest{
			ID:           1,
			IID:          1,
			Title:        updateRequest.Title,
			SourceBranch: "feature/test",
			TargetBranch: "main",
			State:        "opened",
		}
		json.NewEncoder(w).Encode(mr)
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
	}

	// Test successful update
	updateRequest := &MRUpdateRequest{
		Title:       "Updated Title",
		Description: "Updated description",
	}

	err := updateMR(client, config, 1, updateRequest)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRunCreateOnly(t *testing.T) {
	// Mock server that simulates no existing MR
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" && !strings.Contains(r.URL.Path, "merge_requests"):
			project := Project{
				ID:            123,
				Name:          "test-project",
				DefaultBranch: "main",
			}
			json.NewEncoder(w).Encode(project)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			// Return empty array (no existing MRs)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]MergeRequest{})
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &Config{
		GitLabURL:     server.URL,
		ProjectID:     123,
		PrivateToken:  "test-token",
		SourceBranch:  "feature/test",
		TargetBranch:  "main",
		UserIDs:       []int{1},
		CreateOnly:    true,
		CommitPrefix:  "Draft",
		RemoveBranch:  false,
		SquashCommits: false,
	}

	err := run(config)
	if err != nil {
		t.Errorf("Expected no error for create-only with no existing MR, got %v", err)
	}
}

func TestRunUpdateOnly(t *testing.T) {
	// Mock server that simulates existing MR
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" && !strings.Contains(r.URL.Path, "merge_requests"):
			project := Project{
				ID:            123,
				Name:          "test-project",
				DefaultBranch: "main",
			}
			json.NewEncoder(w).Encode(project)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			// Return existing MR
			mrs := []MergeRequest{
				{
					ID:           1,
					IID:          1,
					Title:        "Existing MR",
					SourceBranch: "feature/test",
					TargetBranch: "main",
					State:        "opened",
				},
			}
			json.NewEncoder(w).Encode(mrs)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests/1") && r.Method == "PUT":
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := &Config{
		GitLabURL:     server.URL,
		ProjectID:     123,
		PrivateToken:  "test-token",
		SourceBranch:  "feature/test",
		TargetBranch:  "main",
		UserIDs:       []int{1},
		UpdateMR:      true,
		CommitPrefix:  "Draft",
		RemoveBranch:  false,
		SquashCommits: false,
	}

	err := run(config)
	if err != nil {
		t.Errorf("Expected no error for update-only with existing MR, got %v", err)
	}
}

func TestRunMRExists(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" && !strings.Contains(r.URL.Path, "merge_requests"):
			project := Project{
				ID:            123,
				Name:          "test-project",
				DefaultBranch: "main",
			}
			json.NewEncoder(w).Encode(project)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			// Return existing MR
			mrs := []MergeRequest{
				{
					ID:           1,
					IID:          1,
					Title:        "Existing MR",
					SourceBranch: "feature/test",
					TargetBranch: "main",
					State:        "opened",
				},
			}
			json.NewEncoder(w).Encode(mrs)
		}
	}))
	defer server.Close()

	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
		SourceBranch: "feature/test",
		TargetBranch: "main",
		UserIDs:      []int{1},
		MRExists:     true,
	}

	err := run(config)
	if err != nil {
		t.Errorf("Expected no error for MR exists check, got %v", err)
	}
}

func TestRunErrorCases(t *testing.T) {
	// Test same source and target branches
	config := &Config{
		GitLabURL:    "https://gitlab.com",
		ProjectID:    123,
		PrivateToken: "test-token",
		SourceBranch: "main",
		TargetBranch: "main",
		UserIDs:      []int{1},
	}

	err := run(config)
	if err == nil {
		t.Error("Expected error for same source and target branches")
	}

	// Test create-only with existing MR
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" && !strings.Contains(r.URL.Path, "merge_requests"):
			project := Project{
				ID:            123,
				Name:          "test-project",
				DefaultBranch: "main",
			}
			json.NewEncoder(w).Encode(project)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			mrs := []MergeRequest{
				{
					ID:           1,
					IID:          1,
					Title:        "Existing MR",
					SourceBranch: "feature/test",
					TargetBranch: "main",
					State:        "opened",
				},
			}
			json.NewEncoder(w).Encode(mrs)
		}
	}))
	defer server.Close()

	config = &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
		SourceBranch: "feature/test",
		TargetBranch: "main",
		UserIDs:      []int{1},
		CreateOnly:   true,
	}

	err = run(config)
	if err == nil {
		t.Error("Expected error for create-only with existing MR")
	}

	// Test update-only with no existing MR
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" && !strings.Contains(r.URL.Path, "merge_requests"):
			project := Project{
				ID:            123,
				Name:          "test-project",
				DefaultBranch: "main",
			}
			json.NewEncoder(w).Encode(project)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			json.NewEncoder(w).Encode([]MergeRequest{})
		}
	}))
	defer server2.Close()

	config = &Config{
		GitLabURL:    server2.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
		SourceBranch: "feature/test",
		TargetBranch: "main",
		UserIDs:      []int{1},
		UpdateMR:     true,
	}

	err = run(config)
	if err == nil {
		t.Error("Expected error for update-only with no existing MR")
	}
}

func TestConfigValidation(t *testing.T) {
	// Test environment variable parsing
	os.Setenv("GITLAB_PRIVATE_TOKEN", "test-token")
	os.Setenv("CI_COMMIT_REF_NAME", "feature/test")
	os.Setenv("CI_PROJECT_ID", "123")
	os.Setenv("CI_PROJECT_URL", "https://gitlab.com/test/repo")
	os.Setenv("GITLAB_USER_ID", "456,789")
	defer func() {
		os.Unsetenv("GITLAB_PRIVATE_TOKEN")
		os.Unsetenv("CI_COMMIT_REF_NAME")
		os.Unsetenv("CI_PROJECT_ID")
		os.Unsetenv("CI_PROJECT_URL")
		os.Unsetenv("GITLAB_USER_ID")
	}()

	// Test environment variable reading
	if getEnv("GITLAB_PRIVATE_TOKEN", "") != "test-token" {
		t.Error("Failed to get environment variable")
	}
	if getEnvInt("CI_PROJECT_ID", 0) != 123 {
		t.Error("Failed to get integer environment variable")
	}

	// Test URL cleaning logic
	testURL := "https://gitlab.com/group/project"
	re := regexp.MustCompile(`^https?://[^/]+`)
	matches := re.FindString(testURL)
	if matches != "https://gitlab.com" {
		t.Errorf("Expected 'https://gitlab.com', got '%s'", matches)
	}

	// Test user ID parsing
	userIDs := parseIntSlice("456,789")
	if len(userIDs) != 2 || userIDs[0] != 456 || userIDs[1] != 789 {
		t.Errorf("Expected [456, 789], got %v", userIDs)
	}
}

func TestErrorHandling(t *testing.T) {
	// Test HTTP error responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
	}

	// Test getProject error handling
	_, err := getProject(client, config)
	if err == nil {
		t.Error("Expected error for HTTP 500 response")
	}

	// Test getExistingMR error handling
	_, err = getExistingMR(client, config)
	if err == nil {
		t.Error("Expected error for HTTP 500 response")
	}
}

func TestRunExistingMRWithoutUpdateFlag(t *testing.T) {
	// Mock server that simulates existing MR
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" && !strings.Contains(r.URL.Path, "merge_requests"):
			project := Project{
				ID:            123,
				Name:          "test-project",
				DefaultBranch: "main",
			}
			json.NewEncoder(w).Encode(project)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			// Return existing MR
			mrs := []MergeRequest{
				{
					ID:           1,
					IID:          1,
					Title:        "Existing MR",
					SourceBranch: "feature/test",
					TargetBranch: "main",
					State:        "opened",
				},
			}
			json.NewEncoder(w).Encode(mrs)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests/1") && r.Method == "PUT":
			// This should not be called
			t.Error("Update MR should not be called when --update-mr flag is not set")
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := &Config{
		GitLabURL:     server.URL,
		ProjectID:     123,
		PrivateToken:  "test-token",
		SourceBranch:  "feature/test",
		TargetBranch:  "main",
		UserIDs:       []int{1},
		UpdateMR:      false, // No update flag
		CommitPrefix:  "Draft",
		RemoveBranch:  false,
		SquashCommits: false,
	}

	err := run(config)
	if err != nil {
		t.Errorf("Expected no error when MR exists without --update-mr flag, got %v", err)
	}
}

func TestRunWithIssueData(t *testing.T) {
	// Mock server that supports issue data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" && !strings.Contains(r.URL.Path, "merge_requests") && !strings.Contains(r.URL.Path, "issues"):
			project := Project{
				ID:            123,
				Name:          "test-project",
				DefaultBranch: "main",
			}
			json.NewEncoder(w).Encode(project)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			json.NewEncoder(w).Encode([]MergeRequest{})
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/issues/456") && r.Method == "GET":
			issue := Issue{
				ID:     456,
				IID:    456,
				Title:  "Test Issue",
				Labels: []string{"bug", "urgent"},
				Milestone: struct {
					ID int `json:"id"`
				}{ID: 10},
			}
			json.NewEncoder(w).Encode(issue)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer server.Close()

	config := &Config{
		GitLabURL:     server.URL,
		ProjectID:     123,
		PrivateToken:  "test-token",
		SourceBranch:  "feature/fix-#456",
		TargetBranch:  "main",
		UserIDs:       []int{1},
		UseIssueName:  true,
		CommitPrefix:  "Fix",
		RemoveBranch:  false,
		SquashCommits: false,
	}

	err := run(config)
	if err != nil {
		t.Errorf("Expected no error for run with issue data, got %v", err)
	}
}
