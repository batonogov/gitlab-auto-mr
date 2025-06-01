package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client represents a GitLab API client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new GitLab client
func NewClient(baseURL, token string, insecure bool) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// Ensure baseURL ends with /
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	if insecure {
		// For insecure connections, we would modify the transport
		// but since we're minimizing dependencies, we'll keep it simple
	}

	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: httpClient,
	}, nil
}

// MergeRequestOptions represents options for creating a merge request
type MergeRequestOptions struct {
	SourceBranch       string `json:"source_branch"`
	TargetBranch       string `json:"target_branch"`
	Title              string `json:"title"`
	Description        string `json:"description,omitempty"`
	AssigneeIDs        []int  `json:"assignee_ids,omitempty"`
	ReviewerIDs        []int  `json:"reviewer_ids,omitempty"`
	RemoveSourceBranch bool   `json:"remove_source_branch,omitempty"`
	Squash             bool   `json:"squash,omitempty"`
	AllowCollaboration bool   `json:"allow_collaboration,omitempty"`
}

// MergeRequest represents a GitLab merge request
type MergeRequest struct {
	ID                 int    `json:"id"`
	IID                int    `json:"iid"`
	ProjectID          int    `json:"project_id"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	State              string `json:"state"`
	SourceBranch       string `json:"source_branch"`
	TargetBranch       string `json:"target_branch"`
	WebURL             string `json:"web_url"`
	AssigneeIDs        []int  `json:"assignee_ids"`
	ReviewerIDs        []int  `json:"reviewer_ids"`
	RemoveSourceBranch bool   `json:"remove_source_branch"`
	Squash             bool   `json:"squash"`
	AllowCollaboration bool   `json:"allow_collaboration"`
}

// GitLabError represents an error response from GitLab API
type GitLabError struct {
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors"`
}

func (e *GitLabError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if len(e.Errors) > 0 {
		var msgs []string
		for field, msg := range e.Errors {
			msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
		}
		return strings.Join(msgs, ", ")
	}
	return "unknown GitLab API error"
}

// CreateMergeRequest creates a new merge request
func (c *Client) CreateMergeRequest(projectID int, opts *MergeRequestOptions) (*MergeRequest, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	apiURL := fmt.Sprintf("%sapi/v4/projects/%d/merge_requests", c.baseURL, projectID)

	jsonData, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var gitlabErr GitLabError
		if err := json.Unmarshal(body, &gitlabErr); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, &gitlabErr
	}

	var mr MergeRequest
	if err := json.Unmarshal(body, &mr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &mr, nil
}

// GetMergeRequests gets merge requests for a project
func (c *Client) GetMergeRequests(projectID int, sourceBranch, targetBranch string) ([]MergeRequest, error) {
	apiURL := fmt.Sprintf("%sapi/v4/projects/%d/merge_requests", c.baseURL, projectID)

	// Build query parameters
	params := url.Values{}
	if sourceBranch != "" {
		params.Set("source_branch", sourceBranch)
	}
	if targetBranch != "" {
		params.Set("target_branch", targetBranch)
	}
	params.Set("state", "opened")

	if len(params) > 0 {
		apiURL += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var gitlabErr GitLabError
		if err := json.Unmarshal(body, &gitlabErr); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, &gitlabErr
	}

	var mrs []MergeRequest
	if err := json.Unmarshal(body, &mrs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return mrs, nil
}

// CheckMergeRequestExists checks if a merge request already exists
func (c *Client) CheckMergeRequestExists(projectID int, sourceBranch, targetBranch string) (bool, error) {
	mrs, err := c.GetMergeRequests(projectID, sourceBranch, targetBranch)
	if err != nil {
		return false, err
	}
	return len(mrs) > 0, nil
}

// ExtractProjectIDFromURL extracts project ID from GitLab project URL
func ExtractProjectIDFromURL(projectURL string) (int, error) {
	// This is a simplified version - in reality, you might need to make an API call
	// to get the project ID from the project path
	parts := strings.Split(strings.TrimSuffix(projectURL, "/"), "/")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid project URL format")
	}

	// Try to extract from URL path like https://gitlab.com/group/project
	// This is simplified - real implementation would use API to resolve project path to ID
	return 0, fmt.Errorf("project ID extraction from URL not implemented - please provide project ID directly")
}

// ParseUserIDs parses comma-separated user IDs
func ParseUserIDs(userIDsStr string) ([]int, error) {
	if userIDsStr == "" {
		return nil, nil
	}

	parts := strings.Split(userIDsStr, ",")
	var userIDs []int

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID '%s': %w", part, err)
		}
		userIDs = append(userIDs, id)
	}

	return userIDs, nil
}
