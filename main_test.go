package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net"
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
		RemoveSourceBranch: boolPtr(true),
		Squash:             boolPtr(true),
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

	// Pointer bool fields must be explicitly serialized, even when false,
	// so that --update-mr can turn squash / remove-source-branch OFF.
	// See issue #66: omitempty on bool previously dropped false values.
	cases := []struct {
		name     string
		remove   bool
		squash   bool
		wantRm   string
		wantSq   string
		wantNull bool
	}{
		{name: "true", remove: true, squash: true, wantRm: `"remove_source_branch":true`, wantSq: `"squash":true`},
		{name: "false", remove: false, squash: false, wantRm: `"remove_source_branch":false`, wantSq: `"squash":false`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := MRUpdateRequest{
				RemoveSourceBranch: boolPtr(tc.remove),
				Squash:             boolPtr(tc.squash),
			}
			data, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}
			body := string(data)
			if !strings.Contains(body, tc.wantRm) {
				t.Errorf("%s: expected JSON to contain %s, got %s", tc.name, tc.wantRm, body)
			}
			if !strings.Contains(body, tc.wantSq) {
				t.Errorf("%s: expected JSON to contain %s, got %s", tc.name, tc.wantSq, body)
			}
		})
	}

	// nil pointer means "don't send" and must be omitted.
	nilReq := MRUpdateRequest{}
	nilData, err := json.Marshal(nilReq)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	if strings.Contains(string(nilData), "remove_source_branch") {
		t.Errorf("nil pointer should be omitted, got %s", string(nilData))
	}
	if strings.Contains(string(nilData), "squash") {
		t.Errorf("nil pointer should be omitted, got %s", string(nilData))
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
		switch {
		case config.UpdateMR:
			mode = "update"
		case config.CreateOnly:
			mode = "create"
		default:
			mode = "smart"
		}

		if mode != test.expected {
			t.Errorf("%s: expected '%s', got '%s'", test.name, test.expected, mode)
		}
	}
}

func TestCreateHTTPClient(t *testing.T) {
	// Test secure client
	client := createHTTPClient(&Config{})
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.Timeout)
	}

	// Test insecure client
	insecureClient := createHTTPClient(&Config{Insecure: true})
	if insecureClient == nil {
		t.Fatal("Expected non-nil insecure client")
	}
	if insecureClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", insecureClient.Timeout)
	}

	// An explicit --timeout wins; a zero value keeps the 30s default, so a
	// Config built without one behaves as it always did.
	custom := createHTTPClient(&Config{Timeout: 5 * time.Second})
	if custom.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", custom.Timeout)
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
	project, err := getProject(context.Background(), client, config)
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
	_, err = getProject(context.Background(), client, config)
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
	mr, err := getExistingMR(context.Background(), client, config)
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
	mr, err = getExistingMR(context.Background(), client, config)
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

	mr, err := getExistingMR(context.Background(), client, config)
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

