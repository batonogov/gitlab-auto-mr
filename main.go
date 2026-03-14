package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Version information set via ldflags at build time.
var (
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)

type Config struct {
	PrivateToken       string
	SourceBranch       string
	ProjectID          int
	GitLabURL          string
	UserIDs            []int
	ReviewerIDs        []int
	Insecure           bool
	TargetBranch       string
	CommitPrefix       string
	RemoveBranch       bool
	SquashCommits      bool
	Description        string
	Title              string
	UseIssueName       bool
	AllowCollaboration bool
	MRExists           bool
	UpdateMR           bool
	CreateOnly         bool
	AutoMerge          bool
}

type Project struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
}

type MergeRequest struct {
	ID           int    `json:"id"`
	IID          int    `json:"iid"`
	Title        string `json:"title"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	State        string `json:"state"`
}

type Issue struct {
	ID        int      `json:"id"`
	IID       int      `json:"iid"`
	Title     string   `json:"title"`
	Labels    []string `json:"labels"`
	Milestone struct {
		ID int `json:"id"`
	} `json:"milestone"`
}

type MRCreateRequest struct {
	SourceBranch       string   `json:"source_branch"`
	TargetBranch       string   `json:"target_branch"`
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	AssigneeIDs        []int    `json:"assignee_ids,omitempty"`
	ReviewerIDs        []int    `json:"reviewer_ids,omitempty"`
	RemoveSourceBranch bool     `json:"remove_source_branch"`
	Squash             bool     `json:"squash"`
	AllowCollaboration bool     `json:"allow_collaboration"`
	MilestoneID        int      `json:"milestone_id,omitempty"`
	Labels             []string `json:"labels,omitempty"`
}

type MRUpdateRequest struct {
	Title              string   `json:"title,omitempty"`
	Description        string   `json:"description,omitempty"`
	AssigneeIDs        []int    `json:"assignee_ids,omitempty"`
	ReviewerIDs        []int    `json:"reviewer_ids,omitempty"`
	RemoveSourceBranch bool     `json:"remove_source_branch,omitempty"`
	Squash             bool     `json:"squash,omitempty"`
	AllowCollaboration bool     `json:"allow_collaboration,omitempty"`
	MilestoneID        int      `json:"milestone_id,omitempty"`
	Labels             []string `json:"labels,omitempty"`
}

type MRAcceptRequest struct {
	MergeWhenPipelineSucceeds bool `json:"merge_when_pipeline_succeeds"`
	ShouldRemoveSourceBranch  bool `json:"should_remove_source_branch"`
	Squash                    bool `json:"squash"`
}

func main() {
	config := parseFlags()

	if err := run(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() *Config {
	config := &Config{}

	var userIDsStr, reviewerIDsStr string
	var showVersion bool

	flag.StringVar(&config.PrivateToken, "private-token", getEnv("GITLAB_PRIVATE_TOKEN", ""), "Private GITLAB token")
	flag.StringVar(&config.SourceBranch, "source-branch", getEnv("CI_COMMIT_REF_NAME", ""), "Source branch to merge from")
	flag.IntVar(&config.ProjectID, "project-id", getEnvInt("CI_PROJECT_ID", 0), "GitLab project ID")
	flag.StringVar(&config.GitLabURL, "gitlab-url", getEnv("CI_PROJECT_URL", ""), "GitLab URL")
	flag.StringVar(&userIDsStr, "user-id", getEnv("GITLAB_USER_ID", ""), "User IDs to assign MR to (comma-separated)")
	flag.StringVar(&reviewerIDsStr, "reviewer-id", "", "Reviewer IDs (comma-separated)")
	flag.BoolVar(&config.Insecure, "insecure", false, "Skip SSL verification")
	flag.BoolVar(&config.Insecure, "k", false, "Skip SSL verification (short)")
	flag.StringVar(&config.TargetBranch, "target-branch", "", "Target branch to merge onto")
	flag.StringVar(&config.TargetBranch, "t", "", "Target branch to merge onto (short)")
	flag.StringVar(&config.CommitPrefix, "commit-prefix", "Draft", "Prefix for MR title")
	flag.StringVar(&config.CommitPrefix, "c", "Draft", "Prefix for MR title (short)")
	flag.BoolVar(&config.RemoveBranch, "remove-branch", false, "Remove source branch after merge")
	flag.BoolVar(&config.RemoveBranch, "r", false, "Remove source branch after merge (short)")
	flag.BoolVar(&config.SquashCommits, "squash-commits", false, "Squash commits on merge")
	flag.BoolVar(&config.SquashCommits, "s", false, "Squash commits on merge (short)")
	flag.StringVar(&config.Description, "description", "", "Path to description file")
	flag.StringVar(&config.Description, "d", "", "Path to description file (short)")
	flag.StringVar(&config.Title, "title", "", "Custom MR title")
	flag.BoolVar(&config.UseIssueName, "use-issue-name", false, "Use issue data from branch name")
	flag.BoolVar(&config.UseIssueName, "i", false, "Use issue data from branch name (short)")
	flag.BoolVar(&config.AllowCollaboration, "allow-collaboration", false, "Allow collaboration")
	flag.BoolVar(&config.AllowCollaboration, "a", false, "Allow collaboration (short)")
	flag.BoolVar(&config.MRExists, "mr-exists", false, "Check if MR exists (dry run)")
	flag.BoolVar(&config.UpdateMR, "update-mr", false, "Update existing MR instead of creating new one")
	flag.BoolVar(&config.CreateOnly, "create-only", false, "Only create new MR, fail if MR already exists")
	flag.BoolVar(&config.AutoMerge, "auto-merge", false, "Enable merge when pipeline succeeds (auto-merge)")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&showVersion, "v", false, "Show version information and exit (short)")

	flag.Parse()

	if showVersion {
		fmt.Println(versionInfo())
		os.Exit(0)
	}

	// Validate required fields
	if config.PrivateToken == "" {
		fmt.Fprintf(os.Stderr, "Error: --private-token is required\n")
		os.Exit(1)
	}
	if config.SourceBranch == "" {
		fmt.Fprintf(os.Stderr, "Error: --source-branch is required\n")
		os.Exit(1)
	}
	if config.ProjectID == 0 {
		fmt.Fprintf(os.Stderr, "Error: --project-id is required\n")
		os.Exit(1)
	}
	if config.GitLabURL == "" {
		fmt.Fprintf(os.Stderr, "Error: --gitlab-url is required\n")
		os.Exit(1)
	}
	if userIDsStr == "" {
		fmt.Fprintf(os.Stderr, "Error: --user-id is required\n")
		os.Exit(1)
	}

	// Parse user IDs
	config.UserIDs = parseIntSlice(userIDsStr)
	if reviewerIDsStr != "" {
		config.ReviewerIDs = parseIntSlice(reviewerIDsStr)
	}

	// Clean GitLab URL if it contains full project URL
	if strings.Contains(config.GitLabURL, "/") {
		re := regexp.MustCompile(`^https?://[^/]+`)
		matches := re.FindString(config.GitLabURL)
		if matches != "" {
			config.GitLabURL = matches
		}
	}

	return config
}

