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
		squashCommits   = flag.Bool("squash-commits", false, "Squash commits in the merge request")
		autoMerge       = flag.Bool("auto-merge", false, "Enable auto-merge for the merge request")
		reviewers       = flag.String("reviewers", "", "Comma-separated list of reviewer usernames")
		milestone       = flag.String("milestone", "", "Milestone ID for the merge request")
		assignee        = flag.String("assignee", "", "Username of the assignee")
		waitPipeline    = flag.Bool("wait-pipeline", false, "Wait for pipeline to complete before creating MR")
		pipelineTimeout = flag.Int("pipeline-timeout", 3600, "Maximum time to wait for pipeline in seconds (default 1h)")
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

	commitSHA := os.Getenv("CI_COMMIT_SHA")
	if *waitPipeline && commitSHA == "" {
		fmt.Println("CI_COMMIT_SHA environment variable is required when using --wait-pipeline")
		os.Exit(1)
	}

	// Initialize GitLab client
	//nolint:staticcheck // TODO: migrate to new client when it's stable
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
	state := "opened"
	mrs, _, err := git.MergeRequests.ListProjectMergeRequests(projectID, &gitlab.ListProjectMergeRequestsOptions{
		SourceBranch: &sourceBranch,
		TargetBranch: targetBranch,
		State:        &state,
	})

	if err != nil {
		fmt.Printf("Failed to check existing merge requests: %v\n", err)
		os.Exit(1)
	}

	if len(mrs) > 0 {
		fmt.Printf("Merge request already exists: %s\n", mrs[0].WebURL)
		os.Exit(0)
	}

	// Check pipeline status if requested
	if *waitPipeline {
		fmt.Printf("Waiting for pipeline to complete (timeout: %d seconds)...\n", *pipelineTimeout)

		pipelineID, err := getPipelineID(git, projectID, commitSHA)
		if err != nil {
			fmt.Printf("Failed to get pipeline ID: %v\n", err)
			os.Exit(1)
		}

		status, err := waitForPipeline(git, projectID, pipelineID, *pipelineTimeout)
		if err != nil {
			fmt.Printf("Pipeline check failed: %v\n", err)
			os.Exit(1)
		}

		if !status.Success {
			fmt.Printf("Pipeline failed with status: %s\n", status.Status)
			os.Exit(1)
		}

		fmt.Println("Pipeline completed successfully")
	}

	// Prepare additional options
	var reviewerIDs []int
	if *reviewers != "" {
		for _, username := range strings.Split(*reviewers, ",") {
			users, _, err := git.Users.ListUsers(&gitlab.ListUsersOptions{
				Username: &username,
			})
			if err != nil || len(users) == 0 {
				fmt.Printf("Warning: Failed to get reviewer ID for %s: %v\n", username, err)
				continue
			}
			reviewerIDs = append(reviewerIDs, users[0].ID)
		}
	}

	var milestoneID *int
	if *milestone != "" {
		mid, err := strconv.Atoi(*milestone)
		if err == nil {
			milestoneID = &mid
		} else {
			fmt.Printf("Warning: Invalid milestone ID: %s\n", *milestone)
		}
	}

	var assigneeID *int
	if *assignee != "" {
		users, _, err := git.Users.ListUsers(&gitlab.ListUsersOptions{
			Username: assignee,
		})
		if err != nil || len(users) == 0 {
			fmt.Printf("Warning: Failed to get assignee ID for %s: %v\n", *assignee, err)
		} else {
			assigneeID = &users[0].ID
		}
	}

	// Create merge request
	mrOpts := &gitlab.CreateMergeRequestOptions{
		Title:              &mrTitle,
		SourceBranch:       &sourceBranch,
		TargetBranch:       targetBranch,
		RemoveSourceBranch: removeBranch,
		Squash:             squashCommits,
		AssigneeID:         assigneeID,
		MilestoneID:        milestoneID,
		ReviewerIDs:        &reviewerIDs,
	}

	if description != "" {
		mrOpts.Description = &description
	}

	mr, _, err := git.MergeRequests.CreateMergeRequest(projectID, mrOpts)
	if err != nil {
		fmt.Printf("Failed to create merge request: %v\n", err)
		os.Exit(1)
	}

	// Enable auto-merge if requested
	if *autoMerge {
		_, _, err = git.MergeRequests.AcceptMergeRequest(projectID, mr.IID, &gitlab.AcceptMergeRequestOptions{
			ShouldRemoveSourceBranch: removeBranch,
			Squash:                   squashCommits,
		})
		if err != nil {
			fmt.Printf("Warning: Failed to enable auto-merge: %v\n", err)
		}
	}

	// Link issue if available
	if issueIID != "" {
		stateEvent := "close"
		_, _, err = git.Issues.UpdateIssue(projectID, extractIssueIIDAsInt(issueIID), &gitlab.UpdateIssueOptions{
			StateEvent: &stateEvent,
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
