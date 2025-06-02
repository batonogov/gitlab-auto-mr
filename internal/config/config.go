package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration options for the application
type Config struct {
	PrivateToken       string
	SourceBranch       string
	TargetBranch       string
	ProjectID          int
	GitLabURL          string
	UserIDs            []int
	ReviewerIDs        []int
	Insecure           bool
	CommitPrefix       string
	RemoveBranch       bool
	SquashCommits      bool
	DescriptionFile    string
	Title              string
	UseIssueName       bool
	AllowCollaboration bool
	MRExists           bool
}

// ParseFlags parses command line flags and environment variables
func ParseFlags() (*Config, error) {
	var config Config
	var userIDsStr, reviewerIDsStr string

	// Define flags
	flag.StringVar(&config.PrivateToken, "private-token", "", "Private GITLAB token, used to authenticate when calling the MR API")
	flag.StringVar(&config.SourceBranch, "source-branch", "", "The source branch to merge from")
	flag.StringVar(&config.TargetBranch, "target-branch", "main", "The target branch to merge onto")
	flag.StringVar(&config.TargetBranch, "t", "main", "The target branch to merge onto (short)")
	flag.IntVar(&config.ProjectID, "project-id", 0, "The project ID on GitLab to create the MR for")
	flag.StringVar(&config.GitLabURL, "gitlab-url", "", "The GitLab URL i.e. gitlab.com")
	flag.StringVar(&userIDsStr, "user-id", "", "The GitLab user ID(s) to assign the created MR to (comma-separated)")
	flag.StringVar(&reviewerIDsStr, "reviewer-id", "", "The GitLab user ID(s) to assign the created MR to as reviewer(s) (comma-separated)")
	flag.BoolVar(&config.Insecure, "insecure", false, "Do not verify server SSL certificate")
	flag.BoolVar(&config.Insecure, "k", false, "Do not verify server SSL certificate (short)")
	flag.StringVar(&config.CommitPrefix, "commit-prefix", "Draft", "Prefix for the MR title")
	flag.StringVar(&config.CommitPrefix, "c", "Draft", "Prefix for the MR title (short)")
	flag.BoolVar(&config.RemoveBranch, "remove-branch", false, "If set will remove the source branch after MR")
	flag.BoolVar(&config.RemoveBranch, "r", false, "If set will remove the source branch after MR (short)")
	flag.BoolVar(&config.SquashCommits, "squash-commits", false, "If set will squash commits on merge")
	flag.BoolVar(&config.SquashCommits, "s", false, "If set will squash commits on merge (short)")
	flag.StringVar(&config.DescriptionFile, "description", "", "Path to file to use as the description for the MR")
	flag.StringVar(&config.DescriptionFile, "d", "", "Path to file to use as the description for the MR (short)")
	flag.StringVar(&config.Title, "title", "", "Custom title for the MR")
	flag.BoolVar(&config.UseIssueName, "use-issue-name", false, "If set will use information from issue in branch name, must be in the form #issue-number")
	flag.BoolVar(&config.UseIssueName, "i", false, "If set will use information from issue in branch name (short)")
	flag.BoolVar(&config.AllowCollaboration, "allow-collaboration", false, "If set allow commits from members who can merge to the target branch")
	flag.BoolVar(&config.AllowCollaboration, "a", false, "If set allow commits from members who can merge to the target branch (short)")
	flag.BoolVar(&config.MRExists, "mr-exists", false, "If set, checks if a merge request exists for the given 'source' and 'target' branch and exits without creating it")

	// Version flag (handled in main before flag parsing)
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&showVersion, "v", false, "Show version information and exit (short)")

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Gitlab Auto MR Tool - automatically creates merge requests in GitLab.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  GITLAB_PRIVATE_TOKEN - GitLab private token (overridden by --private-token)\n")
		fmt.Fprintf(os.Stderr, "  CI_COMMIT_REF_NAME   - Source branch name (overridden by --source-branch)\n")
		fmt.Fprintf(os.Stderr, "  CI_PROJECT_ID        - GitLab project ID (overridden by --project-id)\n")
		fmt.Fprintf(os.Stderr, "  CI_PROJECT_URL       - GitLab project URL (overridden by --gitlab-url)\n")
		fmt.Fprintf(os.Stderr, "  GITLAB_USER_ID       - User ID to assign MR to (overridden by --user-id)\n")
		fmt.Fprintf(os.Stderr, "\nVersion:\n")
		fmt.Fprintf(os.Stderr, "  Use --version or -v to show version information\n")
	}

	flag.Parse()

	// Apply environment variable defaults if flags not set
	if config.PrivateToken == "" {
		config.PrivateToken = os.Getenv("GITLAB_PRIVATE_TOKEN")
	}

	if config.SourceBranch == "" {
		config.SourceBranch = os.Getenv("CI_COMMIT_REF_NAME")
	}

	if config.ProjectID == 0 {
		if projectIDStr := os.Getenv("CI_PROJECT_ID"); projectIDStr != "" {
			if id, err := strconv.Atoi(projectIDStr); err == nil {
				config.ProjectID = id
			}
		}
	}

	if config.GitLabURL == "" {
		if projectURL := os.Getenv("CI_PROJECT_URL"); projectURL != "" {
			// Extract base URL from project URL
			// e.g., https://gitlab.com/group/project -> https://gitlab.com
			parts := strings.Split(projectURL, "/")
			if len(parts) >= 3 {
				config.GitLabURL = strings.Join(parts[:3], "/")
			}
		}
	}

	if userIDsStr == "" {
		userIDsStr = os.Getenv("GITLAB_USER_ID")
	}

	// Parse user IDs
	if userIDsStr != "" {
		var err error
		config.UserIDs, err = parseUserIDs(userIDsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid user IDs: %w", err)
		}
	}

	// Parse reviewer IDs
	if reviewerIDsStr != "" {
		var err error
		config.ReviewerIDs, err = parseUserIDs(reviewerIDsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid reviewer IDs: %w", err)
		}
	}

	return &config, nil
}

// Validate checks if all required configuration is present
func (c *Config) Validate() error {
	if c.PrivateToken == "" {
		return fmt.Errorf("private token is required (use --private-token or GITLAB_PRIVATE_TOKEN environment variable)")
	}

	if c.SourceBranch == "" {
		return fmt.Errorf("source branch is required (use --source-branch or CI_COMMIT_REF_NAME environment variable)")
	}

	if c.ProjectID == 0 {
		return fmt.Errorf("project ID is required (use --project-id or CI_PROJECT_ID environment variable)")
	}

	if c.GitLabURL == "" {
		return fmt.Errorf("GitLab URL is required (use --gitlab-url or CI_PROJECT_URL environment variable)")
	}

	// Validate GitLab URL format
	if !strings.HasPrefix(c.GitLabURL, "http://") && !strings.HasPrefix(c.GitLabURL, "https://") {
		c.GitLabURL = "https://" + c.GitLabURL
	}

	return nil
}

// parseUserIDs parses a comma-separated string of user IDs into a slice of integers
func parseUserIDs(userIDsStr string) ([]int, error) {
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