func isDraftPrefix(prefix string) bool {
	lower := strings.ToLower(strings.TrimSpace(prefix))
	return lower == "draft" || lower == "wip"
}

func validateConfig(config *Config) error {
	if config.AutoMerge && config.MRExists {
		return fmt.Errorf("--auto-merge cannot be used with --mr-exists (dry run mode)")
	}

	if config.AutoMerge && isDraftPrefix(config.CommitPrefix) {
		return fmt.Errorf(
			"--auto-merge cannot be used with --commit-prefix %q: "+
				"GitLab does not allow auto-merge for draft merge requests",
			config.CommitPrefix,
		)
	}

	return nil
}

func checkMRExists(config *Config, existingMR *MergeRequest) {
	if existingMR == nil {
		fmt.Printf(
			"Merge request does not exist for this branch %s to %s, "+
				"run without flag '--mr-exists' to open merge request.\n",
			config.SourceBranch, config.TargetBranch)
	} else {
		fmt.Printf("Merge request exists: %s (IID: %d)\n",
			existingMR.Title, existingMR.IID)
	}
}

func run(config *Config) error {
	if err := validateConfig(config); err != nil {
		return err
	}

	client := createHTTPClient(config.Insecure)

	project, err := getProject(client, config)
	if err != nil {
		return fmt.Errorf("unable to get project %d: %v", config.ProjectID, err)
	}

	if config.TargetBranch == "" {
		config.TargetBranch = project.DefaultBranch
	}

	if err := validateMR(config.SourceBranch, config.TargetBranch); err != nil {
		return err
	}

	existingMR, err := getExistingMR(client, config)
	if err != nil {
		return fmt.Errorf("failed to check if MR exists: %v", err)
	}

	if config.MRExists {
		checkMRExists(config, existingMR)
		return nil
	}

	if config.CreateOnly && existingMR != nil {
		return fmt.Errorf(
			"merge request already exists for this branch %s to %s, "+
				"cannot create new MR in create-only mode",
			config.SourceBranch, config.TargetBranch,
		)
	}

	if config.UpdateMR && existingMR == nil {
		return fmt.Errorf(
			"merge request does not exist for this branch %s to %s, "+
				"cannot update non-existent MR",
			config.SourceBranch, config.TargetBranch,
		)
	}

	title := getMRTitle(config.CommitPrefix, config.Title, config.SourceBranch)
	description := getDescriptionData(config.Description)

	mrIID, err := handleMR(client, config, existingMR, title, description)
	if err != nil {
		return err
	}

	if config.AutoMerge {
		return enableAutoMerge(client, config, mrIID)
	}

	return nil
}