func TestGetExistingMRSpecialChars(t *testing.T) {
	// Branch names containing special chars (+, #, &, spaces) must be URL-encoded
	// so the server decodes them back to the exact original values, and per_page=1
	// must be present (issue #65 + #70).
	const (
		sourceBranch = "feature/fix-#123 & x+more"
		targetBranch = "release/v1.0 #2"
	)

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		query := r.URL.Query()

		// per_page=1 must be requested explicitly
		if got := query.Get("per_page"); got != "1" {
			t.Errorf("Expected per_page=1, got %q", got)
		}

		// Decoded branch values must match the originals exactly
		if got := query.Get("source_branch"); got != sourceBranch {
			t.Errorf("Expected source_branch %q, got %q", sourceBranch, got)
		}
		if got := query.Get("target_branch"); got != targetBranch {
			t.Errorf("Expected target_branch %q, got %q", targetBranch, got)
		}

		mrs := []MergeRequest{
			{
				ID:           7,
				IID:          7,
				Title:        "Special chars MR",
				SourceBranch: sourceBranch,
				TargetBranch: targetBranch,
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
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
	}

	mr, err := getExistingMR(context.Background(), client, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if mr == nil {
		t.Fatal("Expected to find MR, got nil")
	}
	if mr.IID != 7 {
		t.Errorf("Expected MR IID 7, got %d", mr.IID)
	}
	if mr.SourceBranch != sourceBranch {
		t.Errorf("Expected MR source branch %q, got %q", sourceBranch, mr.SourceBranch)
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
	issue, err := getIssueData(context.Background(), client, config)
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
	_, err = getIssueData(context.Background(), client, config)
	if err == nil {
		t.Error("Expected error for invalid branch name")
	}

	// Test invalid issue number
	config.SourceBranch = "feature/fix-#invalid"
	_, err = getIssueData(context.Background(), client, config)
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

	mr, err := createMR(context.Background(), client, config, mrRequest)
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
	_, err = createMR(context.Background(), client, config, mrRequest)
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

	err := updateMR(context.Background(), client, config, 1, updateRequest)
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

	err := run(context.Background(), config)
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

	err := run(context.Background(), config)
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

	err := run(context.Background(), config)
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

	err := run(context.Background(), config)
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

	err = run(context.Background(), config)
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

	err = run(context.Background(), config)
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
	_, err := getProject(context.Background(), client, config)
	if err == nil {
		t.Error("Expected error for HTTP 500 response")
	}

	// Test getExistingMR error handling
	_, err = getExistingMR(context.Background(), client, config)
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

	err := run(context.Background(), config)
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

	err := acceptMR(context.Background(), client, config, 42)
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

	err := acceptMR(context.Background(), client, config, 42)
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

	err := acceptMR(context.Background(), client, config, 42)
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

	err := acceptMR(context.Background(), client, config, 42)
	if err == nil {
		t.Error("Expected error for 406 response")
	}
	if !strings.Contains(err.Error(), "unresolved discussions") {
		t.Errorf("Expected 'unresolved discussions' in error message, got: %v", err)
	}
}

func TestIsDraftPrefix(t *testing.T) {
	tests := []struct {
		prefix   string
		expected bool
	}{
		{"Draft", true},
		{"draft", true},
		{"DRAFT", true},
		{"WIP", true},
		{"wip", true},
		{" Draft ", true},
		{"", false},
		{"Fix", false},
		{"Feature", false},
		{"Drafty", false},
	}

	for _, test := range tests {
		result := isDraftPrefix(test.prefix)
		if result != test.expected {
			t.Errorf("isDraftPrefix(%q): expected %v, got %v", test.prefix, test.expected, result)
		}
	}
}

func TestAutoMergeWithDraftPrefixConflict(t *testing.T) {
	config := &Config{
		AutoMerge:    true,
		CommitPrefix: "Draft",
	}

	err := run(context.Background(), config)
	if err == nil {
		t.Error("Expected error for --auto-merge with draft prefix")
	}
	if !strings.Contains(err.Error(), "cannot be used with --commit-prefix") {
		t.Errorf("Expected conflict error message, got: %v", err)
	}

	// Also test with WIP prefix
	config.CommitPrefix = "WIP"
	err = run(context.Background(), config)
	if err == nil {
		t.Error("Expected error for --auto-merge with WIP prefix")
	}
}

func TestAutoMergeWithMRExistsConflict(t *testing.T) {
	config := &Config{
		AutoMerge: true,
		MRExists:  true,
	}

	err := run(context.Background(), config)
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
		CommitPrefix:  "",
		RemoveBranch:  false,
		SquashCommits: false,
	}

	err := run(context.Background(), config)
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
		CommitPrefix: "",
	}

	err := run(context.Background(), config)
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

	mr, err := createMR(context.Background(), client, config, &MRCreateRequest{
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

	_, err := createMR(context.Background(), client, config, &MRCreateRequest{
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
		CommitPrefix: "",
	}

	err := run(context.Background(), config)
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

	err := run(context.Background(), config)
	if err != nil {
		t.Errorf("Expected no error for run with issue data, got %v", err)
	}
}

// TestRunWithIssueDataError verifies that when --use-issue-name is set but the
// issue lookup fails (HTTP 404), a warning is printed to stderr and the MR is
// still created without issue data (backward-compatible, control flow unchanged).
func TestRunWithIssueDataError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" &&
			!strings.Contains(r.URL.Path, "merge_requests") && !strings.Contains(r.URL.Path, "issues"):
			json.NewEncoder(w).Encode(Project{
				ID:            123,
				Name:          "test-project",
				DefaultBranch: "main",
			})
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			json.NewEncoder(w).Encode([]MergeRequest{})
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/issues/789") && r.Method == "GET":
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "404 Not Found"})
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
		SourceBranch:  "feature/fix-#789",
		TargetBranch:  "main",
		UserIDs:       []int{1},
		UseIssueName:  true,
		CommitPrefix:  "Fix",
		RemoveBranch:  false,
		SquashCommits: false,
	}

	// Capture os.Stderr to assert the warning is emitted there (not stdout).
	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	origStderr := os.Stderr
	os.Stderr = wPipe

	t.Cleanup(func() {
		os.Stderr = origStderr
	})

	runErr := run(context.Background(), config)

	if cerr := wPipe.Close(); cerr != nil {
		t.Fatalf("Failed to close stderr pipe: %v", cerr)
	}

	stderrBytes, err := io.ReadAll(rPipe)
	if err != nil {
		t.Fatalf("Failed to read captured stderr: %v", err)
	}
	stderrOutput := string(stderrBytes)

	// The MR must still be created despite the issue lookup failure.
	if runErr != nil {
		t.Errorf("Expected run to succeed even when issue data fetch fails, got error: %v", runErr)
	}

	// The failure must be surfaced as a warning on stderr.
	if !strings.Contains(stderrOutput, "Warning: failed to fetch issue data") {
		t.Errorf("Expected warning about failed issue data fetch on stderr, got: %q", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "issue #789 not found") {
		t.Errorf("Expected underlying error 'issue #789 not found' in warning, got: %q", stderrOutput)
	}
}

func TestParseStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"single", "bug", []string{"bug"}},
		{"multiple", "bug,priority::high", []string{"bug", "priority::high"}},
		{"trims spaces", " bug , needs-review ", []string{"bug", "needs-review"}},
		{"drops empty elements", "bug,,, ,review", []string{"bug", "review"}},
		{"only separators", ",,,", nil},
		{"keeps inner spaces", "needs review,done", []string{"needs review", "done"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStringSlice(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("parseStringSlice(%q) = %v, want %v", tt.input, got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("parseStringSlice(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestMergeLabels(t *testing.T) {
	tests := []struct {
		name     string
		base     []string
		extra    []string
		expected []string
	}{
		{"both empty", nil, nil, nil},
		{"empty non-nil base stays nil", []string{}, nil, nil},
		{"only base", []string{"bug"}, nil, []string{"bug"}},
		{"only extra", nil, []string{"issue-label"}, []string{"issue-label"}},
		{"union", []string{"bug"}, []string{"backend"}, []string{"bug", "backend"}},
		{"deduplicates", []string{"bug", "backend"}, []string{"backend", "ci"}, []string{"bug", "backend", "ci"}},
		{"base order first", []string{"z", "a"}, []string{"a", "m"}, []string{"z", "a", "m"}},
		{"case sensitive", []string{"Bug"}, []string{"bug"}, []string{"Bug", "bug"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeLabels(tt.base, tt.extra)
			if len(got) != len(tt.expected) {
				t.Fatalf("mergeLabels(%v, %v) = %v, want %v", tt.base, tt.extra, got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("mergeLabels(%v, %v)[%d] = %q, want %q", tt.base, tt.extra, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

// newIssueMetadataServer serves a project, no existing MR, and an issue carrying
// milestone 99 and labels ["from-issue", "shared"]. It records the create request.
func newIssueMetadataServer(t *testing.T, captured *MRCreateRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v4/projects/123" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(Project{ID: 123, DefaultBranch: "main"}); err != nil {
				t.Errorf("encode project: %v", err)
			}
		case r.URL.Path == "/api/v4/projects/123/merge_requests" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte("[]")); err != nil {
				t.Errorf("write empty MR list: %v", err)
			}
		case r.URL.Path == "/api/v4/projects/123/issues/42" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(
				`{"id":1,"iid":42,"title":"Issue","labels":["from-issue","shared"],"milestone":{"id":99}}`,
			)); err != nil {
				t.Errorf("write issue: %v", err)
			}
		case r.URL.Path == "/api/v4/projects/123/merge_requests" && r.Method == "POST":
			if err := json.NewDecoder(r.Body).Decode(captured); err != nil {
				t.Errorf("decode create request: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(MergeRequest{ID: 1, IID: 7}); err != nil {
				t.Errorf("encode MR: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// TestRunLabelAndMilestone covers issues #73 and #74: labels and a milestone can
// be set from the command line, they combine with --use-issue-name rather than
// excluding it, and an explicit milestone wins over the issue's.
func TestRunLabelAndMilestone(t *testing.T) {
	tests := []struct {
		name             string
		labels           []string
		milestoneID      int
		useIssueName     bool
		expectedLabels   []string
		expectedMileston int
	}{
		{
			name:             "flags only",
			labels:           []string{"bug", "priority::high"},
			milestoneID:      5,
			expectedLabels:   []string{"bug", "priority::high"},
			expectedMileston: 5,
		},
		{
			name:             "issue only",
			useIssueName:     true,
			expectedLabels:   []string{"from-issue", "shared"},
			expectedMileston: 99,
		},
		{
			name:             "labels merge with issue labels",
			labels:           []string{"bug", "shared"},
			useIssueName:     true,
			expectedLabels:   []string{"bug", "shared", "from-issue"},
			expectedMileston: 99,
		},
		{
			name:             "explicit milestone beats issue milestone",
			milestoneID:      5,
			useIssueName:     true,
			expectedLabels:   []string{"from-issue", "shared"},
			expectedMileston: 5,
		},
		{
			name:             "neither",
			expectedLabels:   nil,
			expectedMileston: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured MRCreateRequest
			server := newIssueMetadataServer(t, &captured)
			defer server.Close()

			config := &Config{
				PrivateToken: "test-token",
				SourceBranch: "feature/#42-test",
				ProjectID:    123,
				GitLabURL:    server.URL,
				UserIDs:      []int{1},
				TargetBranch: "main",
				Labels:       tt.labels,
				MilestoneID:  tt.milestoneID,
				UseIssueName: tt.useIssueName,
			}

			if err := run(context.Background(), config); err != nil {
				t.Fatalf("run() error = %v", err)
			}

			if captured.MilestoneID != tt.expectedMileston {
				t.Errorf("MilestoneID = %d, want %d", captured.MilestoneID, tt.expectedMileston)
			}
			if len(captured.Labels) != len(tt.expectedLabels) {
				t.Fatalf("Labels = %v, want %v", captured.Labels, tt.expectedLabels)
			}
			for i := range captured.Labels {
				if captured.Labels[i] != tt.expectedLabels[i] {
					t.Errorf("Labels[%d] = %q, want %q", i, captured.Labels[i], tt.expectedLabels[i])
				}
			}
		})
	}
}

// captureOutput runs fn with os.Stdout redirected to a pipe and returns what it
// wrote. The pipe is drained on a goroutine so a write larger than the pipe
// buffer cannot deadlock.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		if _, cerr := io.Copy(&buf, r); cerr != nil {
			t.Errorf("drain pipe: %v", cerr)
		}
		done <- buf.String()
	}()

	fn()

	os.Stdout = orig
	if cerr := w.Close(); cerr != nil {
		t.Errorf("close pipe writer: %v", cerr)
	}
	out := <-done
	if cerr := r.Close(); cerr != nil {
		t.Errorf("close pipe reader: %v", cerr)
	}
	return out
}

// TestPrintMRURL covers issue #75: the MR's browser URL is printed after every
// operation that identifies an MR, and nothing is printed when GitLab did not
// return one.
func TestPrintMRURL(t *testing.T) {
	const webURL = "https://gitlab.example.com/group/proj/-/merge_requests/42"

	tests := []struct {
		name      string
		existing  string // JSON body for the MR list endpoint
		createdMR string // JSON body for the create response
		config    func(c *Config)
		wantURL   bool
	}{
		{
			name:      "created",
			existing:  "[]",
			createdMR: `{"id":1,"iid":42,"web_url":"` + webURL + `"}`,
			wantURL:   true,
		},
		{
			name:      "created without web_url",
			existing:  "[]",
			createdMR: `{"id":1,"iid":42}`,
			wantURL:   false,
		},
		{
			name:     "updated",
			existing: `[{"id":1,"iid":42,"title":"Old","web_url":"` + webURL + `"}]`,
			config:   func(c *Config) { c.UpdateMR = true },
			wantURL:  true,
		},
		{
			name:     "exists without update flag",
			existing: `[{"id":1,"iid":42,"title":"Old","web_url":"` + webURL + `"}]`,
			wantURL:  true,
		},
		{
			name:     "dry run",
			existing: `[{"id":1,"iid":42,"title":"Old","web_url":"` + webURL + `"}]`,
			config:   func(c *Config) { c.MRExists = true },
			wantURL:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.URL.Path == "/api/v4/projects/123" && r.Method == "GET":
					if err := json.NewEncoder(w).Encode(Project{ID: 123, DefaultBranch: "main"}); err != nil {
						t.Errorf("encode project: %v", err)
					}
				case r.URL.Path == "/api/v4/projects/123/merge_requests" && r.Method == "GET":
					if _, err := w.Write([]byte(tt.existing)); err != nil {
						t.Errorf("write MR list: %v", err)
					}
				case r.URL.Path == "/api/v4/projects/123/merge_requests" && r.Method == "POST":
					w.WriteHeader(http.StatusCreated)
					if _, err := w.Write([]byte(tt.createdMR)); err != nil {
						t.Errorf("write created MR: %v", err)
					}
				case r.URL.Path == "/api/v4/projects/123/merge_requests/42" && r.Method == "PUT":
					if _, err := w.Write([]byte(`{}`)); err != nil {
						t.Errorf("write update response: %v", err)
					}
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			config := &Config{
				PrivateToken: "test-token",
				SourceBranch: "feature/test",
				ProjectID:    123,
				GitLabURL:    server.URL,
				UserIDs:      []int{1},
				TargetBranch: "main",
			}
			if tt.config != nil {
				tt.config(config)
			}

			var runErr error
			out := captureOutput(t, func() { runErr = run(context.Background(), config) })
			if runErr != nil {
				t.Fatalf("run() error = %v", runErr)
			}

			gotURL := strings.Contains(out, "MR URL: "+webURL)
			if gotURL != tt.wantURL {
				t.Errorf("URL printed = %v, want %v; output was:\n%s", gotURL, tt.wantURL, out)
			}
			if !tt.wantURL && strings.Contains(out, "MR URL:") {
				t.Errorf("expected no MR URL line, output was:\n%s", out)
			}
		})
	}
}

// pipelineServer mocks the two pipeline endpoints and records how often the MR
// pipeline was created. existingSHA, when non-empty, is the SHA of a pipeline
// GitLab already has for the MR.
func pipelineServer(t *testing.T, existingSHA string, listStatus int) (*httptest.Server, *int) {
	t.Helper()

	created := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/projects/123/merge_requests/42/pipelines" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			if listStatus != 0 && listStatus != http.StatusOK {
				w.WriteHeader(listStatus)
				return
			}
			body := "[]"
			if existingSHA != "" {
				body = `[{"id":7,"status":"running","sha":"` + existingSHA +
					`","web_url":"https://gitlab.example.com/p/-/pipelines/7"}]`
			}
			if _, err := w.Write([]byte(body)); err != nil {
				t.Errorf("write pipeline list: %v", err)
			}
		case http.MethodPost:
			created++
			w.WriteHeader(http.StatusCreated)
			if _, err := w.Write([]byte(`{"id":9,"status":"created"}`)); err != nil {
				t.Errorf("write created pipeline: %v", err)
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	return server, &created
}

// TestTriggerMRPipelineDeduplicates covers issue #151: a pipeline is not created
// again for a commit that already has one, unless --force-pipeline is given.
func TestTriggerMRPipelineDeduplicates(t *testing.T) {
	const headSHA = "abc123def456"

	tests := []struct {
		name        string
		existingSHA string
		mrSHA       string
		force       bool
		listStatus  int
		wantCreated int
	}{
		{
			name:        "skips when a pipeline exists for the head commit",
			existingSHA: headSHA,
			mrSHA:       headSHA,
			wantCreated: 0,
		},
		{
			name:        "creates when the existing pipeline is for an older commit",
			existingSHA: "0000000000",
			mrSHA:       headSHA,
			wantCreated: 1,
		},
		{
			name:        "creates when the MR has no pipelines",
			mrSHA:       headSHA,
			wantCreated: 1,
		},
		{
			name:        "--force-pipeline creates anyway",
			existingSHA: headSHA,
			mrSHA:       headSHA,
			force:       true,
			wantCreated: 1,
		},
		{
			name:        "creates when the head SHA is unknown",
			existingSHA: headSHA,
			mrSHA:       "",
			wantCreated: 1,
		},
		{
			name:        "creates when the pipeline list cannot be read",
			existingSHA: headSHA,
			mrSHA:       headSHA,
			listStatus:  http.StatusInternalServerError,
			wantCreated: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, created := pipelineServer(t, tt.existingSHA, tt.listStatus)
			defer server.Close()

			config := &Config{
				PrivateToken:    "test-token",
				ProjectID:       123,
				GitLabURL:       server.URL,
				TriggerPipeline: true,
				ForcePipeline:   tt.force,
			}

			err := triggerMRPipeline(context.Background(), server.Client(), config, &MergeRequest{IID: 42, SHA: tt.mrSHA})
			if err != nil {
				t.Fatalf("triggerMRPipeline() error = %v", err)
			}
			if *created != tt.wantCreated {
				t.Errorf("pipelines created = %d, want %d", *created, tt.wantCreated)
			}
		})
	}
}

// TestRunTriggersPipelineOnUpdate covers issue #150: updating an existing MR
// triggers a pipeline too, since the branch has moved. The MR carries a SHA
// that has no pipeline yet, which is the case where a new one is wanted.
func TestRunTriggersPipelineOnUpdate(t *testing.T) {
	created := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v4/projects/123" && r.Method == http.MethodGet:
			if err := json.NewEncoder(w).Encode(Project{ID: 123, DefaultBranch: "main"}); err != nil {
				t.Errorf("encode project: %v", err)
			}
		case r.URL.Path == "/api/v4/projects/123/merge_requests" && r.Method == http.MethodGet:
			if _, err := w.Write([]byte(`[{"id":1,"iid":42,"title":"Old","sha":"newsha"}]`)); err != nil {
				t.Errorf("write MR list: %v", err)
			}
		case r.URL.Path == "/api/v4/projects/123/merge_requests/42" && r.Method == http.MethodPut:
			if _, err := w.Write([]byte(`{"id":1,"iid":42}`)); err != nil {
				t.Errorf("write update response: %v", err)
			}
		case r.URL.Path == "/api/v4/projects/123/merge_requests/42/pipelines" && r.Method == http.MethodGet:
			if _, err := w.Write([]byte(`[{"id":7,"status":"success","sha":"oldsha"}]`)); err != nil {
				t.Errorf("write pipeline list: %v", err)
			}
		case r.URL.Path == "/api/v4/projects/123/merge_requests/42/pipelines" && r.Method == http.MethodPost:
			created++
			w.WriteHeader(http.StatusCreated)
			if _, err := w.Write([]byte(`{"id":9,"status":"created"}`)); err != nil {
				t.Errorf("write created pipeline: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &Config{
		PrivateToken:    "test-token",
		SourceBranch:    "feature/test",
		ProjectID:       123,
		GitLabURL:       server.URL,
		UserIDs:         []int{1},
		TargetBranch:    "main",
		UpdateMR:        true,
		TriggerPipeline: true,
	}

	if err := run(context.Background(), config); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if created != 1 {
		t.Errorf("pipelines created on update = %d, want 1", created)
	}
}

// TestForcePipelineRequiresTriggerPipeline checks the flag is refused on its
// own rather than silently doing nothing.
func TestForcePipelineRequiresTriggerPipeline(t *testing.T) {
	err := validateConfig(&Config{ForcePipeline: true})
	if err == nil {
		t.Fatal("expected an error for --force-pipeline without --trigger-pipeline")
	}
	if !strings.Contains(err.Error(), "--force-pipeline has no effect without --trigger-pipeline") {
		t.Errorf("error = %v", err)
	}
}

func TestShortSHA(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"abc", "abc"},
		{"abcdef12", "abcdef12"},
		{"abcdef1234567890", "abcdef12"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := shortSHA(tt.in); got != tt.want {
				t.Errorf("shortSHA(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestAPIErrorMessage pins the wording apiError produces, since it is what
// every API function reports for an unexpected status.
func TestAPIErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  *apiError
		want string
	}{
		{"with body", &apiError{StatusCode: 400, Body: "bad request"}, "HTTP 400: bad request"},
		{"empty body", &apiError{StatusCode: 500}, "HTTP 500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDoRequestUnauthorized checks that a 401 from any endpoint produces the
// single shared errUnauthorized, so callers can match it with errors.Is.
func TestDoRequestUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	config := &Config{PrivateToken: "bad", ProjectID: 123, GitLabURL: server.URL}

	_, err := doRequest(context.Background(), server.Client(), config, http.MethodGet, "projects/123", nil)
	if !errors.Is(err, errUnauthorized) {
		t.Errorf("expected errUnauthorized, got %v", err)
	}
}

// TestDoRequestSendsTokenAndBody checks the three things the helper is
// responsible for on the way out: the URL, the token header, and the JSON body
// with its Content-Type. A GET must send neither body nor Content-Type.
func TestDoRequestSendsTokenAndBody(t *testing.T) {
	var (
		gotPath        string
		gotToken       string
		gotContentType string
		gotBody        string
		gotMethod      string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("PRIVATE-TOKEN")
		gotContentType = r.Header.Get("Content-Type")
		gotMethod = r.Method
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		gotBody = string(body)
		if _, err := w.Write([]byte(`{"ok":true}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	config := &Config{PrivateToken: "test-token", ProjectID: 123, GitLabURL: server.URL}

	t.Run("post with body", func(t *testing.T) {
		resp, err := doRequest(context.Background(), server.Client(), config,
			http.MethodPost, "projects/123/merge_requests", map[string]string{"title": "x"})
		if err != nil {
			t.Fatalf("doRequest() error = %v", err)
		}
		if string(resp) != `{"ok":true}` {
			t.Errorf("body = %q, want %q", string(resp), `{"ok":true}`)
		}
		if gotMethod != http.MethodPost {
			t.Errorf("method = %q, want POST", gotMethod)
		}
		if gotPath != "/api/v4/projects/123/merge_requests" {
			t.Errorf("path = %q", gotPath)
		}
		if gotToken != "test-token" {
			t.Errorf("PRIVATE-TOKEN = %q, want %q", gotToken, "test-token")
		}
		if gotContentType != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", gotContentType)
		}
		if gotBody != `{"title":"x"}` {
			t.Errorf("request body = %q, want %q", gotBody, `{"title":"x"}`)
		}
	})

	t.Run("get without body", func(t *testing.T) {
		if _, err := doRequest(context.Background(), server.Client(), config,
			http.MethodGet, "projects/123", nil); err != nil {
			t.Fatalf("doRequest() error = %v", err)
		}
		if gotBody != "" {
			t.Errorf("GET sent a body: %q", gotBody)
		}
		if gotContentType != "" {
			t.Errorf("GET sent Content-Type: %q", gotContentType)
		}
	})
}

// TestRunContextCancellation covers issue #69: a canceled context aborts the
// request in flight rather than waiting out the client timeout.
func TestRunContextCancellation(t *testing.T) {
	released := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hold the response until the client gives up on it.
		select {
		case <-released:
		case <-r.Context().Done():
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer func() {
		close(released)
		server.Close()
	}()

	config := &Config{
		PrivateToken: "test-token",
		SourceBranch: "feature/test",
		ProjectID:    123,
		GitLabURL:    server.URL,
		UserIDs:      []int{1},
		TargetBranch: "main",
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := run(ctx, config)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from the canceled request, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	// The client timeout is 30s; returning anywhere near it means cancellation
	// was ignored.
	if elapsed > 5*time.Second {
		t.Errorf("run() took %v, expected it to abort as soon as the context was canceled", elapsed)
	}
}

// TestIsRetryable pins the retry policy: idempotent verbs may repeat on
// transient answers, POST may repeat only when the request never left.
func TestIsRetryable(t *testing.T) {
	dialErr := &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	readErr := &net.OpError{Op: "read", Err: errors.New("connection reset by peer")}

	tests := []struct {
		name   string
		method string
		err    error
		want   bool
	}{
		{"GET 500", http.MethodGet, &apiError{StatusCode: 500}, true},
		{"GET 503", http.MethodGet, &apiError{StatusCode: 503}, true},
		{"GET 429", http.MethodGet, &apiError{StatusCode: 429}, true},
		{"GET 404", http.MethodGet, &apiError{StatusCode: 404}, false},
		{"GET 400", http.MethodGet, &apiError{StatusCode: 400}, false},
		{"PUT 502", http.MethodPut, &apiError{StatusCode: 502}, true},
		{"POST 500 is not repeated", http.MethodPost, &apiError{StatusCode: 500}, false},
		{"GET dial error", http.MethodGet, dialErr, true},
		{"GET read error", http.MethodGet, readErr, true},
		{"POST dial error", http.MethodPost, dialErr, true},
		{"POST read error is not repeated", http.MethodPost, readErr, false},
		{"unauthorized", http.MethodGet, errUnauthorized, false},
		{"canceled", http.MethodGet, context.Canceled, false},
		{"deadline exceeded", http.MethodGet, context.DeadlineExceeded, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryable(tt.method, tt.err); got != tt.want {
				t.Errorf("isRetryable(%s, %v) = %v, want %v", tt.method, tt.err, got, tt.want)
			}
		})
	}
}

func TestRetryDelay(t *testing.T) {
	config := &Config{RetryDelay: time.Second}

	tests := []struct {
		name       string
		config     *Config
		attempt    int
		retryAfter time.Duration
		want       time.Duration
	}{
		{"first attempt", config, 0, 0, time.Second},
		{"doubles", config, 1, 0, 2 * time.Second},
		{"doubles again", config, 2, 0, 4 * time.Second},
		{"capped", config, 20, 0, maxRetryDelay},
		{"no overflow", config, 64, 0, maxRetryDelay},
		{"zero delay falls back to the default", &Config{}, 0, 0, defaultRetryDelay},
		{"Retry-After wins", config, 3, 7 * time.Second, 7 * time.Second},
		{"Retry-After is capped too", config, 0, time.Hour, maxRetryDelay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retryDelay(tt.config, tt.attempt, tt.retryAfter); got != tt.want {
				t.Errorf("retryDelay(attempt=%d, retryAfter=%s) = %s, want %s",
					tt.attempt, tt.retryAfter, got, tt.want)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		value string
		want  time.Duration
	}{
		{"", 0},
		{"5", 5 * time.Second},
		{" 12 ", 12 * time.Second},
		{"0", 0},
		{"-1", 0},
		{"Wed, 21 Oct 2015 07:28:00 GMT", 0},
		{"nonsense", 0},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := parseRetryAfter(tt.value); got != tt.want {
				t.Errorf("parseRetryAfter(%q) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}
}

// TestDoRequestRetries covers issue #145 end-to-end against a server that fails
// a fixed number of times before succeeding.
func TestDoRequestRetries(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		failures     int
		failStatus   int
		retries      int
		wantAttempts int
		wantErr      bool
	}{
		{"GET recovers from 503", http.MethodGet, 2, 503, 2, 3, false},
		{"GET gives up after the last retry", http.MethodGet, 5, 503, 2, 3, true},
		{"GET does not retry 404", http.MethodGet, 5, 404, 2, 1, true},
		{"POST does not retry 503", http.MethodPost, 5, 503, 2, 1, true},
		{"no retries configured", http.MethodGet, 1, 500, 0, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attempts int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempts++
				if attempts <= tt.failures {
					w.WriteHeader(tt.failStatus)
					return
				}
				if _, err := w.Write([]byte(`{"ok":true}`)); err != nil {
					t.Errorf("write response: %v", err)
				}
			}))
			defer server.Close()

			config := &Config{
				PrivateToken: "test-token",
				ProjectID:    123,
				GitLabURL:    server.URL,
				Retries:      tt.retries,
				RetryDelay:   time.Millisecond,
			}

			var reqBody any
			if tt.method == http.MethodPost {
				reqBody = map[string]string{"title": "x"}
			}

			_, err := doRequest(context.Background(), server.Client(), config, tt.method, "projects/123", reqBody)

			if tt.wantErr && err == nil {
				t.Error("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if attempts != tt.wantAttempts {
				t.Errorf("attempts = %d, want %d", attempts, tt.wantAttempts)
			}
		})
	}
}

// TestDoRequestRespectsRetryAfter checks that a 429 carrying Retry-After waits
// for the interval GitLab asked for rather than the configured backoff.
func TestDoRequestRespectsRetryAfter(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if _, err := w.Write([]byte(`{}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	config := &Config{
		PrivateToken: "test-token",
		ProjectID:    123,
		GitLabURL:    server.URL,
		Retries:      1,
		RetryDelay:   time.Millisecond, // far shorter than the Retry-After
	}

	start := time.Now()
	if _, err := doRequest(context.Background(), server.Client(), config,
		http.MethodGet, "projects/123", nil); err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	elapsed := time.Since(start)

	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
	if elapsed < time.Second {
		t.Errorf("waited %s, expected at least the 1s from Retry-After", elapsed)
	}
}

// TestDoRequestRetrySendsBodyAgain guards against the reader being consumed by
// the first attempt: a retried PUT must send the same payload.
func TestDoRequestRetrySendsBodyAgain(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		bodies = append(bodies, string(body))
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		if _, werr := w.Write([]byte(`{}`)); werr != nil {
			t.Errorf("write response: %v", werr)
		}
	}))
	defer server.Close()

	config := &Config{
		PrivateToken: "test-token",
		ProjectID:    123,
		GitLabURL:    server.URL,
		Retries:      1,
		RetryDelay:   time.Millisecond,
	}

	if _, err := doRequest(context.Background(), server.Client(), config,
		http.MethodPut, "projects/123/merge_requests/1", map[string]string{"title": "x"}); err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}

	if len(bodies) != 2 {
		t.Fatalf("got %d attempts, want 2", len(bodies))
	}
	if bodies[0] != bodies[1] {
		t.Errorf("retry sent %q, first attempt sent %q", bodies[1], bodies[0])
	}
}

// TestSleepRespectsContext checks that a canceled context ends the backoff wait
// immediately instead of blocking for the full delay.
func TestSleepRespectsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := sleep(ctx, time.Hour)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("sleep returned after %s, expected it to be immediate", elapsed)
	}
}

// resetFlagSet replaces the package-level flag.CommandLine with a fresh FlagSet
// using ContinueOnError. parseFlags re-registers every flag on flag.CommandLine,
// so without a reset the second subtest would panic with "flag redefined".
// ContinueOnError also prevents a parse error from os.Exit-ing the test binary.
// The original FlagSet is restored on cleanup so other tests are unaffected.
func resetFlagSet(t *testing.T) {
	t.Helper()
	saved := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	t.Cleanup(func() { flag.CommandLine = saved })
}

// clearRequiredParseEnv clears the five env vars that parseFlags reads as flag
// defaults. getEnv/getEnvInt treat an empty value identically to an unset
// variable (both yield the default), so setting each to "" reliably clears it.
// t.Setenv restores the previous value on cleanup automatically.
func clearRequiredParseEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"GITLAB_PRIVATE_TOKEN",
		"CI_COMMIT_REF_NAME",
		"CI_PROJECT_ID",
		"CI_PROJECT_URL",
		"GITLAB_USER_ID",
		"GITLAB_AUTO_MR_TARGET_BRANCH",
		"GITLAB_AUTO_MR_LABELS",
		"GITLAB_AUTO_MR_MILESTONE",
	} {
		t.Setenv(key, "")
	}
}

// setRequiredParseEnv sets all five required env vars to known-good values.
func setRequiredParseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GITLAB_PRIVATE_TOKEN", "tok")
	t.Setenv("CI_COMMIT_REF_NAME", "feat/x")
	t.Setenv("CI_PROJECT_ID", "42")
	t.Setenv("CI_PROJECT_URL", "https://gl.example.com/group/proj")
	t.Setenv("GITLAB_USER_ID", "7")
}

// setParseArgs temporarily replaces os.Args so parseFlags parses the given args.
func setParseArgs(t *testing.T, args []string) {
	t.Helper()
	saved := os.Args
	os.Args = args
	t.Cleanup(func() { os.Args = saved })
}

// captureStdout redirects os.Stdout to a discarded pipe for the test lifetime
// so versionInfo() output does not pollute `go test -v` output.
func captureStdout(t *testing.T) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = orig
		if cerr := w.Close(); cerr != nil {
			t.Logf("close pipe writer: %v", cerr)
		}
		if _, err := io.Copy(io.Discard, r); err != nil {
			t.Logf("drain pipe reader: %v", err)
		}
		if cerr := r.Close(); cerr != nil {
			t.Logf("close pipe reader: %v", cerr)
		}
	})
}

// TestParseFlags covers parseFlags end-to-end, exercising every validation
// branch, the --version sentinel path, and the success path. See issue #68:
// parseFlags previously called os.Exit directly and was untestable.
func TestParseFlags(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		setup       func(t *testing.T)
		wantErr     bool
		sentinel    bool
		errSubstr   string
		checkConfig func(t *testing.T, c *Config)
	}{
		{
			name:      "missing-private-token",
			args:      []string{"prog"},
			setup:     func(t *testing.T) {},
			wantErr:   true,
			errSubstr: "--private-token is required",
		},
		{
			name: "missing-source-branch",
			args: []string{"prog"},
			setup: func(t *testing.T) {
				t.Setenv("GITLAB_PRIVATE_TOKEN", "tok")
			},
			wantErr:   true,
			errSubstr: "--source-branch is required",
		},
		{
			name: "missing-project-id",
			args: []string{"prog"},
			setup: func(t *testing.T) {
				t.Setenv("GITLAB_PRIVATE_TOKEN", "tok")
				t.Setenv("CI_COMMIT_REF_NAME", "feat/x")
			},
			wantErr:   true,
			errSubstr: "--project-id is required",
		},
		{
			name: "missing-gitlab-url",
			args: []string{"prog"},
			setup: func(t *testing.T) {
				t.Setenv("GITLAB_PRIVATE_TOKEN", "tok")
				t.Setenv("CI_COMMIT_REF_NAME", "feat/x")
				t.Setenv("CI_PROJECT_ID", "42")
			},
			wantErr:   true,
			errSubstr: "--gitlab-url is required",
		},
		{
			name: "missing-user-id",
			args: []string{"prog"},
			setup: func(t *testing.T) {
				t.Setenv("GITLAB_PRIVATE_TOKEN", "tok")
				t.Setenv("CI_COMMIT_REF_NAME", "feat/x")
				t.Setenv("CI_PROJECT_ID", "42")
				t.Setenv("CI_PROJECT_URL", "https://gl.example.com/group/proj")
			},
			wantErr:   true,
			errSubstr: "--user-id is required",
		},
		{
			name:     "version",
			args:     []string{"prog", "--version"},
			setup:    setRequiredParseEnv,
			wantErr:  true,
			sentinel: true,
		},
		{
			name: "target-branch-from-env",
			args: []string{"prog"},
			setup: func(t *testing.T) {
				setRequiredParseEnv(t)
				t.Setenv("GITLAB_AUTO_MR_TARGET_BRANCH", "develop")
			},
			checkConfig: func(t *testing.T, c *Config) {
				t.Helper()
				if c.TargetBranch != "develop" {
					t.Errorf("TargetBranch = %q, want %q", c.TargetBranch, "develop")
				}
			},
		},
		{
			name: "target-branch-flag-overrides-env",
			args: []string{"prog", "--target-branch", "release/1.x"},
			setup: func(t *testing.T) {
				setRequiredParseEnv(t)
				t.Setenv("GITLAB_AUTO_MR_TARGET_BRANCH", "develop")
			},
			checkConfig: func(t *testing.T, c *Config) {
				t.Helper()
				if c.TargetBranch != "release/1.x" {
					t.Errorf("TargetBranch = %q, want %q", c.TargetBranch, "release/1.x")
				}
			},
		},
		{
			name: "target-branch-short-flag-overrides-env",
			args: []string{"prog", "-t", "release/1.x"},
			setup: func(t *testing.T) {
				setRequiredParseEnv(t)
				t.Setenv("GITLAB_AUTO_MR_TARGET_BRANCH", "develop")
			},
			checkConfig: func(t *testing.T, c *Config) {
				t.Helper()
				if c.TargetBranch != "release/1.x" {
					t.Errorf("TargetBranch = %q, want %q", c.TargetBranch, "release/1.x")
				}
			},
		},
		{
			name: "target-branch-unset-falls-through-to-project-default",
			args: []string{"prog"},
			setup: func(t *testing.T) {
				setRequiredParseEnv(t)
			},
			checkConfig: func(t *testing.T, c *Config) {
				t.Helper()
				// run() substitutes the project's default branch when this is empty.
				if c.TargetBranch != "" {
					t.Errorf("TargetBranch = %q, want empty", c.TargetBranch)
				}
			},
		},
		{
			name: "labels-and-milestone-from-env",
			args: []string{"prog"},
			setup: func(t *testing.T) {
				setRequiredParseEnv(t)
				t.Setenv("GITLAB_AUTO_MR_LABELS", "bug, needs-review")
				t.Setenv("GITLAB_AUTO_MR_MILESTONE", "5")
			},
			checkConfig: func(t *testing.T, c *Config) {
				t.Helper()
				if len(c.Labels) != 2 || c.Labels[0] != "bug" || c.Labels[1] != "needs-review" {
					t.Errorf("Labels = %v, want [bug needs-review]", c.Labels)
				}
				if c.MilestoneID != 5 {
					t.Errorf("MilestoneID = %d, want 5", c.MilestoneID)
				}
			},
		},
		{
			name: "negative-milestone",
			args: []string{"prog", "--milestone", "-1"},
			setup: func(t *testing.T) {
				setRequiredParseEnv(t)
			},
			wantErr:   true,
			errSubstr: "--milestone must not be negative",
		},
		{
			name: "labels-and-milestone-flags-override-env",
			args: []string{"prog", "--label", "ci", "--milestone", "9"},
			setup: func(t *testing.T) {
				setRequiredParseEnv(t)
				t.Setenv("GITLAB_AUTO_MR_LABELS", "bug")
				t.Setenv("GITLAB_AUTO_MR_MILESTONE", "5")
			},
			checkConfig: func(t *testing.T, c *Config) {
				t.Helper()
				if len(c.Labels) != 1 || c.Labels[0] != "ci" {
					t.Errorf("Labels = %v, want [ci]", c.Labels)
				}
				if c.MilestoneID != 9 {
					t.Errorf("MilestoneID = %d, want 9", c.MilestoneID)
				}
			},
		},
		{
			name:  "success",
			args:  []string{"prog"},
			setup: setRequiredParseEnv,
			checkConfig: func(t *testing.T, c *Config) {
				t.Helper()
				if c == nil {
					t.Fatal("expected non-nil config")
				}
				if c.ProjectID != 42 {
					t.Errorf("ProjectID = %d, want 42", c.ProjectID)
				}
				if c.SourceBranch != "feat/x" {
					t.Errorf("SourceBranch = %q, want %q", c.SourceBranch, "feat/x")
				}
				if len(c.UserIDs) != 1 || c.UserIDs[0] != 7 {
					t.Errorf("UserIDs = %v, want [7]", c.UserIDs)
				}
				if c.GitLabURL == "" {
					t.Error("GitLabURL should be non-empty")
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			clearRequiredParseEnv(t)
			resetFlagSet(t)
			setParseArgs(t, tc.args)
			tc.setup(t)
			if tc.sentinel {
				captureStdout(t)
			}

			cfg, err := parseFlags()

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (config=%+v)", cfg)
				}
				if tc.sentinel {
					if !errors.Is(err, errShowVersion) {
						t.Errorf("expected errShowVersion, got %v", err)
					}
					return
				}
				if errors.Is(err, errShowVersion) {
					t.Errorf("expected non-sentinel error, got errShowVersion")
					return
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
			if tc.checkConfig != nil {
				tc.checkConfig(t, cfg)
			}
		})
	}
}

func TestTriggerMRPipeline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/merge_requests/42/pipelines") {
			t.Errorf("Expected path ending with /merge_requests/42/pipelines, got %s", r.URL.Path)
		}
		if r.Header.Get("PRIVATE-TOKEN") != "test-token" {
			t.Errorf("Expected PRIVATE-TOKEN header, got %s", r.Header.Get("PRIVATE-TOKEN"))
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Pipeline{ID: 7, Status: "created", WebURL: "https://gitlab.example.com/p/7"})
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
	}

	if err := triggerMRPipeline(context.Background(), client, config, &MergeRequest{IID: 42}); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// A missing IID must not fail the run: the MR itself has already been created.
