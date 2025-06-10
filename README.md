# GitLab Auto MR Docker Image

Automatically create Merge Requests in GitLab using Docker.

## Usage

```yaml
create_mr:
  stage: prepare
  image: ghcr.io/batonogov/gitlab-auto-mr:latest
  script:
    - |
      gitlab_auto_mr \
        --target-branch main \
        --commit-prefix Draft \
        --description ./.gitlab/merge_request/template.md \
        --remove-branch \
        --squash-commits \
        --use-issue-name
  rules:
    - if: $CI_COMMIT_BRANCH != "main" && $CI_PIPELINE_SOURCE != "merge_request_event"
```

## Required Environment Variables

Set these in your GitLab project's CI/CD Variables:

- `GITLAB_PRIVATE_TOKEN` - GitLab personal access token with `api` scope
- `GITLAB_USER_ID` - Your GitLab user ID

## All Available Options

### Required Options
| Option | Environment Variable | Description |
|--------|---------------------|-------------|
| `--private-token` | `GITLAB_PRIVATE_TOKEN` | GitLab private token for authentication |
| `--source-branch` | `CI_COMMIT_REF_NAME` | Source branch to merge from |
| `--project-id` | `CI_PROJECT_ID` | GitLab project ID |
| `--gitlab-url` | `CI_PROJECT_URL` | GitLab URL (e.g., https://gitlab.com) |
| `--user-id` | `GITLAB_USER_ID` | GitLab user ID to assign MR to (repeatable) |

### Optional Options
| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--target-branch` | `-t` | Target branch for MR | Project default branch |
| `--commit-prefix` | `-c` | MR title prefix | `Draft` |
| `--title` | | Custom MR title | Source branch name |
| `--description` | `-d` | Path to description file | - |
| `--remove-branch` | `-r` | Delete source branch after merge | `false` |
| `--squash-commits` | `-s` | Squash commits on merge | `false` |
| `--use-issue-name` | `-i` | Use issue data from branch name | `false` |
| `--allow-collaboration` | `-a` | Allow commits from merge target members | `false` |
| `--reviewer-id` | | Reviewer user ID (repeatable) | - |
| `--mr-exists` | | Only check if MR exists (dry run) | `false` |
| `--insecure` | `-k` | Skip SSL certificate verification | `false` |

## Examples

### Basic MR
```yaml
script:
  - gitlab_auto_mr --target-branch main
```

### With reviewers
```yaml
script:
  - gitlab_auto_mr --target-branch main --reviewer-id 12345 --reviewer-id 67890
```

### Check if MR exists
```yaml
script:
  - gitlab_auto_mr --mr-exists --target-branch main
```

## Setup

1. Create GitLab Personal Access Token with `api` scope
2. Add `GITLAB_PRIVATE_TOKEN` to project CI/CD variables
3. Add `GITLAB_USER_ID` to project CI/CD variables
4. Use the Docker image in your pipeline

## Troubleshooting

- **Authentication error**: Check your `GITLAB_PRIVATE_TOKEN`
- **MR already exists**: Normal behavior, script exits successfully
- **Same source/target branch**: Ensure you're not on the target branch