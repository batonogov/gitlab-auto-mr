package main

import (
	"fmt"
	"time"

	"github.com/xanzy/go-gitlab"
)

// PipelineStatus represents the status of a pipeline
type PipelineStatus struct {
	Status    string
	Completed bool
	Success   bool
}

// waitForPipeline waits for the pipeline to complete and returns its status
func waitForPipeline(git *gitlab.Client, projectID string, pipelineID int, timeout int) (*PipelineStatus, error) {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)

	for time.Now().Before(deadline) {
		pipeline, _, err := git.Pipelines.GetPipeline(projectID, pipelineID)
		if err != nil {
			return nil, fmt.Errorf("failed to get pipeline status: %v", err)
		}

		status := &PipelineStatus{
			Status:    pipeline.Status,
			Completed: pipeline.Status != "pending" && pipeline.Status != "running",
			Success:   pipeline.Status == "success",
		}

		if status.Completed {
			return status, nil
		}

		// Wait before next check to avoid API rate limiting
		// In tests, we'll use a shorter interval
		if timeout < 60 { // If timeout is less than 1 minute, assume we're testing
			time.Sleep(100 * time.Millisecond)
		} else {
			time.Sleep(10 * time.Second)
		}
	}

	return nil, fmt.Errorf("pipeline check timed out after %d seconds", timeout)
}

// getPipelineID gets the pipeline ID for the current commit
func getPipelineID(git *gitlab.Client, projectID, commitSHA string) (int, error) {
	pipelines, _, err := git.Pipelines.ListProjectPipelines(projectID, &gitlab.ListProjectPipelinesOptions{
		SHA: &commitSHA,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list pipelines: %v", err)
	}

	if len(pipelines) == 0 {
		return 0, fmt.Errorf("no pipeline found for commit %s", commitSHA)
	}

	return pipelines[0].ID, nil
}
