# GitLab Auto MR

Create MR in GitLab automatically (Go implementation)

This is a Go implementation of the GitLab Auto MR tool for automatically creating Merge Requests from GitLab CI pipelines.

## Development

### Prerequisites

- Go 1.x or higher
- [Task](https://taskfile.dev/) (task runner)
- Python (for pre-commit)
- pip (for pre-commit installation)

### Setup Development Environment

1. Clone the repository
2. Install Task runner (if not installed):
   ```bash
   # On macOS
   brew install go-task/tap/go-task
   # Other platforms: https://taskfile.dev/installation/
   ```

### Development with Task

Task is a task runner / build tool that aims to be simpler and easier to use than, for example, GNU Make.

#### Available Tasks

```bash
task --list
```

Main development commands:
- `task build` - build the binary
- `task test` - run tests
- `task test:cover` - run tests with coverage and generate HTML report
- `task lint` - run linters
- `task clean` - clean build artifacts
- `task dev:setup` - setup development environment

Docker-related tasks:
- `task docker:build` - build Docker image
- `task docker:run` - run in Docker (requires environment variables)

Additional commands:
- `task install-deps` - install development dependencies
- `task help` - show available commands

3. Install pre-commit hooks:
   ```bash
   task setup-pre-commit
   ```

### Available Tasks

You can use the following commands for development:

- `task test` - run tests
- `task test:cover` - run tests with coverage report
- `task lint` - run linters
- `task build` - build the application
- `task clean` - clean build artifacts
- `task all` - run all checks and tests

### Pre-commit Hooks

The project uses pre-commit hooks to ensure code quality. The following checks are run before each commit:

- Go tests
- Go linters (go vet)
- Code formatting (gofmt)

If any of these checks fail, the commit will be rejected. You can run the checks manually:
```bash
pre-commit run --all-files
```

### CI/CD Pipeline

The project uses GitHub Actions with the following jobs:

### Installation

You can install gitlab-auto-mr in several ways:

1. **Using pre-built binaries**:
   Download the latest release from [GitHub Releases](https://github.com/your-username/gitlab-auto-mr/releases)

2. **Using Docker**:
   ```bash
   docker pull ghcr.io/your-username/gitlab-auto-mr:latest
   ```

3. **From source**:
   ```bash
   go install github.com/your-username/gitlab-auto-mr@latest
   ```

### Release Process

The project uses GitHub Actions for automated releases:

1. Create and push a new tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. GitHub Actions will automatically:
   - Build binaries for multiple platforms (Linux, macOS, Windows)
   - Create a GitHub Release with changelog
   - Push Docker image to GitHub Container Registry (ghcr.io)
   - Tag Docker images with version and latest tags

1. **Test**:
   - Runs unit tests
   - Generates and uploads code coverage to Codecov
   - Runs tests with race detection

2. **Lint**:
   - Runs `go vet`
   - Checks code formatting

3. **Build**:
   - Compiles the application
   - Uploads binary as an artifact

4. **Integration**:
   - Runs integration tests (only on main branch)
   - Requires GitLab credentials in GitHub Secrets:
     - `GITLAB_TOKEN`
     - `GITLAB_URL`
     - `GITLAB_PROJECT_ID`

To run tests locally:
```bash
# Run unit tests
task test

# Run tests with coverage
task test:cover

# Run integration tests
go test -v -tags=integration ./...
```

## Usage in .gitlab-ci.yml

```yaml
open_merge_request:
  stage: prepare
  image: docker pull ghcr.io/batonogov/gitlab-auto-mr:latest
  script:
    - >
      gitlab_auto_mr \
        --target-branch dev \
        --commit-prefix Draft \
        --description ./.gitlab/merge_request/merge_request.md \
        --remove-branch \
        --use-issue-name \
        --wait-pipeline \
        --pipeline-timeout 1800 \
        --reviewers "tech-lead,senior-dev" \
        --assignee "product-owner" \
        --milestone 42 \
        --squash-commits \
        --auto-merge
  rules:
    - if: $CI_COMMIT_BRANCH != "dev" && $CI_PIPELINE_SOURCE != "merge_request_event"
    - when: on_success # Only create MR if the pipeline succeeds
```

## Command Line Options

### Required Options
- `--target-branch`: Target branch for the merge request

### Pipeline Control
- `--wait-pipeline`: Wait for pipeline to complete before creating MR
- `--pipeline-timeout`: Maximum time to wait for pipeline in seconds (default: 3600)

### MR Settings
- `--commit-prefix`: Prefix to add to the commit message
- `--description`: Path to a file containing merge request description
- `--remove-branch`: Remove source branch after merge
- `--use-issue-name`: Use issue name for merge request title
- `--squash-commits`: Squash commits in the merge request
- `--auto-merge`: Enable auto-merge for the merge request

### Reviewers and Assignment
- `--reviewers`: Comma-separated list of reviewer usernames
- `--assignee`: Username of the assignee
- `--milestone`: Milestone ID for the merge request

## Environment Variables

This tool relies on GitLab CI environment variables:

### Required Variables
- `GITLAB_TOKEN`: Personal Access Token with API access
- `CI_PROJECT_ID`: GitLab project ID
- `CI_COMMIT_REF_NAME`: Current branch name
- `CI_COMMIT_TITLE`: Current commit title

### Required for Pipeline Check
- `CI_COMMIT_SHA`: Current commit SHA (required when using --wait-pipeline)

### Optional Variables
- `CI_SERVER_URL`: GitLab server URL (defaults to https://gitlab.com)
- `CI_PROJECT_PATH`: GitLab project path

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