func handleMR(
	client *http.Client, config *Config,
	existingMR *MergeRequest, title, description string,
) (int, error) {
	switch {
	case existingMR != nil && !config.UpdateMR:
		if config.AutoMerge {
			fmt.Printf(
				"Merge request already exists: %s (IID: %d), enabling auto-merge.\n",
				existingMR.Title, existingMR.IID,
			)
		} else {
			fmt.Printf(
				"Merge request already exists: %s (IID: %d). "+
					"Use --update-mr flag to update it.\n",
				existingMR.Title, existingMR.IID,
			)
		}
		return existingMR.IID, nil

	case existingMR != nil:
		return handleUpdateMR(client, config, existingMR, title, description)

	default:
		return handleCreateMR(client, config, title, description)
	}
}

func handleUpdateMR(
	client *http.Client, config *Config,
	existingMR *MergeRequest, title, description string,
) (int, error) {
	updateRequest := &MRUpdateRequest{
		Title:              title,
		Description:        description,
		AssigneeIDs:        config.UserIDs,
		ReviewerIDs:        config.ReviewerIDs,
		RemoveSourceBranch: config.RemoveBranch,
		Squash:             config.SquashCommits,
		AllowCollaboration: config.AllowCollaboration,
	}

	if config.UseIssueName {
		issueData, err := getIssueData(client, config)
		if err == nil {
			updateRequest.MilestoneID = issueData.Milestone.ID
			updateRequest.Labels = issueData.Labels
		}
	}

	if err := updateMR(client, config, existingMR.IID, updateRequest); err != nil {
		return 0, fmt.Errorf("failed to update MR: %v", err)
	}

	fmt.Printf("Updated existing MR %s (IID: %d)\n", title, existingMR.IID)
	return existingMR.IID, nil
}

func handleCreateMR(
	client *http.Client, config *Config,
	title, description string,
) (int, error) {
	mrRequest := &MRCreateRequest{
		SourceBranch:       config.SourceBranch,
		TargetBranch:       config.TargetBranch,
		Title:              title,
		Description:        description,
		AssigneeIDs:        config.UserIDs,
		ReviewerIDs:        config.ReviewerIDs,
		RemoveSourceBranch: config.RemoveBranch,
		Squash:             config.SquashCommits,
		AllowCollaboration: config.AllowCollaboration,
	}

	if config.UseIssueName {
		issueData, err := getIssueData(client, config)
		if err == nil {
			mrRequest.MilestoneID = issueData.Milestone.ID
			mrRequest.Labels = issueData.Labels
		}
	}

	createdMR, err := createMR(client, config, mrRequest)
	if err != nil {
		return 0, fmt.Errorf("failed to create MR: %v", err)
	}

	fmt.Printf("Created a new MR %s, assigned to you.\n", title)
	return createdMR.IID, nil
}

func enableAutoMerge(client *http.Client, config *Config, mrIID int) error {
	if mrIID == 0 {
		fmt.Println("Warning: could not determine MR IID, skipping auto-merge")
		return nil
	}

	if err := acceptMR(client, config, mrIID); err != nil {
		return fmt.Errorf("failed to enable auto-merge: %v", err)
	}

	fmt.Printf("Auto-merge enabled for MR (IID: %d)\n", mrIID)
	return nil
}

func createHTTPClient(insecure bool) *http.Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // user-requested via --insecure flag
		}
		client.Transport = tr
	}

	return client
}

