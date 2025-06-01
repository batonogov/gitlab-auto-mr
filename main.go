package main

import (
	"fmt"
	"log"
	"os"

	"gitlab-auto-mr/internal/config"
	"gitlab-auto-mr/internal/gitlab"
	"gitlab-auto-mr/internal/utils"
)

// Version information - set during build via ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	// Check for version flag before parsing all flags
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Printf("gitlab-auto-mr version %s\n", Version)
			fmt.Printf("Git commit: %s\n", GitCommit)
			fmt.Printf("Build date: %s\n", BuildDate)
			os.Exit(0)
		}
	}

	// Parse configuration from flags and environment variables
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatalf("Error parsing configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Sanitize branch names
	cfg.SourceBranch = utils.SanitizeBranchName(cfg.SourceBranch)
	cfg.TargetBranch = utils.SanitizeBranchName(cfg.TargetBranch)

	// Create GitLab client
	client, err := gitlab.NewClient(cfg.GitLabURL, cfg.PrivateToken, cfg.Insecure)
	if err != nil {
		log.Fatalf("Failed to create GitLab client: %v", err)
	}

	// Check if merge request already exists (if requested)
	if cfg.MRExists {
		exists, err := client.CheckMergeRequestExists(cfg.ProjectID, cfg.SourceBranch, cfg.TargetBranch)
		if err != nil {
			log.Fatalf("Failed to check if merge request exists: %v", err)
		}

		if exists {
			fmt.Printf("Merge request already exists for source branch '%s' and target branch '%s'\n",
				cfg.SourceBranch, cfg.TargetBranch)
			os.Exit(0)
		} else {
			fmt.Printf("No existing merge request found for source branch '%s' and target branch '%s'\n",
				cfg.SourceBranch, cfg.TargetBranch)
			os.Exit(0)
		}
	}

	// Check if merge request already exists (prevent duplicates)
	exists, err := client.CheckMergeRequestExists(cfg.ProjectID, cfg.SourceBranch, cfg.TargetBranch)
	if err != nil {
		log.Fatalf("Failed to check if merge request exists: %v", err)
	}

	if exists {
		fmt.Printf("Merge request already exists for source branch '%s' and target branch '%s'\n",
			cfg.SourceBranch, cfg.TargetBranch)
		os.Exit(0)
	}

	// Read description file if provided
	description, err := utils.ReadDescriptionFile(cfg.DescriptionFile)
	if err != nil {
		log.Fatalf("Failed to read description file: %v", err)
	}

	// Generate MR title
	title, err := utils.GenerateMRTitle(cfg.SourceBranch, cfg.CommitPrefix, cfg.Title, cfg.UseIssueName)
	if err != nil {
		log.Fatalf("Failed to generate merge request title: %v", err)
	}

	// Prepare merge request options
	mrOpts := &gitlab.MergeRequestOptions{
		SourceBranch:       cfg.SourceBranch,
		TargetBranch:       cfg.TargetBranch,
		Title:              title,
		Description:        description,
		AssigneeIDs:        cfg.UserIDs,
		ReviewerIDs:        cfg.ReviewerIDs,
		RemoveSourceBranch: cfg.RemoveBranch,
		Squash:             cfg.SquashCommits,
		AllowCollaboration: cfg.AllowCollaboration,
	}

	// Create merge request
	fmt.Printf("Creating merge request...\n")
	fmt.Printf("  Source branch: %s\n", cfg.SourceBranch)
	fmt.Printf("  Target branch: %s\n", cfg.TargetBranch)
	fmt.Printf("  Title: %s\n", title)
	if len(cfg.UserIDs) > 0 {
		fmt.Printf("  Assignees: %s\n", utils.FormatUserIDs(cfg.UserIDs))
	}
	if len(cfg.ReviewerIDs) > 0 {
		fmt.Printf("  Reviewers: %s\n", utils.FormatUserIDs(cfg.ReviewerIDs))
	}

	mr, err := client.CreateMergeRequest(cfg.ProjectID, mrOpts)
	if err != nil {
		log.Fatalf("Failed to create merge request: %v", err)
	}

	// Success output
	fmt.Printf("\n✅ Merge request created successfully!\n")
	fmt.Printf("  ID: %d\n", mr.ID)
	fmt.Printf("  IID: %d\n", mr.IID)
	fmt.Printf("  Title: %s\n", mr.Title)
	fmt.Printf("  Web URL: %s\n", mr.WebURL)
	fmt.Printf("  State: %s\n", mr.State)
	fmt.Printf("  Created by: gitlab-auto-mr v%s\n", Version)

	if mr.RemoveSourceBranch {
		fmt.Printf("  ⚠️  Source branch will be removed after merge\n")
	}
	if mr.Squash {
		fmt.Printf("  📦 Commits will be squashed on merge\n")
	}
}
