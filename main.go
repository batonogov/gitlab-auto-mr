package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// draftPrefix is the lowercase title prefix marking an MR as a draft.
// Shared by isDraftPrefix so the literals are not duplicated (goconst).
const (
	draftPrefix = "draft"
	wipPrefix   = "wip"
)

// errShowVersion is a sentinel error returned by parseFlags when --version has
// been handled (version printed to stdout). Callers should treat it as a clean
// exit with status 0 rather than a real error.
var errShowVersion = errors.New("version shown")

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
	TriggerPipeline    bool
	Labels             []string
	MilestoneID        int
	ForcePipeline      bool
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
	WebURL       string `json:"web_url"`
	SHA          string `json:"sha"`
}

type Pipeline struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	WebURL string `json:"web_url"`
	SHA    string `json:"sha"`
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
	RemoveSourceBranch *bool    `json:"remove_source_branch,omitempty"`
	Squash             *bool    `json:"squash,omitempty"`
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
	config, err := parseFlags()
	if err != nil {
		if errors.Is(err, errShowVersion) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := run(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() (*Config, error) {
	config := &Config{}

	var userIDsStr, reviewerIDsStr, labelsStr string
	var showVersion bool

	flag.StringVar(&config.PrivateToken, "private-token", getEnv("GITLAB_PRIVATE_TOKEN", ""), "Private GITLAB token")
	flag.StringVar(&config.SourceBranch, "source-branch", getEnv("CI_COMMIT_REF_NAME", ""), "Source branch to merge from")
	flag.IntVar(&config.ProjectID, "project-id", getEnvInt("CI_PROJECT_ID", 0), "GitLab project ID")
	flag.StringVar(&config.GitLabURL, "gitlab-url", getEnv("CI_PROJECT_URL", ""), "GitLab URL")
	flag.StringVar(&userIDsStr, "user-id", getEnv("GITLAB_USER_ID", ""), "User IDs to assign MR to (comma-separated)")
	flag.StringVar(&reviewerIDsStr, "reviewer-id", "", "Reviewer IDs (comma-separated)")
	flag.BoolVar(&config.Insecure, "insecure", false, "Skip SSL verification")
	flag.BoolVar(&config.Insecure, "k", false, "Skip SSL verification (short)")
	// Both spellings share one variable, so they must share one default: whichever
	// flag.StringVar runs last decides the initial value.
	targetBranchDefault := getEnv("GITLAB_AUTO_MR_TARGET_BRANCH", "")
	flag.StringVar(&config.TargetBranch, "target-branch", targetBranchDefault, "Target branch to merge onto")
	flag.StringVar(&config.TargetBranch, "t", targetBranchDefault, "Target branch to merge onto (short)")
	flag.StringVar(&config.CommitPrefix, "commit-prefix", "Draft", "Prefix for MR title")
	flag.StringVar(&config.CommitPrefix, "c", "Draft", "Prefix for MR title (short)")
	flag.BoolVar(&config.RemoveBranch, "remove-branch", false, "Remove source branch after merge")
	flag.BoolVar(&config.RemoveBranch, "r", false, "Remove source branch after merge (short)")
	flag.BoolVar(&config.SquashCommits, "squash-commits", false, "Squash commits on merge")
	flag.BoolVar(&config.SquashCommits, "s", false, "Squash commits on merge (short)")
	flag.StringVar(&config.Description, "description", "", "Path to description file")
	flag.StringVar(&config.Description, "d", "", "Path to description file (short)")
	flag.StringVar(&config.Title, "title", "", "Custom MR title")
	flag.StringVar(&labelsStr, "label", getEnv("GITLAB_AUTO_MR_LABELS", ""),
		"Labels to set on the MR (comma-separated)")
	flag.IntVar(&config.MilestoneID, "milestone", getEnvInt("GITLAB_AUTO_MR_MILESTONE", 0),
		"Milestone ID to set on the MR")
	flag.BoolVar(&config.UseIssueName, "use-issue-name", false, "Use issue data from branch name")
	flag.BoolVar(&config.UseIssueName, "i", false, "Use issue data from branch name (short)")
	flag.BoolVar(&config.AllowCollaboration, "allow-collaboration", false, "Allow collaboration")
	flag.BoolVar(&config.AllowCollaboration, "a", false, "Allow collaboration (short)")
	flag.BoolVar(&config.MRExists, "mr-exists", false, "Check if MR exists (dry run)")
	flag.BoolVar(&config.UpdateMR, "update-mr", false, "Update existing MR instead of creating new one")
	flag.BoolVar(&config.CreateOnly, "create-only", false, "Only create new MR, fail if MR already exists")
	flag.BoolVar(&config.AutoMerge, "auto-merge", false, "Enable merge when pipeline succeeds (auto-merge)")
	flag.BoolVar(&config.ForcePipeline, "force-pipeline", false,
		"With --trigger-pipeline, create a pipeline even if one exists for the same commit")
	flag.BoolVar(&config.TriggerPipeline, "trigger-pipeline", false,
		"Create a merge request pipeline for the MR, whether it was created or updated")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&showVersion, "v", false, "Show version information and exit (short)")

	flag.Parse()

	if showVersion {
		fmt.Println(versionInfo())
		return nil, errShowVersion
	}

	// Validate required fields
	if config.PrivateToken == "" {
		return nil, fmt.Errorf("--private-token is required")
	}
	if config.SourceBranch == "" {
		return nil, fmt.Errorf("--source-branch is required")
	}
	if config.ProjectID == 0 {
		return nil, fmt.Errorf("--project-id is required")
	}
	if config.GitLabURL == "" {
		return nil, fmt.Errorf("--gitlab-url is required")
	}
	if userIDsStr == "" {
		return nil, fmt.Errorf("--user-id is required")
	}

	// Parse user IDs
	config.UserIDs = parseIntSlice(userIDsStr)
	if reviewerIDsStr != "" {
		config.ReviewerIDs = parseIntSlice(reviewerIDsStr)
	}
	config.Labels = parseStringSlice(labelsStr)

	if config.MilestoneID < 0 {
		return nil, fmt.Errorf("--milestone must not be negative, got %d", config.MilestoneID)
	}

	// Clean GitLab URL if it contains full project URL
	if strings.Contains(config.GitLabURL, "/") {
		re := regexp.MustCompile(`^https?://[^/]+`)
		matches := re.FindString(config.GitLabURL)
		if matches != "" {
			config.GitLabURL = matches
		}
	}

	return config, nil
}

func isDraftPrefix(prefix string) bool {
	lower := strings.ToLower(strings.TrimSpace(prefix))
	return lower == draftPrefix || lower == wipPrefix
}

func validateConfig(config *Config) error {
	if config.ForcePipeline && !config.TriggerPipeline {
		return fmt.Errorf("--force-pipeline has no effect without --trigger-pipeline")
	}

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
		printMRURL(existingMR)
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

	if err := checkMRMode(config, existingMR); err != nil {
		return err
	}

	title := getMRTitle(config.CommitPrefix, config.Title, config.SourceBranch)
	description := getDescriptionData(config.Description)

	mr, err := handleMR(client, config, existingMR, title, description)
	if err != nil {
		return err
	}
	if mr == nil {
		// Defensive: every handleMR branch returns an MR on success.
		mr = &MergeRequest{}
	}

	if config.TriggerPipeline {
		if err := triggerMRPipeline(client, config, mr); err != nil {
			return fmt.Errorf("failed to trigger merge request pipeline: %v", err)
		}
	}

	if config.AutoMerge {
		return enableAutoMerge(client, config, mr.IID)
	}

	return nil
}

// checkMRMode rejects the two combinations where the mode the user asked for
// contradicts what is actually on the server.
func checkMRMode(config *Config, existingMR *MergeRequest) error {
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

	return nil
}

func handleMR(
	client *http.Client, config *Config,
	existingMR *MergeRequest, title, description string,
) (*MergeRequest, error) {
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
		printMRURL(existingMR)
		return existingMR, nil

	case existingMR != nil:
		return handleUpdateMR(client, config, existingMR, title, description)

	default:
		return handleCreateMR(client, config, title, description)
	}
}

