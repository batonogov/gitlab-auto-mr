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

func TestVersionInfo(t *testing.T) {
	result := versionInfo()
	if !strings.Contains(result, "gitlab-auto-mr") {
		t.Errorf("Expected version info to contain 'gitlab-auto-mr', got '%s'", result)
	}
	if !strings.Contains(result, Version) {
		t.Errorf("Expected version info to contain version '%s', got '%s'", Version, result)
	}
	if !strings.Contains(result, GitCommit) {
		t.Errorf("Expected version info to contain commit '%s', got '%s'", GitCommit, result)
	}
	if !strings.Contains(result, BuildDate) {
		t.Errorf("Expected version info to contain build date '%s', got '%s'", BuildDate, result)
	}
}

func TestVersionInfoDefaults(t *testing.T) {
	result := versionInfo()
	expected := "gitlab-auto-mr dev (commit: none, built: unknown)"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

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
	// Mock server that filters by source_branch and target_branch query params
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sourceBranch := r.URL.Query().Get("source_branch")
		targetBranch := r.URL.Query().Get("target_branch")

		if sourceBranch == "" || targetBranch == "" {
			t.Error("Expected source_branch and target_branch query parameters")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Simulate GitLab API filtering
		allMRs := []MergeRequest{
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

		var filtered []MergeRequest
		for _, mr := range allMRs {
			if mr.SourceBranch == sourceBranch && mr.TargetBranch == targetBranch {
				filtered = append(filtered, mr)
			}
		}

		json.NewEncoder(w).Encode(filtered)
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
		t.Fatal("Expected to find MR, got nil")
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

func TestGetExistingMRServerSideFiltering(t *testing.T) {
	// Verify that the function relies on server-side filtering
	// by returning only filtered results (no local filtering needed)
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		sourceBranch := r.URL.Query().Get("source_branch")
		targetBranch := r.URL.Query().Get("target_branch")

		if sourceBranch != "feature/target" {
			t.Errorf("Expected source_branch 'feature/target', got '%s'", sourceBranch)
		}
		if targetBranch != "main" {
			t.Errorf("Expected target_branch 'main', got '%s'", targetBranch)
		}

		// Return a single matching MR (as GitLab API would after filtering)
		mrs := []MergeRequest{
			{
				ID:           42,
				IID:          42,
				Title:        "Filtered MR",
				SourceBranch: "feature/target",
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
		SourceBranch: "feature/target",
		TargetBranch: "main",
	}

	mr, err := getExistingMR(client, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if mr == nil {
		t.Fatal("Expected to find MR, got nil")
	}
	if mr.IID != 42 {
		t.Errorf("Expected MR IID 42, got %d", mr.IID)
	}
	if requestCount != 1 {
		t.Errorf("Expected exactly 1 API request, got %d", requestCount)
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

	mr, err := createMR(client, config, mrRequest)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if mr == nil {
		t.Fatal("Expected non-nil MR")
	}
	if mr.IID != 1 {
		t.Errorf("Expected MR IID 1, got %d", mr.IID)
	}
	if mr.Title != "Test MR" {
		t.Errorf("Expected MR title 'Test MR', got '%s'", mr.Title)
	}

	// Test failed creation
	mrRequest.SourceBranch = ""
	_, err = createMR(client, config, mrRequest)
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
			json.NewEncoder(w).Encode(MergeRequest{ID: 1, IID: 1, Title: "Test MR"})
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

func TestAcceptMR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/merge_requests/42/merge") {
			t.Errorf("Expected path ending with /merge_requests/42/merge, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["merge_when_pipeline_succeeds"] != true {
			t.Error("Expected merge_when_pipeline_succeeds to be true")
		}
		if body["should_remove_source_branch"] != true {
			t.Error("Expected should_remove_source_branch to be true")
		}
		if body["squash"] != true {
			t.Error("Expected squash to be true")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MergeRequest{ID: 1, IID: 42})
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:     server.URL,
		ProjectID:     123,
		PrivateToken:  "test-token",
		RemoveBranch:  true,
		SquashCommits: true,
	}

	err := acceptMR(client, config, 42)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestAcceptMR405(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
	}

	err := acceptMR(client, config, 42)
	if err == nil {
		t.Error("Expected error for 405 response")
	}
	if !strings.Contains(err.Error(), "cannot be merged") {
		t.Errorf("Expected 'cannot be merged' in error message, got: %v", err)
	}
}

func TestAcceptMR401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "bad-token",
	}

	err := acceptMR(client, config, 42)
	if err == nil {
		t.Error("Expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("Expected 'unauthorized' in error message, got: %v", err)
	}
}

func TestAcceptMR406(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotAcceptable)
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
	}

	err := acceptMR(client, config, 42)
	if err == nil {
		t.Error("Expected error for 406 response")
	}
	if !strings.Contains(err.Error(), "unresolved discussions") {
		t.Errorf("Expected 'unresolved discussions' in error message, got: %v", err)
	}
}

func TestAutoMergeWithMRExistsConflict(t *testing.T) {
	config := &Config{
		AutoMerge: true,
		MRExists:  true,
	}

	err := run(config)
	if err == nil {
		t.Error("Expected error for --auto-merge with --mr-exists")
	}
	if !strings.Contains(err.Error(), "cannot be used with --mr-exists") {
		t.Errorf("Expected conflict error message, got: %v", err)
	}
}

func TestRunWithAutoMerge(t *testing.T) {
	autoMergeCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" && !strings.Contains(r.URL.Path, "merge_requests"):
			project := Project{ID: 123, Name: "test-project", DefaultBranch: "main"}
			json.NewEncoder(w).Encode(project)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			json.NewEncoder(w).Encode([]MergeRequest{})
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(MergeRequest{ID: 1, IID: 10, Title: "Test MR"})
		case strings.Contains(r.URL.Path, "/merge_requests/10/merge") && r.Method == "PUT":
			autoMergeCalled = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(MergeRequest{ID: 1, IID: 10})
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
		AutoMerge:     true,
		CommitPrefix:  "Draft",
		RemoveBranch:  false,
		SquashCommits: false,
	}

	err := run(config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !autoMergeCalled {
		t.Error("Expected auto-merge endpoint to be called")
	}
}

func TestRunWithAutoMergeUpdate(t *testing.T) {
	autoMergeCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" && !strings.Contains(r.URL.Path, "merge_requests"):
			project := Project{ID: 123, Name: "test-project", DefaultBranch: "main"}
			json.NewEncoder(w).Encode(project)
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			mrs := []MergeRequest{{ID: 1, IID: 5, Title: "Existing MR", SourceBranch: "feature/test", TargetBranch: "main", State: "opened"}}
			json.NewEncoder(w).Encode(mrs)
		case r.URL.Path == "/api/v4/projects/123/merge_requests/5/merge" && r.Method == "PUT":
			autoMergeCalled = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(MergeRequest{ID: 1, IID: 5})
		case r.URL.Path == "/api/v4/projects/123/merge_requests/5" && r.Method == "PUT":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
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
		AutoMerge:    true,
		UpdateMR:     true,
		CommitPrefix: "Draft",
	}

	err := run(config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !autoMergeCalled {
		t.Error("Expected auto-merge endpoint to be called after update")
	}
}

func TestCreateMREmptyResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		// Empty body — no JSON
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
	}

	mr, err := createMR(client, config, &MRCreateRequest{
		SourceBranch: "feature/test",
		TargetBranch: "main",
		Title:        "Test MR",
	})
	if err != nil {
		t.Errorf("Expected no error even with empty body, got %v", err)
	}
	if mr == nil {
		t.Fatal("Expected non-nil MR")
	}
	if mr.IID != 0 {
		t.Errorf("Expected zero IID for empty body, got %d", mr.IID)
	}
}

func TestCreateMRInvalidResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("<html>Bad Gateway</html>"))
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
	}

	_, err := createMR(client, config, &MRCreateRequest{
		SourceBranch: "feature/test",
		TargetBranch: "main",
		Title:        "Test MR",
	})
	if err == nil {
		t.Error("Expected error for invalid JSON body")
	}
	if !strings.Contains(err.Error(), "MR created but response is invalid") {
		t.Errorf("Expected 'MR created but response is invalid' in error, got: %v", err)
	}
}

func TestRunWithAutoMergeExistingMRNoUpdate(t *testing.T) {
	autoMergeCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v4/projects/123" && r.Method == "GET":
			json.NewEncoder(w).Encode(Project{ID: 123, Name: "test-project", DefaultBranch: "main"})
		case r.URL.Path == "/api/v4/projects/123/merge_requests" && r.Method == "GET":
			mrs := []MergeRequest{{ID: 1, IID: 7, Title: "Existing MR", SourceBranch: "feature/test", TargetBranch: "main", State: "opened"}}
			json.NewEncoder(w).Encode(mrs)
		case r.URL.Path == "/api/v4/projects/123/merge_requests/7/merge" && r.Method == "PUT":
			autoMergeCalled = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(MergeRequest{ID: 1, IID: 7})
		default:
			w.WriteHeader(http.StatusNotFound)
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
		AutoMerge:    true,
		UpdateMR:     false,
		CommitPrefix: "Draft",
	}

	err := run(config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !autoMergeCalled {
		t.Error("Expected auto-merge endpoint to be called even without --update-mr")
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
			json.NewEncoder(w).Encode(MergeRequest{ID: 1, IID: 1, Title: "Test MR"})
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
