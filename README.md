# GitLab Auto MR - Go Version

Automatically create Merge Requests in GitLab using a lightweight Go binary.

## Features

- ✅ Zero external dependencies - uses only Go standard library
- ✅ Small Docker image (~26MB with Alpine 3.22)
- ✅ Fast execution (~10ms startup time)
- ✅ Compatible with GitLab CI/CD
- ✅ Cross-platform support (Linux, macOS, Windows)
- ✅ Built with Go 1.24 for optimal performance
- ✅ Update existing MRs with new commits and metadata
- ✅ Smart MR detection and management

## Docker Images

Docker images are automatically built and published to GitHub Container Registry for each release.

### Available Tags

- `latest` - Latest stable release
- `vX.Y.Z` - Specific version (e.g., `v1.6.0`)
- `vX.Y` - Latest patch version of a minor release (e.g., `v1.6`)
- `vX` - Latest minor and patch version of a major release (e.g., `v1`)

### Pull the Image

```bash
# Latest release
docker pull ghcr.io/batonogov/gitlab-auto-mr:latest

# Specific version
docker pull ghcr.io/batonogov/gitlab-auto-mr:v1.6.0
```

### Supported Platforms

- `linux/amd64`
- `linux/arm64`

## Quick Start

1. **Set required environment variables:**

```bash
export GITLAB_PRIVATE_TOKEN="your-gitlab-token"
export GITLAB_USER_ID="your-user-id"
export CI_PROJECT_ID="12345"
export CI_PROJECT_URL="https://gitlab.com/user/project"
export CI_COMMIT_REF_NAME="feature/my-branch"
```

2. **Run with Docker (recommended):**

```bash
docker run --rm \
  -e GITLAB_PRIVATE_TOKEN \
  -e GITLAB_USER_ID \
  -e CI_PROJECT_ID \
  -e CI_PROJECT_URL \
  -e CI_COMMIT_REF_NAME \
  ghcr.io/batonogov/gitlab-auto-mr:latest \
  gitlab_auto_mr --target-branch main
```

3. **Or build and run locally:**

```bash
git clone https://github.com/user/gitlab-auto-mr.git
cd gitlab-auto-mr
task build
./gitlab_auto_mr --target-branch main
```

## Usage

### GitLab CI/CD

#### Basic MR Creation

```yaml
create_mr:
  stage: create-mr
  image: ghcr.io/batonogov/gitlab-auto-mr:latest
  script:
    - |
      gitlab_auto_mr \
        --target-branch main \
        --commit-prefix "Draft" \
        --remove-branch \
        --squash-commits
  rules:
    - if: $CI_COMMIT_BRANCH != "main" && $CI_PIPELINE_SOURCE != "merge_request_event"
```

#### Update Existing MR on New Commits

```yaml
update_mr:
  stage: update-mr
  image: ghcr.io/batonogov/gitlab-auto-mr:latest
  script:
    - |
      # Check if MR exists first
      if gitlab_auto_mr --mr-exists --target-branch main; then
        # Update existing MR with new information
        gitlab_auto_mr \
          --update-mr \
          --target-branch main \
          --commit-prefix "Ready" \
          --description .gitlab/merge_request_template.md \
          --use-issue-name
      else
        # Create new MR if it doesn't exist
        gitlab_auto_mr \
          --target-branch main \
          --commit-prefix "Draft" \
          --description .gitlab/merge_request_template.md
      fi
  rules:
    - if: $CI_COMMIT_BRANCH != "main" && $CI_PIPELINE_SOURCE != "merge_request_event"
```

#### Smart MR Management

```yaml
manage_mr:
  stage: manage-mr
  image: ghcr.io/batonogov/gitlab-auto-mr:latest
  variables:
    MR_DESCRIPTION_FILE: ".gitlab/merge_request_description.md"
  script:
    - |
      # Try to update existing MR first, fallback to creation
      gitlab_auto_mr \
        --update-mr \
        --target-branch main \
        --title "feat: ${CI_COMMIT_SHORT_SHA} - Auto-generated MR" \
        --description "${MR_DESCRIPTION_FILE}" \
        --reviewer-id "${REVIEWER_IDS}" \
        --use-issue-name \
        --allow-collaboration || \
      gitlab_auto_mr \
        --target-branch main \
        --commit-prefix "Draft" \
        --description "${MR_DESCRIPTION_FILE}" \
        --reviewer-id "${REVIEWER_IDS}"
  rules:
    - if: $CI_COMMIT_BRANCH != "main" && $CI_PIPELINE_SOURCE != "merge_request_event"
```