func handleUpdateMR(
	client *http.Client, config *Config,
	existingMR *MergeRequest, title, description string,
) (*MergeRequest, error) {
	updateRequest := &MRUpdateRequest{
		Title:              title,
		Description:        description,
		AssigneeIDs:        config.UserIDs,
		ReviewerIDs:        config.ReviewerIDs,
		RemoveSourceBranch: boolPtr(config.RemoveBranch),
		Squash:             boolPtr(config.SquashCommits),
		AllowCollaboration: config.AllowCollaboration,
	}

	updateRequest.MilestoneID, updateRequest.Labels = resolveMRMetadata(client, config)

	if err := updateMR(client, config, existingMR.IID, updateRequest); err != nil {
		return nil, fmt.Errorf("failed to update MR: %v", err)
	}

	fmt.Printf("Updated existing MR %s (IID: %d)\n", title, existingMR.IID)
	printMRURL(existingMR)
	return existingMR, nil
}

func handleCreateMR(
	client *http.Client, config *Config,
	title, description string,
) (*MergeRequest, error) {
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

	mrRequest.MilestoneID, mrRequest.Labels = resolveMRMetadata(client, config)

	createdMR, err := createMR(client, config, mrRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create MR: %v", err)
	}

	fmt.Printf("Created a new MR %s, assigned to you.\n", title)
	printMRURL(createdMR)
	return createdMR, nil
}

