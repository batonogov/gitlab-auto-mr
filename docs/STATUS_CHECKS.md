# Настройка статусных проверок для Pull Request

Данная документация описывает настройку статусных проверок для автоматической сборки образов при создании Pull Request.

## Обзор

В проекте настроено несколько пайплайнов для проверки качества кода при создании PR:

1. **PR Status Checks** - основные проверки статуса
1. **CI** - тестирование и сборка
1. **Docker Image CI** - сборка Docker образов
1. **Security Scan** - проверка безопасности

## Пайплайны

### 1. PR Status Checks (`.github/workflows/pr-status.yml`)

Быстрые проверки статуса для PR:

- **Статусные проверки**: форматирование, тесты, сборка
- **Docker Build Test**: проверка сборки Docker образа без публикации
- **PR Information**: отображение информации о PR

**Триггеры:**

- Создание PR
- Обновление PR
- Повторное открытие PR

### 2. CI Pipeline (`.github/workflows/ci.yml`)

Полный цикл тестирования и сборки:

- **Test**: тесты с покрытием кода
- **Lint**: проверка кода с golangci-lint
- **Build**: сборка для нескольких платформ
- **Security**: сканирование безопасности

### 3. Docker Image CI (`.github/workflows/docker-image.yml`)

Сборка и публикация Docker образов:

- **Для PR**: сборка статусных образов без публикации
- **Для main**: публикация образов в GHCR
- **Для тегов**: релизные образы

### 4. Badges (`.github/workflows/badges.yml`)

Обновление статусных бейджей:

- Покрытие кода
- Статус сборки
- Версия Go
- Последний релиз

## Настройка статусных образов

### Docker образы для PR

При создании PR автоматически создаются статусные образы с тегами:

```
ghcr.io/username/gitlab-auto-mr:pr-123
ghcr.io/username/gitlab-auto-mr:pr-feature-branch-sha
```

### Конфигурация

Основные настройки в `docker-image.yml`:

```yaml
on:
  pull_request:
    branches: [ main ]

tags: |
  type=ref,event=pr
  type=sha,prefix=pr-{{branch}}-
```

## Статусные проверки

### Обязательные проверки

Следующие проверки должны пройти успешно для мержа PR:

1. **PR Status Checks / status-checks**
1. **PR Status Checks / docker-build-test**
1. **CI / test**
1. **CI / lint**
1. **CI / build**

### Опциональные проверки

- **CI / security** - сканирование безопасности
- **Update Badges** - обновление бейджей

## Настройка репозитория

### Branch Protection Rules

Рекомендуемые настройки защиты ветки `main`:

```
Settings > Branches > Add rule

Branch name pattern: main

Protect matching branches:
☑ Require a pull request before merging
  ☑ Require approvals: 1
  ☑ Dismiss stale PR approvals when new commits are pushed
  ☑ Require review from code owners

☑ Require status checks to pass before merging
  ☑ Require branches to be up to date before merging

Required status checks:
- PR Status Checks / status-checks
- PR Status Checks / docker-build-test
- CI / test
- CI / lint
- CI / build

☑ Require conversation resolution before merging
☑ Include administrators
```

### Секреты для бейджей

Для работы динамических бейджей нужно настроить:

1. Создать публичный Gist для хранения бейджей
1. Добавить секреты в репозиторий:
   - `GIST_ID` - ID созданного Gist

### Настройка Dependabot

Автоматические обновления зависимостей настроены для:

- GitHub Actions (еженедельно)
- Docker образов (ежедневно)
- Go модулей (еженедельно)

## Локальная разработка

### Предварительные проверки

Перед созданием PR рекомендуется запустить локально:

```bash
# Форматирование
go fmt ./...

# Проверка линтером
golangci-lint run

# Тесты
go test -v -race ./...

# Сборка
go build -v ./...

# Docker сборка
docker build -t gitlab-auto-mr:test .
```

### Pre-commit hooks

Установка хуков для автоматических проверок:

```bash
pip install pre-commit
pre-commit install
pre-commit install --hook-type pre-push
```

## Мониторинг

### Статусные бейджи

В README отображаются актуальные статусы:

- **CI Status** - статус основного пайплайна
- **Docker Image** - статус сборки образов
- **Coverage** - покрытие кода тестами
- **Go Version** - используемая версия Go
- **Release** - последний релиз

### Notifications

GitHub автоматически отправляет уведомления о:

- Статусе проверок PR
- Неудачных сборках
- Завершенных проверках

## Troubleshooting

### Частые проблемы

1. **Не проходят линтеры**

   ```bash
   golangci-lint run --fix
   ```

1. **Падают тесты**

   ```bash
   go test -v -race ./...
   ```

1. **Не собирается Docker образ**

   ```bash
   docker build --no-cache -t test .
   ```

1. **Устаревшие зависимости**

   ```bash
   go mod tidy
   go mod verify
   ```

### Логи и отладка

- Логи пайплайнов доступны в разделе Actions
- Детальная информация о проверках в PR
- Артефакты сборки сохраняются на 30 дней
