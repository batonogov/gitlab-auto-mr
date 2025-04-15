# Использование Docker-образа gitlab-auto-mr

## Сборка образа

```bash
docker build -t gitlab-auto-mr:latest .
```

## Запуск в GitLab CI

Добавьте следующий код в ваш `.gitlab-ci.yml` файл:

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

## Локальное тестирование

```bash
docker run --rm \
  -e GITLAB_TOKEN=your_token \
  -e CI_SERVER_URL=https://gitlab.com \
  -e CI_PROJECT_ID=123456 \
  -e CI_COMMIT_REF_NAME=feature/123-something \
  -e CI_COMMIT_TITLE="Add new feature" \
  -v $(pwd):/workspace \
  -w /workspace \
  gitlab-auto-mr:latest \
  --target-branch main \
  --description ./description.md
```