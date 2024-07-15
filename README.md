# GitLab Auto MR

Create MR in GitLab automatically

Example:

```yaml
open_merge_request:
  stage: prepare
  image: ghcr.io/batonogov/gitlab-auto-mr:v1.2.0
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