func TestTriggerMRPipelineZeroIID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("Expected no request when MR IID is unknown")
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
	}

	if err := triggerMRPipeline(context.Background(), client, config, &MergeRequest{}); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestTriggerMRPipelineErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{"unauthorized", http.StatusUnauthorized, "", "unauthorized access"},
		{"forbidden", http.StatusForbidden, "", "Developer role"},
		{"no jobs for MR pipelines", http.StatusBadRequest, `{"message":"No stages / jobs"}`, "refused to create"},
		{"server error", http.StatusInternalServerError, "boom", "HTTP 500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := &http.Client{}
			config := &Config{
				GitLabURL:    server.URL,
				ProjectID:    123,
				PrivateToken: "test-token",
			}

			err := triggerMRPipeline(context.Background(), client, config, &MergeRequest{IID: 42})
			if err == nil {
				t.Fatal("Expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

// A response body we cannot parse must not fail the run: GitLab has already
// created the pipeline, and the body is only used to report what was created.
func TestTriggerMRPipelineUnreadableBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := &http.Client{}
	config := &Config{
		GitLabURL:    server.URL,
		ProjectID:    123,
		PrivateToken: "test-token",
	}

	if err := triggerMRPipeline(context.Background(), client, config, &MergeRequest{IID: 42}); err != nil {
		t.Errorf("Expected no error for an unreadable body, got %v", err)
	}
}

// The pipeline must be requested through run(), and before auto-merge: enabling
// merge-when-pipeline-succeeds is what the pipeline is needed for.
func TestRunWithTriggerPipeline(t *testing.T) {
	var calls []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" &&
			!strings.Contains(r.URL.Path, "merge_requests"):
			json.NewEncoder(w).Encode(Project{ID: 123, Name: "test-project", DefaultBranch: "main"})
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			json.NewEncoder(w).Encode([]MergeRequest{})
		case strings.HasSuffix(r.URL.Path, "/merge_requests") && r.Method == "POST":
			calls = append(calls, "create")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(MergeRequest{ID: 1, IID: 10, Title: "Test MR"})
		case strings.HasSuffix(r.URL.Path, "/merge_requests/10/pipelines") && r.Method == "POST":
			calls = append(calls, "pipeline")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Pipeline{ID: 7, Status: "created"})
		case strings.HasSuffix(r.URL.Path, "/merge_requests/10/merge") && r.Method == "PUT":
			calls = append(calls, "merge")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(MergeRequest{ID: 1, IID: 10})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &Config{
		GitLabURL:       server.URL,
		ProjectID:       123,
		PrivateToken:    "test-token",
		SourceBranch:    "feature/test",
		TargetBranch:    "main",
		UserIDs:         []int{1},
		TriggerPipeline: true,
		AutoMerge:       true,
	}

	if err := run(context.Background(), config); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	want := []string{"create", "pipeline", "merge"}
	if len(calls) != len(want) {
		t.Fatalf("Expected calls %v, got %v", want, calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("Expected calls %v, got %v", want, calls)
		}
	}
}

// Without the flag nothing extra is requested — the default behavior is unchanged.
func TestRunWithoutTriggerPipeline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/pipelines") && r.Method == "POST":
			t.Error("Expected no pipeline request without --trigger-pipeline")
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123") && r.Method == "GET" &&
			!strings.Contains(r.URL.Path, "merge_requests"):
			json.NewEncoder(w).Encode(Project{ID: 123, Name: "test-project", DefaultBranch: "main"})
		case strings.HasPrefix(r.URL.Path, "/api/v4/projects/123/merge_requests") && r.Method == "GET":
			json.NewEncoder(w).Encode([]MergeRequest{})
		case strings.HasSuffix(r.URL.Path, "/merge_requests") && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(MergeRequest{ID: 1, IID: 10, Title: "Test MR"})
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
	}

	if err := run(context.Background(), config); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
