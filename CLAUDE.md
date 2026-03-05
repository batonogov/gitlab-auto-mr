# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go CLI tool that automatically creates and manages GitLab Merge Requests. Zero external dependencies — uses only Go standard library.

## Build & Test Commands

```bash
task build          # Build binary with version info → ./gitlab_auto_mr
task test           # Run tests
task test-coverage  # Tests with coverage report → coverage.out
task fmt            # Format code (goimports + gofmt)
task lint           # Run golangci-lint
task ci             # Full pipeline: deps → fmt → lint → test → build

# Run a single test
go test -run TestFunctionName -v ./...

# Run tests with race detection
go test -race -v ./...
```

## Architecture

Single-file application (`main.go`) with all logic:

- **Config** struct holds all CLI flags parsed by `parseFlags()`
- **run()** is the main orchestrator: get project → validate → check existing MR → create/update MR → optionally enable auto-merge
- API functions (`getProject`, `getExistingMR`, `createMR`, `updateMR`, `acceptMR`) each handle one GitLab API endpoint
- All HTTP calls use `*http.Client` and `*Config` as parameters for testability

Tests (`main_test.go`) use `httptest.NewServer` to mock the GitLab API. Integration tests go through `run()` with a mock server.

## Code Style

- Max line length: 120 chars (enforced by `.golangci.yml`)
- Max cyclomatic complexity: 15
- Linters: golangci-lint with 16+ linters including `gosec`, `gocritic`, `dupl`
- Imports ordered: stdlib, then local (`github.com/batonogov/gitlab-auto-mr`)
- Conventional commits required (feat, fix, chore, etc.) — used by Release Please for versioning

## Key Patterns

- `createMR` returns `(*MergeRequest, error)` — the returned MR is needed for auto-merge IID
- Error messages from API functions include HTTP status and response body
- `parseFlags()` calls `os.Exit(1)` on validation errors — not testable via `run()`
- The `--update-mr` flag is required to update an existing MR; without it, the tool just informs and exits
