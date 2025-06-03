# GitLab Auto MR

[![CI](https://github.com/your-org/gitlab-auto-mr/workflows/CI/badge.svg)](https://github.com/your-org/gitlab-auto-mr/actions/workflows/ci.yml)
[![Docker Image](https://github.com/your-org/gitlab-auto-mr/workflows/Docker%20Image%20CI%20Linux%20ghcr.io/badge.svg)](https://github.com/your-org/gitlab-auto-mr/actions/workflows/docker-image.yml)
[![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/your-username/gist-id/raw/coverage.json)](https://github.com/your-org/gitlab-auto-mr/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/your-username/gist-id/raw/go-version.json)](https://golang.org/)
[![Release](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/your-username/gist-id/raw/release.json)](https://github.com/your-org/gitlab-auto-mr/releases)
[![License](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/your-username/gist-id/raw/license.json)](https://github.com/your-org/gitlab-auto-mr/blob/main/LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/ghcr.io/your-org/gitlab-auto-mr)](https://ghcr.io/your-org/gitlab-auto-mr)
[![Go Report Card](https://goreportcard.com/badge/github.com/your-org/gitlab-auto-mr)](https://goreportcard.com/report/github.com/your-org/gitlab-auto-mr)

A Go CLI tool for automatically creating merge requests in GitLab. Zero external dependencies, uses only Go standard library.

## Installation

### Download Binary

```bash
# Linux
curl -L -o gitlab_auto_mr https://github.com/your-org/gitlab-auto-mr/releases/latest/download/gitlab_auto_mr-linux-amd64
chmod +x gitlab_auto_mr && sudo mv gitlab_auto_mr /usr/local/bin/

# macOS
curl -L -o gitlab_auto_mr https://github.com/your-org/gitlab-auto-mr/releases/latest/download/gitlab_auto_mr-darwin-amd64
chmod +x gitlab_auto_mr && sudo mv gitlab_auto_mr /usr/local/bin/

# Windows
curl -L -o gitlab_auto_mr.exe https://github.com/your-org/gitlab-auto-mr/releases/latest/download/gitlab_auto_mr-windows-amd64.exe
```

### Build from Source

```bash
git clone https://github.com/your-org/gitlab-auto-mr.git
cd gitlab-auto-mr
go build -o gitlab_auto_mr .
```

### Docker

```bash
docker pull registry.gitlab.com/your-group/gitlab-auto-mr:latest
```

## Usage

### Basic Example

```bash
gitlab_auto_mr \
  --private-token "glpat-xxxxxxxxxxxxxxxxxxxx" \
  --source-branch "feature/my-feature" \
  --target-branch "main" \
  --project-id 12345 \
  --gitlab-url "https://gitlab.com"
```

### With Environment Variables

```bash
export GITLAB_PRIVATE_TOKEN="glpat-xxxxxxxxxxxxxxxxxxxx"
export CI_COMMIT_REF_NAME="feature/my-feature"
export CI_PROJECT_ID="12345"
export CI_PROJECT_URL="https://gitlab.com/group/project"

gitlab_auto_mr --target-branch main
```

### Advanced Usage

```bash
gitlab_auto_mr \
  --target-branch main \
  --title "Add new feature" \
  --description ./docs/mr_template.md \
  --user-id "123,456" \
  --reviewer-id "789" \
  --commit-prefix "Feature" \
  --remove-branch \
  --squash-commits
```

## GitLab CI/CD Integration

Add to your `.gitlab-ci.yml`:

```yaml
stages:
  - merge-request

create_mr:
  stage: merge-request
  image: registry.gitlab.com/your-group/gitlab-auto-mr:latest
  script:
    - gitlab_auto_mr --target-branch main --commit-prefix "Auto"
  rules:
    - if: $CI_PIPELINE_SOURCE == "push" && $CI_COMMIT_BRANCH != "main"
  variables:
    GITLAB_PRIVATE_TOKEN: $GITLAB_PRIVATE_TOKEN
```

## Command Line Options

| Option             | Environment Variable   | Description                           |
| ------------------ | ---------------------- | ------------------------------------- |
| `--private-token`  | `GITLAB_PRIVATE_TOKEN` | GitLab private token (required)       |
| `--source-branch`  | `CI_COMMIT_REF_NAME`   | Source branch name (required)         |
| `--target-branch`  |                        | Target branch name (default: "main")  |
| `--project-id`     | `CI_PROJECT_ID`        | GitLab project ID (required)          |
| `--gitlab-url`     | `CI_PROJECT_URL`       | GitLab URL (required)                 |
| `--user-id`        | `GITLAB_USER_ID`       | Assignee user IDs (comma-separated)   |
| `--reviewer-id`    |                        | Reviewer user IDs (comma-separated)   |
| `--title`          |                        | Custom MR title                       |
| `--description`    |                        | Path to description file              |
| `--commit-prefix`  |                        | Title prefix (default: "Draft")       |
| `--use-issue-name` |                        | Extract issue number from branch name |
| `--remove-branch`  |                        | Remove source branch after merge      |
| `--squash-commits` |                        | Squash commits on merge               |
| `--mr-exists`      |                        | Check if MR exists (don't create)     |
| `--version`        |                        | Show version information              |

## Development

Requires Go 1.21+ and [Task](https://taskfile.dev/).

```bash
task --list        # Show available tasks
task test          # Run tests
task build         # Build for current platform
task build-all     # Build for all platforms
task docker-build  # Build Docker image
```

### Pre-commit Hooks

The project uses [pre-commit](https://pre-commit.com/) to ensure code quality:

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install
pre-commit install --hook-type pre-push

# Run hooks manually
pre-commit run --all-files
```

Available hooks:

- **go-deps**: Verify Go dependencies are clean
- **go-fmt**: Format Go code with `gofmt`
- **go-lint**: Run Go linters (`go vet` + format check)
- **go-test**: Run all tests
- **go-build**: Verify build works (pre-push only)
- **taskfile-validation**: Validate Taskfile syntax
- **dockerfile-validation**: Basic Dockerfile syntax check

All hooks use Taskfile commands for consistency.

## Versioning

Version information is automatically generated from git tags and build metadata:

- **Development builds**: `v0.0.3-alpha-8-g672a18b-dirty` (based on git describe)
- **Tagged releases**: `v1.0.0` (clean tag name)
- **Unknown**: `dev` (fallback when git is not available)

### Creating Releases

```bash
# Check current version info
task version

# Create a new tag
task tag TAG=v1.0.0

# Push the tag
git push origin v1.0.0

# Build release artifacts (only works on tagged commits)
task release
```

### Version Components

The version information includes:

- **Version**: Git tag or describe output
- **Git Commit**: Full SHA of the build commit
- **Build Date**: UTC timestamp when binary was built

All version info is embedded at build time via Go's `-ldflags`.

## License

Apache License 2.0
