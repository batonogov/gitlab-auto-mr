package main

import (
	"os"
	"testing"
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