### Required Environment Variables

- `GITLAB_PRIVATE_TOKEN` - GitLab personal access token with `api` scope
- `GITLAB_USER_ID` - Your GitLab user ID (comma-separated for multiple assignees)
- `CI_PROJECT_ID` - GitLab project ID
- `CI_PROJECT_URL` - GitLab URL
- `CI_COMMIT_REF_NAME` - Source branch name

### CLI Options

| Option                  | Short | Description                                    | Default                |
| ----------------------- | ----- | ---------------------------------------------- | ---------------------- |
| `--target-branch`       | `-t`  | Target branch for MR                           | Project default branch |
| `--commit-prefix`       | `-c`  | MR title prefix                                | `Draft`                |
| `--title`               |       | Custom MR title                                | Source branch name     |
| `--description`         | `-d`  | Path to description file                       | -                      |
| `--remove-branch`       | `-r`  | Delete source branch after merge               | `false`                |
| `--squash-commits`      | `-s`  | Squash commits on merge                        | `false`                |
| `--use-issue-name`      | `-i`  | Use issue data from branch name                | `false`                |
| `--allow-collaboration` | `-a`  | Allow commits from merge target members        | `false`                |
| `--reviewer-id`         |       | Reviewer user ID(s) (comma-separated)          | -                      |
| `--mr-exists`           |       | Only check if MR exists (dry run)              | `false`                |
| `--update-mr`           |       | Update existing MR instead of creating new one | `false`                |
| `--insecure`            | `-k`  | Skip SSL certificate verification              | `false`                |

## Building

### Prerequisites

- Go 1.24 or later
- Task (optional, for build automation)

### Local Development

```bash
# Using Task (recommended)
task build      # Build binary
task test       # Run tests
task docker-build  # Build Docker image
task clean      # Clean artifacts
task --list     # Show all tasks

# Or using Go directly
go build -o gitlab_auto_mr .
go test ./...
```

### Pre-commit Hooks

This project uses pre-commit hooks for code quality and consistency:

```bash
# Install pre-commit (if not already installed)
pip install pre-commit

# Install hooks
pre-commit install

# Run hooks on all files
pre-commit run --all-files

# Run hooks on specific files
pre-commit run --files main.go
```

**Included hooks:**

- Go dependency verification
- Go code formatting
- Go linting
- Go tests
- Taskfile validation
- Dockerfile validation
- YAML validation
- Markdown formatting
- Large file detection
- Merge conflict detection

### Docker Build

```bash
task docker-build
# or
docker build -t gitlab-auto-mr .
```

## Examples

### Basic MR Creation

```bash
./gitlab_auto_mr \
  --target-branch main \
  --commit-prefix "Draft"
```

### With Reviewers

```bash
./gitlab_auto_mr \
  --target-branch main \
  --reviewer-id "12345,67890"
```

### Check if MR Exists

```bash
./gitlab_auto_mr \
  --mr-exists \
  --target-branch main
```

### Update Existing MR

```bash
./gitlab_auto_mr \
  --update-mr \
  --target-branch main \
  --title "Updated Title" \
  --description new_description.md
```

### Force Update MR with New Reviewers

```bash
./gitlab_auto_mr \
  --update-mr \
  --target-branch main \
  --reviewer-id "12345,67890" \
  --use-issue-name
```

## Troubleshooting

**Authentication Error**

```
Error: unable to get project 12345: unauthorized access, check your access token is valid
```

- Check your `GITLAB_PRIVATE_TOKEN` is valid and has `api` scope

**MR Already Exists**

```
Merge request already exists for this branch feature/test to main
Use --update-mr flag to update the existing MR instead.
```

- This is normal behavior - use `--mr-exists` to check without creating
- Use `--update-mr` to update the existing MR with new parameters

**Updating Non-Existent MR**

```
Merge request does not exist for this branch feature/test to main, run without flag '--mr-exists' to open merge request.
```

- Remove `--update-mr` flag to create a new MR instead

**Same Source/Target Branch**

```
Error: source branch and target branches must be different
```

- Ensure you're not running on the target branch

## Performance vs Python Version

| Metric       | Go Version | Python Version |
| ------------ | ---------- | -------------- |
| Binary Size  | ~5.7MB     | ~50MB+         |
| Docker Image | ~26MB      | ~80MB+         |
| Startup Time | ~10ms      | ~200ms+        |
| Memory Usage | ~5MB       | ~20MB+         |
| Dependencies | 0 external | 10+ packages   |

## License

MIT License - see LICENSE file for details.