// resolveMRMetadata determines the milestone and labels for the MR, combining
// what was given on the command line with what the linked issue carries.
//
// --milestone wins over the issue's milestone; labels are the union of the two,
// with --label values first. A failure to fetch the issue is a warning, not an
// error: the MR is still worth creating without its issue metadata.
func resolveMRMetadata(client *http.Client, config *Config) (int, []string) {
	milestoneID := config.MilestoneID
	labels := config.Labels

	if !config.UseIssueName {
		return milestoneID, labels
	}

	issue, err := getIssueData(client, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to fetch issue data: %v\n", err)
		return milestoneID, labels
	}

	if milestoneID == 0 {
		milestoneID = issue.Milestone.ID
	}

	return milestoneID, mergeLabels(labels, issue.Labels)
}

// mergeLabels appends the labels from extra that are not already in base,
// preserving the order of both and never returning a non-nil empty slice
// (MRCreateRequest.Labels is omitempty, and an empty list would clear labels).
func mergeLabels(base, extra []string) []string {
	if len(extra) == 0 {
		if len(base) == 0 {
			return nil
		}
		return base
	}

	seen := make(map[string]struct{}, len(base)+len(extra))
	merged := make([]string, 0, len(base)+len(extra))

	for _, group := range [][]string{base, extra} {
		for _, label := range group {
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			merged = append(merged, label)
		}
	}

	if len(merged) == 0 {
		return nil
	}
	return merged
}

// printMRURL writes the MR's browser URL so it can be clicked straight out of a
// CI job log. The URL comes from the API response rather than being assembled
// locally: --gitlab-url is only the instance host, so the project path with its
// namespace is not knowable here. Nothing is printed if GitLab omitted web_url.
func printMRURL(mr *MergeRequest) {
	if mr == nil || mr.WebURL == "" {
		return
	}
	fmt.Printf("MR URL: %s\n", mr.WebURL)
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

// triggerMRPipeline asks GitLab to create a merge request pipeline for an
// existing MR.
//
// GitLab does not start one on its own when an MR is created through the API
// for a commit that already has a branch pipeline. A CI configuration whose
// jobs run only on `merge_request_event` would therefore never see the MR, and
// the missing checks are easy to mistake for passing ones.
//
// The call runs for updated MRs as well as new ones, because a moved branch is
// exactly when the checks are worth re-running. To keep that from producing a
// pipeline per job run, an existing pipeline for the same commit is left alone
// unless --force-pipeline says otherwise.
func triggerMRPipeline(client *http.Client, config *Config, mr *MergeRequest) error {
	if mr == nil || mr.IID == 0 {
		fmt.Println("Warning: could not determine MR IID, skipping pipeline trigger")
		return nil
	}

	if !config.ForcePipeline {
		existing, err := findPipelineForSHA(client, config, mr)
		if err != nil {
			// Not being able to list pipelines is no reason to skip creating one;
			// the worst case is the duplicate this check exists to avoid.
			fmt.Fprintf(os.Stderr, "Warning: could not check for existing pipelines: %v\n", err)
		} else if existing != nil {
			fmt.Printf(
				"Merge request pipeline already exists for commit %s (ID: %d, status: %s)%s\n",
				shortSHA(mr.SHA), existing.ID, existing.Status, urlSuffix(existing.WebURL),
			)
			fmt.Println("Skipping pipeline creation; pass --force-pipeline to create another.")
			return nil
		}
	}

	apiURL := fmt.Sprintf(
		"%s/api/v4/projects/%d/merge_requests/%d/pipelines",
		config.GitLabURL, config.ProjectID, mr.IID,
	)

	req, err := http.NewRequest("POST", apiURL, http.NoBody)
	if err != nil {
		return err
	}

	req.Header.Set("PRIVATE-TOKEN", config.PrivateToken)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200, 201:
		// The pipeline already exists at this point, so a response we cannot read
		// is worth a warning but must not fail the run: the body is only used to
		// tell the user what was created.
		var pipeline Pipeline
		if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
			fmt.Printf("Warning: merge request pipeline created but response could not be read: %v\n", err)
			return nil
		}
		fmt.Printf(
			"Merge request pipeline created (ID: %d, status: %s)%s\n",
			pipeline.ID, pipeline.Status, urlSuffix(pipeline.WebURL),
		)
		return nil
	case 401:
		return fmt.Errorf("unauthorized access, check your access token permissions")
	case 403:
		return fmt.Errorf(
			"forbidden, the token owner needs at least the Developer role on the project " +
				"to create pipelines",
		)
	case 400:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"GitLab refused to create the pipeline, "+
				"the CI configuration may define no jobs for merge request pipelines: %s",
			string(respBody),
		)
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
}

