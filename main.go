package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"
)

func main() {
	var (
		targetBranch    = flag.String("target-branch", "", "Target branch for the merge request")
		commitPrefix    = flag.String("commit-prefix", "", "Prefix to add to the commit message")
		descriptionFile = flag.String("description", "", "Path to a file containing merge request description")
		removeBranch    = flag.Bool("remove-branch", false, "Remove source branch after merge")
		useIssueName    = flag.Bool("use-issue-name", false, "Use issue name for merge request title")
		gitlabToken     = os.Getenv("GITLAB_TOKEN")
		gitlabURL       = os.Getenv("CI_SERVER_URL")
		projectID       = os.Getenv("CI_PROJECT_ID")
		sourceBranch    = os.Getenv("CI_COMMIT_REF_NAME")
		commitTitle     = os.Getenv("CI_COMMIT_TITLE")
		issueIID        = extractIssueIID(sourceBranch)
	)

	flag.Parse()

	if gitlabToken == "" {
		fmt.Println("GITLAB_TOKEN environment variable is required")
		os.Exit(1)
	}

	if *targetBranch == "" {
		fmt.Println("Target branch is required")
		os.Exit(1)
	}

	if gitlabURL == "" {
		gitlabURL = "https://gitlab.com"
	}

	// Initialize GitLab client
	git, err := gitlab.NewClient(gitlabToken, gitlab.WithBaseURL(gitlabURL+"/api/v4"))
	if err != nil {
		fmt.Printf("Failed to create GitLab client: %v\n", err)
		os.Exit(1)
	}

	// Prepare merge request title
	var mrTitle string
	if *useIssueName && issueIID != "" {
		issue, _, err := git.Issues.GetIssue(projectID, extractIssueIIDAsInt(issueIID))
		if err != nil {
			fmt.Printf("Failed to get issue details: %v\n", err)
			// Fallback to commit title
			mrTitle = getTitle(*commitPrefix, commitTitle)
		} else {
			mrTitle = getTitle(*commitPrefix, issue.Title)
		}
	} else {
		// Use commit title as MR title
		mrTitle = getTitle(*commitPrefix, commitTitle)
	}

	// Prepare description
	description := ""
	if *descriptionFile != "" {
		content, err := os.ReadFile(*descriptionFile)
		if err != nil {
			fmt.Printf("Failed to read description file: %v\n", err)
		} else {
			description = string(content)
		}
	}

	// Check if MR already exists
	mrs, _, err := git.MergeRequests.ListProjectMergeRequests(projectID, &gitlab.ListProjectMergeRequestsOptions{
		SourceBranch: &sourceBranch,
		TargetBranch: targetBranch,
		State:        gitlab.String("opened"),
	})

	if err != nil {
		fmt.Printf("Failed to check existing merge requests: %v\n", err)
		os.Exit(1)
	}

	if len(mrs) > 0 {
		fmt.Printf("Merge request already exists: %s\n", mrs[0].WebURL)
		os.Exit(0)
	}

	// Create merge request
	mrOpts := &gitlab.CreateMergeRequestOptions{
		Title:              &mrTitle,
		SourceBranch:       &sourceBranch,
		TargetBranch:       targetBranch,
		RemoveSourceBranch: removeBranch,
	}

	if description != "" {
		mrOpts.Description = &description
	}

	mr, _, err := git.MergeRequests.CreateMergeRequest(projectID, mrOpts)
	if err != nil {
		fmt.Printf("Failed to create merge request: %v\n", err)
		os.Exit(1)
	}

	// Link issue if available
	if issueIID != "" {
		_, _, err = git.Issues.UpdateIssue(projectID, extractIssueIIDAsInt(issueIID), &gitlab.UpdateIssueOptions{
			StateEvent: gitlab.String("close"),
		})
		if err != nil {
			fmt.Printf("Warning: Failed to close linked issue: %v\n", err)
		}
	}

	fmt.Printf("Created merge request: %s\n", mr.WebURL)
}

// Extract issue IID from branch name (e.g., "feature/123-description" -> "123")
func extractIssueIID(branchName string) string {
	findNumericID := func(s string) string {
		parts := strings.Split(s, "-")
		if len(parts) > 0 {
			if _, err := strconv.Atoi(parts[0]); err == nil {
				return parts[0]
			}
		}
		return ""
	}

	parts := strings.Split(branchName, "/")
	if len(parts) >= 2 {
		if id := findNumericID(parts[1]); id != "" {
			return id
		}
	}
	return findNumericID(branchName)
}

// Convert issue IID string to int
func extractIssueIIDAsInt(iid string) int {
	var issueIID int
	n, err := fmt.Sscanf(iid, "%d", &issueIID)
	if err != nil || n != 1 || fmt.Sprintf("%d", issueIID) != iid {
		return 0
	}
	return issueIID
}

// Prepare title with optional prefix
func getTitle(prefix, title string) string {
	if prefix == "" {
		return title
	}
	if strings.HasPrefix(title, prefix) {
		return title
	}
	return prefix + ": " + title
}
