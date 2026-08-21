# GitLab Auto MR - Go Version

Automatically create Merge Requests in GitLab using a lightweight Go binary.

## Features

- ✅ Zero external dependencies - uses only Go standard library
- ✅ Small Docker image (Alpine-based)
- ✅ Fast execution
- ✅ Compatible with GitLab CI/CD
- ✅ Cross-platform support (Linux, macOS, Windows)
- ✅ Safe MR management - explicit control over creating or updating MRs
- ✅ Prevents accidental overwrites - requires `--update-mr` flag to update existing MRs

## Docker Images

Docker images are automatically built and published to GitHub Container Registry for each release.

### Available Tags

- `latest` - Latest stable release
- `vX.Y.Z` - Specific version
- `vX.Y` - Latest patch version of a minor release
- `vX` - Latest minor and patch version of a major release (e.g., `v1`)

### Pull the Image

```bash
# Latest release
docker pull ghcr.io/batonogov/gitlab-auto-mr:latest

# Specific version
docker pull ghcr.io/batonogov/gitlab-auto-mr:vX.Y.Z
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
# Creates new MR or informs if MR already exists
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
# Creates new MR or informs if MR already exists
./gitlab_auto_mr --target-branch main --commit-prefix "Ready"
```

## Usage

### GitLab CI/CD

#### Create MR (Default Behavior)

```yaml
create_mr:
  stage: create-mr
  image: ghcr.io/batonogov/gitlab-auto-mr:latest
  script:
    - |
      # Creates new MR or informs if MR already exists
      gitlab_auto_mr \
        --target-branch main \
        --commit-prefix "Ready" \
        --remove-branch \
        --squash-commits
  rules:
    - if: $CI_COMMIT_BRANCH != "main" && $CI_PIPELINE_SOURCE != "merge_request_event"
```

#### Update Existing MR

```yaml
update_mr:
  stage: update-mr
  image: ghcr.io/batonogov/gitlab-auto-mr:latest
  script:
    - |
      # Updates existing MR (requires --update-mr flag)
      gitlab_auto_mr \
        --update-mr \
        --target-branch main \
        --commit-prefix "Ready" \
        --description .gitlab/merge_request_template.md \
        --use-issue-name \
        --reviewer-id "${REVIEWER_IDS}"
  rules:
    - if: $CI_COMMIT_BRANCH != "main" && $CI_PIPELINE_SOURCE != "merge_request_event"
```

#### Advanced Control

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

### Optional Environment Variables

- `GITLAB_AUTO_MR_TARGET_BRANCH` - Target branch for the MR. Overridden by
  `--target-branch`/`-t`; when neither is set, the project's default branch is used.

### CLI Options

| Option                  | Short | Description                                    | Default                |
| ----------------------- | ----- | ---------------------------------------------- | ---------------------- |
| `--target-branch`       | `-t`  | Target branch for MR (`GITLAB_AUTO_MR_TARGET_BRANCH`) | Project default branch |
| `--commit-prefix`       | `-c`  | MR title prefix                                | `Draft`                |
| `--title`               |       | Custom MR title                                | Source branch name     |
| `--description`         | `-d`  | Path to description file                       | -                      |
| `--remove-branch`       | `-r`  | Delete source branch after merge               | `false`                |
| `--squash-commits`      | `-s`  | Squash commits on merge                        | `false`                |
| `--use-issue-name`      | `-i`  | Use issue data from branch name                | `false`                |
| `--allow-collaboration` | `-a`  | Allow commits from merge target members        | `false`                |
| `--reviewer-id`         |       | Reviewer user ID(s) (comma-separated)          | -                      |
| `--mr-exists`           |       | Only check if MR exists (dry run)              | `false`                |
| `--update-mr`           |       | Update existing MR (required to update, fail if none exists) | `false`                |
| `--create-only`         |       | Force create new MR (fail if already exists)   | `false`                |
| `--auto-merge`          |       | Enable merge when pipeline succeeds (auto-merge) | `false`              |
| `--trigger-pipeline`    |       | Create a merge request pipeline after the MR is created | `false`         |
| `--insecure`            | `-k`  | Skip SSL certificate verification              | `false`                |

### Merge Request Pipelines

GitLab does not start a merge request pipeline on its own when the MR is created
through the API for a commit that already has a branch pipeline. If your CI
configuration runs jobs only on `merge_request_event`, those jobs never run for
the newly created MR — and the branch pipeline that did run may look green while
having executed none of them.

`--trigger-pipeline` asks GitLab for that pipeline explicitly, right after the MR
is created:

```yaml
open_merge_request:
  image: $GITLAB_AUTO_MR_IMAGE
  script:
    - gitlab_auto_mr --target-branch main --trigger-pipeline
```

Two things to expect:

- On the push that creates the MR you will have **two pipelines for the same
  commit** — the branch pipeline that ran this job, and the merge request
  pipeline created here. A `workflow` rule such as
  `$CI_COMMIT_BRANCH && $CI_OPEN_MERGE_REQUESTS == null` keeps later pushes to
  the same branch down to one.
- The token needs the `api` scope, and its owner needs at least the Developer
  role on the project. A `CI_JOB_TOKEN` cannot be used: the Merge Requests API
  is read-only for job tokens, so it can neither create the MR nor request a
  pipeline for it.

## Operating

The tool acts on behalf of whoever owns `GITLAB_PRIVATE_TOKEN`. That dependency is
easy to lose track of: merge requests start appearing "by themselves", and months
later nobody remembers that behind them stands a live account holding an
`api`-scoped token.

- **Use a service account, not a person.** A personal token disappears when that
  person leaves the company or rotates their credentials, and every MR the tool
  opens is attributed to them.
- **Give the account Developer on the project — no more.** That is enough to read
  the project and to create, update and merge merge requests.
- **`CI_JOB_TOKEN` cannot be substituted.** The Merge Requests API is read-only for
  job tokens, so a personal or project access token with the `api` scope is
  required.
- **Plan for rotation.** GitLab no longer issues non-expiring personal access
  tokens, so expiry is scheduled work. When the token expires, MR creation simply
  stops — and "no new MRs" is a symptom that is easy to miss for weeks.
- **Name the account so its purpose is obvious** (for example
  `svc-gitlab-auto-mr`). Whoever finds it during a directory cleanup will decide
  its fate based on the name alone; an account called `test1` gets deleted.

## Building

### Prerequisites

- Go (see `go.mod` for minimum version)
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

### Create New MR (Default)

```bash
# Creates new MR or informs if MR already exists
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

### Update Existing MR

```bash
# Update existing MR (requires --update-mr flag, fail if MR doesn't exist)
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
- An expired or revoked token gives the same message — see [Operating](#operating)
  for who the token should belong to and why rotation is scheduled work

**MR Already Exists**

```
Merge request already exists: Feature XYZ (IID: 42). Use --update-mr flag to update it.
```

- This is the default behavior when MR already exists
- Add `--update-mr` flag to update the existing MR instead

**Update Mode Error**

```
Error: merge request does not exist for this branch feature/test to main, cannot update non-existent MR
```

- This happens when using `--update-mr` flag but no MR exists
- Remove `--update-mr` flag to create a new MR instead

**Force Create Mode Error**

```
Error: merge request already exists for this branch feature/test to main, cannot create new MR in create-only mode
```

- This happens when using `--create-only` flag but MR already exists
- Remove `--create-only` flag to allow the tool to inform about existing MR
- Use `--update-mr` flag instead if you want to update the existing MR

**Same Source/Target Branch**

```
Error: source branch and target branches must be different
```

- Ensure you're not running on the target branch

## License

MIT License - see LICENSE file for details.