func getProject(client *http.Client, config *Config) (*Project, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%d", config.GitLabURL, config.ProjectID)

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", config.PrivateToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("unauthorized access, check your access token is valid")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, err
	}

	return &project, nil
}

func validateMR(sourceBranch, targetBranch string) error {
	if sourceBranch == targetBranch {
		return fmt.Errorf("source branch and target branches must be different, source: %s and target: %s",
			sourceBranch, targetBranch)
	}
	return nil
}

func getExistingMR(client *http.Client, config *Config) (*MergeRequest, error) {
	apiURL := fmt.Sprintf("%s/api/v4/projects/%d/merge_requests?state=opened&source_branch=%s&target_branch=%s",
		config.GitLabURL, config.ProjectID, config.SourceBranch, config.TargetBranch)

	req, err := http.NewRequest("GET", apiURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", config.PrivateToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var mrs []MergeRequest
	if err := json.NewDecoder(resp.Body).Decode(&mrs); err != nil {
		return nil, err
	}

	if len(mrs) > 0 {
		return &mrs[0], nil
	}

	return nil, nil
}

func getMRTitle(prefix, title, sourceBranch string) string {
	if title != "" {
		if prefix != "" {
			return fmt.Sprintf("%s: %s", prefix, title)
		}
		return title
	}

	if prefix != "" {
		return fmt.Sprintf("%s: %s", prefix, sourceBranch)
	}
	return sourceBranch
}

func getDescriptionData(descriptionPath string) string {
	if descriptionPath == "" {
		return ""
	}

	data, err := os.ReadFile(descriptionPath)
	if err != nil {
		fmt.Printf("Unable to read description file at %s: %v. No description will be set.\n",
			descriptionPath, err)
		return ""
	}

	return string(data)
}

func getIssueData(client *http.Client, config *Config) (*Issue, error) {
	re := regexp.MustCompile(`#(\d+)`)
	matches := re.FindStringSubmatch(config.SourceBranch)
	if len(matches) < 2 {
		return nil, fmt.Errorf("issue number not found in %s", config.SourceBranch)
	}

	issueID, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid issue number: %s", matches[1])
	}

	url := fmt.Sprintf("%s/api/v4/projects/%d/issues/%d", config.GitLabURL, config.ProjectID, issueID)

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", config.PrivateToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("issue #%d not found", issueID)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func createMR(client *http.Client, config *Config, mrRequest *MRCreateRequest) (*MergeRequest, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%d/merge_requests", config.GitLabURL, config.ProjectID)

	jsonData, err := json.Marshal(mrRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", config.PrivateToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var mr MergeRequest
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		if err == io.EOF {
			return &mr, nil
		}
		return nil, fmt.Errorf("MR created but response is invalid: %v", err)
	}

	return &mr, nil
}

func updateMR(client *http.Client, config *Config, mrIID int, updateRequest *MRUpdateRequest) error {
	url := fmt.Sprintf("%s/api/v4/projects/%d/merge_requests/%d", config.GitLabURL, config.ProjectID, mrIID)

	jsonData, err := json.Marshal(updateRequest)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("PRIVATE-TOKEN", config.PrivateToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func acceptMR(client *http.Client, config *Config, mrIID int) error {
	url := fmt.Sprintf("%s/api/v4/projects/%d/merge_requests/%d/merge", config.GitLabURL, config.ProjectID, mrIID)

	acceptRequest := &MRAcceptRequest{
		MergeWhenPipelineSucceeds: true,
		ShouldRemoveSourceBranch:  config.RemoveBranch,
		Squash:                    config.SquashCommits,
	}

	jsonData, err := json.Marshal(acceptRequest)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("PRIVATE-TOKEN", config.PrivateToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		return nil
	case 401:
		return fmt.Errorf("unauthorized access, check your access token permissions")
	case 405:
		return fmt.Errorf(
			"merge request cannot be merged, " +
				"the pipeline may not have started yet or other merge conditions are not met",
		)
	case 406:
		return fmt.Errorf(
			"merge request cannot be merged, " +
				"there may be unresolved discussions or other blocking conditions",
		)
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
}

func versionInfo() string {
	return fmt.Sprintf("gitlab-auto-mr %s (commit: %s, built: %s)", Version, GitCommit, BuildDate)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func parseIntSlice(s string) []int {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if num, err := strconv.Atoi(part); err == nil {
			result = append(result, num)
		}
	}

	return result
}
