# GitLab Auto MR - Go Version

Automatically create Merge Requests in GitLab using a lightweight Go binary.

## Features

- ✅ Zero external dependencies - uses only Go standard library
- ✅ Small Docker image (~26MB with Alpine 3.22)
- ✅ Fast execution (~10ms startup time)
- ✅ Compatible with GitLab CI/CD
- ✅ Cross-platform support (Linux, macOS, Windows)
- ✅ Built with Go 1.25 for optimal performance
- ✅ Smart MR management - automatically updates existing MRs or creates new ones
- ✅ No need to remember flags - works intelligently by default

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
# Smart behavior: automatically updates existing MR or creates new one
docker run --rm \
  -e GITLAB_PRIVATE_TOKEN \
  -e GITLAB_USER_ID \
  -e CI_PROJECT_ID \
  -e CI_PROJECT_URL \
  -e CI_COMMIT_REF_NAME \
  ghcr.io/batonogov/gitlab-auto-mr:latest \
  gitlab_auto_mr --target-branch main --commit-prefix "Ready"
```

3. **Or build and run locally:**

```bash
git clone https://github.com/user/gitlab-auto-mr.git
cd gitlab-auto-mr
task build
# Smart behavior: automatically updates existing MR or creates new one
./gitlab_auto_mr --target-branch main --commit-prefix "Ready"
```

## Usage

### GitLab CI/CD

#### Smart MR Management (Default)

```yaml
smart_mr:
  stage: create-mr
  image: ghcr.io/batonogov/gitlab-auto-mr:latest
  script:
    - |
      # Automatically updates existing MR or creates new one
      gitlab_auto_mr \
        --target-branch main \
        --commit-prefix "Ready" \
        --remove-branch \
        --squash-commits
  rules:
    - if: $CI_COMMIT_BRANCH != "main" && $CI_PIPELINE_SOURCE != "merge_request_event"
```

#### Smart MR Management (Default Behavior)

```yaml
manage_mr:
  stage: manage-mr
  image: ghcr.io/batonogov/gitlab-auto-mr:latest
  script:
    - |
      # Smart behavior: automatically updates existing MR or creates new one
      gitlab_auto_mr \
        --target-branch main \
        --commit-prefix "Ready" \
        --description .gitlab/merge_request_template.md \
        --use-issue-name \
        --reviewer-id "${REVIEWER_IDS}"
  rules:
    - if: $CI_COMMIT_BRANCH != "main" && $CI_PIPELINE_SOURCE != "merge_request_event"
```

#### Advanced Control (Optional)

```yaml
# Force update only (fail if MR doesn't exist)
update_only:
  script:
    - gitlab_auto_mr --update-mr --target-branch main

# Force create only (fail if MR already exists)
create_only:
  script:
    - gitlab_auto_mr --create-only --target-branch main
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
| `--update-mr`           |       | Force update existing MR (fail if none exists) | `false`                |
| `--create-only`         |       | Force create new MR (fail if already exists)   | `false`                |
| `--insecure`            | `-k`  | Skip SSL certificate verification              | `false`                |

## Building

### Prerequisites

- Go 1.25 or later
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

### Smart MR Management (Default)

```bash
# Automatically updates existing MR or creates new one
./gitlab_auto_mr \
  --target-branch main \
  --commit-prefix "Ready"
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

### Force Update Only

```bash
# Only update existing MR (fail if MR doesn't exist)
./gitlab_auto_mr \
  --update-mr \
  --target-branch main \
  --title "Updated Title" \
  --description new_description.md
```

### Force Create Only

```bash
# Only create new MR (fail if MR already exists)
./gitlab_auto_mr \
  --create-only \
  --target-branch main \
  --reviewer-id "12345,67890"
```

## Troubleshooting

**Authentication Error**

```
Error: unable to get project 12345: unauthorized access, check your access token is valid
```

- Check your `GITLAB_PRIVATE_TOKEN` is valid and has `api` scope

**Force Update Mode Error**

```
Error: merge request does not exist for this branch feature/test to main, cannot update non-existent MR
```

- This happens when using `--update-mr` flag but no MR exists
- Remove `--update-mr` flag to use smart mode (create new MR)

**Force Create Mode Error**

```
Error: merge request already exists for this branch feature/test to main, cannot create new MR in create-only mode
```

- This happens when using `--create-only` flag but MR already exists
- Remove `--create-only` flag to use smart mode (update existing MR)

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