// findPipelineForSHA returns the MR's existing pipeline for its head commit, or
// nil when there is none.
//
// With an unknown head SHA there is nothing to compare against, so it reports
// no match and the caller creates a pipeline: a duplicate is a better outcome
// than silently skipping the checks.
func findPipelineForSHA(client *http.Client, config *Config, mr *MergeRequest) (*Pipeline, error) {
	if mr.SHA == "" {
		return nil, nil
	}

	apiURL := fmt.Sprintf(
		"%s/api/v4/projects/%d/merge_requests/%d/pipelines",
		config.GitLabURL, config.ProjectID, mr.IID,
	)

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
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var pipelines []Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipelines); err != nil {
		return nil, err
	}

	for i := range pipelines {
		if pipelines[i].SHA == mr.SHA {
			return &pipelines[i], nil
		}
	}

	return nil, nil
}

// shortSHA abbreviates a commit hash for log output, the way git does.
func shortSHA(sha string) string {
	const shortLen = 8
	if len(sha) <= shortLen {
		return sha
	}
	return sha[:shortLen]
}

// urlSuffix renders ": <url>" when there is a URL, and nothing when there is not.
func urlSuffix(webURL string) string {
	if webURL == "" {
		return ""
	}
	return ": " + webURL
}

func createHTTPClient(insecure bool) *http.Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if insecure {
		// #nosec G402 -- certificate verification is disabled only at the user's
		// explicit request, via --insecure/-k.
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = tr
	}

	return client
}

func getProject(client *http.Client, config *Config) (*Project, error) {
	apiURL := fmt.Sprintf("%s/api/v4/projects/%d", config.GitLabURL, config.ProjectID)

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
	params := url.Values{}
	params.Set("state", "opened")
	params.Set("source_branch", config.SourceBranch)
	params.Set("target_branch", config.TargetBranch)
	params.Set("per_page", "1")
	apiURL := fmt.Sprintf("%s/api/v4/projects/%d/merge_requests?%s",
		config.GitLabURL, config.ProjectID, params.Encode())

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

	// #nosec G304 -- the path comes from the caller's own --description flag and the
	// tool runs with the caller's rights, so there is no privilege boundary to cross.
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

	apiURL := fmt.Sprintf("%s/api/v4/projects/%d/issues/%d", config.GitLabURL, config.ProjectID, issueID)

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
		return nil, fmt.Errorf("issue #%d not found", issueID)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func createMR(client *http.Client, config *Config, mrRequest *MRCreateRequest) (*MergeRequest, error) {
	apiURL := fmt.Sprintf("%s/api/v4/projects/%d/merge_requests", config.GitLabURL, config.ProjectID)

	jsonData, err := json.Marshal(mrRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
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
	apiURL := fmt.Sprintf("%s/api/v4/projects/%d/merge_requests/%d", config.GitLabURL, config.ProjectID, mrIID)

	jsonData, err := json.Marshal(updateRequest)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", apiURL, bytes.NewBuffer(jsonData))
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
	apiURL := fmt.Sprintf("%s/api/v4/projects/%d/merge_requests/%d/merge", config.GitLabURL, config.ProjectID, mrIID)

	acceptRequest := &MRAcceptRequest{
		MergeWhenPipelineSucceeds: true,
		ShouldRemoveSourceBranch:  config.RemoveBranch,
		Squash:                    config.SquashCommits,
	}

	jsonData, err := json.Marshal(acceptRequest)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", apiURL, bytes.NewBuffer(jsonData))
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

func boolPtr(b bool) *bool {
	return &b
}

// parseStringSlice splits a comma-separated flag value, trimming each element
// and dropping empty ones. Returns nil for an empty input so the field stays
// omitted from the JSON request rather than being sent as an empty list.
func parseStringSlice(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
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
