package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
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

// Defaults for the HTTP behavior. They are applied where they are used, not
// only in parseFlags, so a zero-valued Config still behaves like the tool did
// before these flags existed.
const (
	defaultTimeout    = 30 * time.Second
	defaultRetryDelay = time.Second
	maxRetryDelay     = 30 * time.Second
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
	Timeout            time.Duration
	Retries            int
	RetryDelay         time.Duration
	CACert             string
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

	// Canceling on SIGINT/SIGTERM aborts the in-flight request instead of
	// leaving the process to sit out the client timeout. stop() is called
	// directly rather than deferred, since os.Exit below would skip a defer.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	runErr := run(ctx, config)
	stop()

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", runErr)
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
	flag.StringVar(&config.CACert, "ca-cert", getEnv("GITLAB_AUTO_MR_CA_CERT", ""),
		"Path to a PEM CA certificate to trust in addition to the system pool")
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
	flag.DurationVar(&config.Timeout, "timeout", getEnvDuration("GITLAB_AUTO_MR_TIMEOUT", defaultTimeout),
		"Timeout for a single GitLab API request")
	flag.IntVar(&config.Retries, "retries", getEnvInt("GITLAB_AUTO_MR_RETRIES", 2),
		"Retries for transient GitLab failures (network errors, 5xx, 429)")
	flag.DurationVar(&config.RetryDelay, "retry-delay",
		getEnvDuration("GITLAB_AUTO_MR_RETRY_DELAY", defaultRetryDelay),
		"Delay before the first retry, doubled on each further attempt")
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

	if config.Timeout <= 0 {
		return nil, fmt.Errorf("--timeout must be positive, got %s", config.Timeout)
	}
	if config.Retries < 0 {
		return nil, fmt.Errorf("--retries must not be negative, got %d", config.Retries)
	}
	if config.RetryDelay < 0 {
		return nil, fmt.Errorf("--retry-delay must not be negative, got %s", config.RetryDelay)
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
	if config.CACert != "" && config.Insecure {
		return fmt.Errorf(
			"--ca-cert cannot be used with --insecure: " +
				"--insecure disables verification entirely, which would make the CA pointless",
		)
	}

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

func run(ctx context.Context, config *Config) error {
	if err := validateConfig(config); err != nil {
		return err
	}

	client, err := createHTTPClient(config)
	if err != nil {
		return err
	}

	project, err := getProject(ctx, client, config)
	if err != nil {
		return fmt.Errorf("unable to get project %d: %w", config.ProjectID, err)
	}

	if config.TargetBranch == "" {
		config.TargetBranch = project.DefaultBranch
	}

	if err := validateMR(config.SourceBranch, config.TargetBranch); err != nil {
		return err
	}

	existingMR, err := getExistingMR(ctx, client, config)
	if err != nil {
		return fmt.Errorf("failed to check if MR exists: %w", err)
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

	mr, err := handleMR(ctx, client, config, existingMR, title, description)
	if err != nil {
		return err
	}
	if mr == nil {
		// Defensive: every handleMR branch returns an MR on success.
		mr = &MergeRequest{}
	}

	if config.TriggerPipeline {
		if err := triggerMRPipeline(ctx, client, config, mr); err != nil {
			return fmt.Errorf("failed to trigger merge request pipeline: %w", err)
		}
	}

	if config.AutoMerge {
		return enableAutoMerge(ctx, client, config, mr.IID)
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
	ctx context.Context, client *http.Client, config *Config,
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
		return handleUpdateMR(ctx, client, config, existingMR, title, description)

	default:
		return handleCreateMR(ctx, client, config, title, description)
	}
}

func handleUpdateMR(
	ctx context.Context, client *http.Client, config *Config,
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

	updateRequest.MilestoneID, updateRequest.Labels = resolveMRMetadata(ctx, client, config)

	if err := updateMR(ctx, client, config, existingMR.IID, updateRequest); err != nil {
		return nil, fmt.Errorf("failed to update MR: %w", err)
	}

	fmt.Printf("Updated existing MR %s (IID: %d)\n", title, existingMR.IID)
	printMRURL(existingMR)
	return existingMR, nil
}

func handleCreateMR(
	ctx context.Context, client *http.Client, config *Config,
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

	mrRequest.MilestoneID, mrRequest.Labels = resolveMRMetadata(ctx, client, config)

	createdMR, err := createMR(ctx, client, config, mrRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create MR: %w", err)
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
func resolveMRMetadata(ctx context.Context, client *http.Client, config *Config) (int, []string) {
	milestoneID := config.MilestoneID
	labels := config.Labels

	if !config.UseIssueName {
		return milestoneID, labels
	}

	issue, err := getIssueData(ctx, client, config)
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

func enableAutoMerge(ctx context.Context, client *http.Client, config *Config, mrIID int) error {
	if mrIID == 0 {
		fmt.Println("Warning: could not determine MR IID, skipping auto-merge")
		return nil
	}

	if err := acceptMR(ctx, client, config, mrIID); err != nil {
		return fmt.Errorf("failed to enable auto-merge: %w", err)
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
func triggerMRPipeline(ctx context.Context, client *http.Client, config *Config, mr *MergeRequest) error {
	if mr == nil || mr.IID == 0 {
		fmt.Println("Warning: could not determine MR IID, skipping pipeline trigger")
		return nil
	}

	if !config.ForcePipeline {
		existing, err := findPipelineForSHA(ctx, client, config, mr)
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

	body, err := doRequest(ctx, client, config, http.MethodPost,
		fmt.Sprintf("projects/%d/merge_requests/%d/pipelines", config.ProjectID, mr.IID), nil)
	if err != nil {
		var apiErr *apiError
		if errors.As(err, &apiErr) {
			switch apiErr.StatusCode {
			case http.StatusForbidden:
				return fmt.Errorf(
					"forbidden, the token owner needs at least the Developer role on the project " +
						"to create pipelines",
				)
			case http.StatusBadRequest:
				return fmt.Errorf(
					"GitLab refused to create the pipeline, "+
						"the CI configuration may define no jobs for merge request pipelines: %s",
					apiErr.Body,
				)
			}
		}
		return err
	}

	// The pipeline already exists at this point, so a response we cannot read is
	// worth a warning but must not fail the run: the body is only used to tell
	// the user what was created.
	var pipeline Pipeline
	if err := json.Unmarshal(body, &pipeline); err != nil {
		fmt.Printf("Warning: merge request pipeline created but response could not be read: %v\n", err)
		return nil
	}

	fmt.Printf(
		"Merge request pipeline created (ID: %d, status: %s)%s\n",
		pipeline.ID, pipeline.Status, urlSuffix(pipeline.WebURL),
	)
	return nil
}

// findPipelineForSHA returns the MR's existing pipeline for its head commit, or
// nil when there is none.
//
// With an unknown head SHA there is nothing to compare against, so it reports
// no match and the caller creates a pipeline: a duplicate is a better outcome
// than silently skipping the checks.
func findPipelineForSHA(
	ctx context.Context, client *http.Client, config *Config, mr *MergeRequest,
) (*Pipeline, error) {
	if mr.SHA == "" {
		return nil, nil
	}

	body, err := doRequest(ctx, client, config, http.MethodGet,
		fmt.Sprintf("projects/%d/merge_requests/%d/pipelines", config.ProjectID, mr.IID), nil)
	if err != nil {
		return nil, err
	}

	var pipelines []Pipeline
	if err := json.Unmarshal(body, &pipelines); err != nil {
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

func createHTTPClient(config *Config) (*http.Client, error) {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	client := &http.Client{
		Timeout: timeout,
	}

	switch {
	case config.Insecure:
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // user-requested via --insecure flag
		}

	case config.CACert != "":
		pool, err := caCertPool(config.CACert)
		if err != nil {
			return nil, err
		}
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
		}
	}

	return client, nil
}

// caCertPool returns the system trust store with the given PEM certificate
// added. It is additive on purpose: a self-hosted GitLab behind an internal CA
// is usually reached alongside other hosts with public certificates.
func caCertPool(path string) (*x509.CertPool, error) {
	pem, err := os.ReadFile(path) //nolint:gosec // path is the caller's own --ca-cert value
	if err != nil {
		return nil, fmt.Errorf("unable to read CA certificate %s: %w", path, err)
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		// Windows returns an error here rather than a pool; starting from an
		// empty one still lets the given CA work.
		pool = x509.NewCertPool()
	}

	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("no PEM certificate found in %s", path)
	}

	return pool, nil
}

// apiError is returned by doRequest when GitLab answered with a status the
// helper does not treat as success. Callers use errors.As to give the statuses
// that mean something specific to their endpoint a message of their own; the
// rest fall through to Error(), which is the wording every API function used to
// build by hand.
type apiError struct {
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// errUnauthorized is returned for every 401. GitLab answers 401 for the same
// reason on every endpoint, so the advice lives in one place rather than being
// re-worded per function.
var errUnauthorized = errors.New("unauthorized access, check your access token is valid and has the api scope")

// doRequest performs one GitLab API call and returns the raw response body,
// retrying the attempt when the failure looks transient.
//
// path is relative to /api/v4 and must already be escaped. body, when non-nil,
// is sent as JSON. Any 2xx is success; 401 yields errUnauthorized and every
// other status yields *apiError carrying the code and body.
//
// Decoding is left to the caller: the bodies are single objects, small enough
// to hold in memory, and each caller has its own wording for a malformed one.
func doRequest(
	ctx context.Context, client *http.Client, config *Config,
	method, path string, body any,
) ([]byte, error) {
	apiURL := fmt.Sprintf("%s/api/v4/%s", config.GitLabURL, path)

	var jsonData []byte
	if body != nil {
		var err error
		if jsonData, err = json.Marshal(body); err != nil {
			return nil, err
		}
	}

	for attempt := 0; ; attempt++ {
		respBody, retryAfter, err := sendRequest(ctx, client, config, method, apiURL, jsonData, body != nil)
		if err == nil {
			return respBody, nil
		}

		if attempt >= config.Retries || !isRetryable(ctx, method, err) {
			return nil, err
		}

		delay := retryDelay(config, attempt, retryAfter)
		fmt.Fprintf(os.Stderr, "Warning: %s %s failed (%v), retrying in %s (%d/%d)\n",
			method, path, err, delay, attempt+1, config.Retries)

		if err := sleep(ctx, delay); err != nil {
			return nil, err
		}
	}
}

// sendRequest performs a single attempt. The returned duration is the value of
// a Retry-After header, when GitLab sent one that could be parsed.
func sendRequest(
	ctx context.Context, client *http.Client, config *Config,
	method, apiURL string, jsonData []byte, hasBody bool,
) ([]byte, time.Duration, error) {
	var reader io.Reader = http.NoBody
	if hasBody {
		reader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, reader)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("PRIVATE-TOKEN", config.PrivateToken)
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, 0, errUnauthorized
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, parseRetryAfter(resp.Header.Get("Retry-After")),
			&apiError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	return respBody, 0, nil
}

// isRetryable decides whether a failed attempt is worth repeating.
//
// The rule is about what a repeat could do, not only about what failed. GET and
// PUT are idempotent, so repeating one is harmless: at worst GitLab applies the
// same change twice. POST is not — retrying POST /merge_requests after a
// timeout could open a second MR — so it is repeated only when the request
// demonstrably never reached GitLab, which is what a dial failure means.
//
// Whether the caller gave up is read from ctx, not from err. A --timeout that
// elapsed also surfaces as context.DeadlineExceeded, and that one is a
// transient failure worth repeating — the very case retries exist for.
func isRetryable(ctx context.Context, method string, err error) bool {
	if ctx.Err() != nil {
		return false
	}

	if errors.Is(err, errUnauthorized) {
		return false
	}

	var apiErr *apiError
	if errors.As(err, &apiErr) {
		// GitLab answered, so the request did reach it: repeating is only safe
		// for the idempotent verbs.
		if !isIdempotent(method) {
			return false
		}
		return apiErr.StatusCode >= 500 || apiErr.StatusCode == http.StatusTooManyRequests
	}

	if isIdempotent(method) {
		return true
	}

	return isDialError(err)
}

func isIdempotent(method string) bool {
	return method == http.MethodGet || method == http.MethodPut || method == http.MethodHead
}

// isDialError reports whether the connection was never established, which means
// the server cannot have seen the request.
func isDialError(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Op == "dial"
	}
	return false
}

// retryDelay is the exponential backoff, unless GitLab asked for a specific
// wait via Retry-After — it knows better, and ignoring it on a 429 only earns
// another one.
func retryDelay(config *Config, attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if retryAfter > maxRetryDelay {
			return maxRetryDelay
		}
		return retryAfter
	}

	base := config.RetryDelay
	if base <= 0 {
		base = defaultRetryDelay
	}

	delay := base << attempt
	// Shifting past the width of the type wraps; both cases mean "too long".
	if delay > maxRetryDelay || delay <= 0 {
		return maxRetryDelay
	}
	return delay
}

// parseRetryAfter reads the delay-seconds form of Retry-After, which is what
// GitLab sends. The HTTP-date form is ignored rather than guessed at.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds < 0 {
		return 0
	}

	return time.Duration(seconds) * time.Second
}

// sleep waits for d, or returns early if the context is canceled first.
func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func getProject(ctx context.Context, client *http.Client, config *Config) (*Project, error) {
	body, err := doRequest(ctx, client, config, http.MethodGet,
		fmt.Sprintf("projects/%d", config.ProjectID), nil)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := json.Unmarshal(body, &project); err != nil {
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

func getExistingMR(ctx context.Context, client *http.Client, config *Config) (*MergeRequest, error) {
	params := url.Values{}
	params.Set("state", "opened")
	params.Set("source_branch", config.SourceBranch)
	params.Set("target_branch", config.TargetBranch)
	params.Set("per_page", "1")

	body, err := doRequest(ctx, client, config, http.MethodGet,
		fmt.Sprintf("projects/%d/merge_requests?%s", config.ProjectID, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	var mrs []MergeRequest
	if err := json.Unmarshal(body, &mrs); err != nil {
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

func getIssueData(ctx context.Context, client *http.Client, config *Config) (*Issue, error) {
	re := regexp.MustCompile(`#(\d+)`)
	matches := re.FindStringSubmatch(config.SourceBranch)
	if len(matches) < 2 {
		return nil, fmt.Errorf("issue number not found in %s", config.SourceBranch)
	}

	issueID, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid issue number: %s", matches[1])
	}

	body, err := doRequest(ctx, client, config, http.MethodGet,
		fmt.Sprintf("projects/%d/issues/%d", config.ProjectID, issueID), nil)
	if err != nil {
		// Any answer from GitLab other than success means the issue is not
		// usable here, whatever the status; transport errors pass through.
		var apiErr *apiError
		if errors.As(err, &apiErr) {
			return nil, fmt.Errorf("issue #%d not found", issueID)
		}
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func createMR(
	ctx context.Context, client *http.Client, config *Config, mrRequest *MRCreateRequest,
) (*MergeRequest, error) {
	body, err := doRequest(ctx, client, config, http.MethodPost,
		fmt.Sprintf("projects/%d/merge_requests", config.ProjectID), mrRequest)
	if err != nil {
		return nil, err
	}

	// GitLab always sends the created MR back, but an empty body is not a
	// failure: the MR exists, only its IID is unknown.
	var mr MergeRequest
	if len(body) == 0 {
		return &mr, nil
	}

	if err := json.Unmarshal(body, &mr); err != nil {
		return nil, fmt.Errorf("MR created but response is invalid: %v", err)
	}

	return &mr, nil
}

func updateMR(
	ctx context.Context, client *http.Client, config *Config, mrIID int, updateRequest *MRUpdateRequest,
) error {
	_, err := doRequest(ctx, client, config, http.MethodPut,
		fmt.Sprintf("projects/%d/merge_requests/%d", config.ProjectID, mrIID), updateRequest)
	return err
}

func acceptMR(ctx context.Context, client *http.Client, config *Config, mrIID int) error {
	acceptRequest := &MRAcceptRequest{
		MergeWhenPipelineSucceeds: true,
		ShouldRemoveSourceBranch:  config.RemoveBranch,
		Squash:                    config.SquashCommits,
	}

	_, err := doRequest(ctx, client, config, http.MethodPut,
		fmt.Sprintf("projects/%d/merge_requests/%d/merge", config.ProjectID, mrIID), acceptRequest)

	var apiErr *apiError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusMethodNotAllowed:
			return fmt.Errorf(
				"merge request cannot be merged, " +
					"the pipeline may not have started yet or other merge conditions are not met",
			)
		case http.StatusNotAcceptable:
			return fmt.Errorf(
				"merge request cannot be merged, " +
					"there may be unresolved discussions or other blocking conditions",
			)
		}
	}

	return err
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

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
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
