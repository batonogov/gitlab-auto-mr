# GitLab Auto MR

Create MR in GitLab automatically (Go implementation)

This is a Go implementation of the GitLab Auto MR tool for automatically creating Merge Requests from GitLab CI pipelines.

## Usage in .gitlab-ci.yml

```yaml
open_merge_request:
  stage: prepare
  image: ghcr.io/your-username/gitlab-auto-mr:latest
  script:
    - >
      gitlab_auto_mr \
        --target-branch dev \
        --commit-prefix Draft \
        --description ./.gitlab/merge_request/merge_request.md \
        --remove-branch \
        --use-issue-name
  rules:
    - if: $CI_COMMIT_BRANCH != "dev" && $CI_PIPELINE_SOURCE != "merge_request_event"
```

## Command Line Options

- `--target-branch`: Target branch for the merge request (required)
- `--commit-prefix`: Prefix to add to the commit message (optional)
- `--description`: Path to a file containing merge request description (optional)
- `--remove-branch`: Remove source branch after merge (flag)
- `--use-issue-name`: Use issue name for merge request title (flag)

## Environment Variables

This tool relies on GitLab CI environment variables:

- `GITLAB_TOKEN`: Personal Access Token with API access (required)
- `CI_SERVER_URL`: GitLab server URL (defaults to https://gitlab.com if not set)
- `CI_PROJECT_ID`: GitLab project ID
- `CI_PROJECT_PATH`: GitLab project path
- `CI_COMMIT_REF_NAME`: Current branch name
- `CI_COMMIT_TITLE`: Current commit title

## Building

```bash
go mod tidy
go build -o gitlab-auto-mr
```

## Running with Docker

```bash
docker run \
  -e GITLAB_TOKEN=${YOUR_GITLAB_TOKEN} \
  -e CI_SERVER_URL=https://gitlab.com \
  -e CI_PROJECT_ID=12345 \
  -e CI_PROJECT_PATH=username/project \
  -e CI_COMMIT_REF_NAME=feature/123-feature \
  -e CI_COMMIT_TITLE="Add new feature" \
  ghcr.io/your-username/gitlab-auto-mr:latest \
  --target-branch main \
  --remove-branch
```