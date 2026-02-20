# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This repository contains example applications demonstrating how to use the `slackmgr` (Slack Manager) library. There are two examples:

- **`minimal/`** — A bare-bones integration using in-memory queue, in-memory DB, and no cache. Intended for testing and development only.
- **`flexible/`** — A production-ready integration with pluggable backends (Redis/SQS queues, Postgres/DynamoDB database, Redis cache, Prometheus metrics, hot-reloadable settings).

Each example is a standalone Go module with its own `go.mod`.

## Commands

### minimal example

```sh
cd minimal
make modules        # go mod tidy
make compile        # build binary to bin/slack-manager
make build          # run tests + compile
make run            # compile and run (reads from .env)
make lint           # golangci-lint run ./...
go test ./... -timeout 5s --cover   # run tests directly
```

Requires a `.env` file (copy `_sample.env`):
```
SLACK_APP_TOKEN=xapp-...
SLACK_BOT_TOKEN=xoxb-...
ALERT_CHANNEL_ID=C...
```

### flexible example

```sh
cd flexible
go build -o bin/slack-manager .     # build
go test ./... -timeout 5s --cover   # run tests
golangci-lint run ./...             # lint
docker compose up -d                # start Postgres + Redis dependencies
```

Requires `manager-settings.yaml` and `api-settings.yaml` at runtime (copy from `_sample_*` files). All other config is via environment variables.

### Linting (both examples)

Linting uses golangci-lint v2 with config in `.golangci.yaml` at the repo root. Formatters enabled: `gci`, `gofmt`, `gofumpt`, `goimports`.

## Code Quality Requirements

**CRITICAL:** Before committing any changes, you MUST ensure both `make test` and `make lint` pass without errors. This applies to ALL changes, regardless of who made them (human, Claude, or other tools/linters).

```bash
# Always run before committing:
make test    # Must pass: gosec, go fmt, go test (with race detector), go vet
make lint    # Must pass: golangci-lint with zero issues
```

If either command fails:
1. Fix all reported issues
2. Re-run both commands to verify
3. Only commit after both pass

This ensures code quality, prevents broken releases, and maintains consistency across the codebase.

## Keeping Documentation in Sync

After every code change, check whether the affected example apps's `README.md` needs updating. The README is the public-facing documentation and must always reflect the actual code.

## Architecture

### Core concepts

Both examples use the same two-component architecture from `github.com/slackmgr/core`:

- **Manager** (`managerpkg.New(...)`) — processes alerts from a FIFO queue, manages Slack channels, handles deduplication/grouping/resolution. Runs as a long-lived goroutine.
- **API Server** (`api.New(...)`) — exposes a REST API where clients POST alerts. Enqueues alerts into the shared FIFO queue. Runs as a long-lived goroutine.

Both components share an alert queue, and the manager also has a command queue for inter-component communication.

### Pluggable interfaces (flexible example)

The flexible example demonstrates swapping backends via env vars:

| Concern | Env var | Options |
|---|---|---|
| Queue backend | `QUEUE_MODE` | `redis`, `sqs`, `in-memory` |
| Database | `DATABASE_MODE` | `postgres`, `dynamodb` |
| Cache | hardcoded Redis | — |
| Channel locking | hardcoded Redis | `NoopChannelLocker` for single-instance |
| Metrics | `ENABLE_METRICS` | Prometheus on `METRICS_PORT` (default 9090) |

Channel locking (`manager.ChannelLocker`) prevents concurrent processing of the same Slack channel across multiple instances. Required for distributed deployments (e.g., k8s); use `NoopChannelLocker` for single-instance.

### Settings vs Config

- **Config** (`flexible/config/config.go`) — startup config read from environment variables. Cannot change at runtime.
- **Settings** (`manager-settings.yaml`, `api-settings.yaml`) — runtime settings read from YAML files. Hot-reloaded every 10 seconds when the file hash changes. API settings define routing rules; manager settings define global admins.

### Logger interface

Both examples implement `types.Logger` using `zerolog`. The flexible example supports JSON output (`LOG_JSON=true`) and verbose/debug mode (`VERBOSE=true`).

### Alert routing

Alerts are routed to Slack channels via `routingRules` in `api-settings.yaml`. Rules match on `routeKey` using `equals`, `hasPrefix`, or `matchAll`. The last rule should always be a `matchAll` fallback.

### Sending test alerts

```sh
curl -X POST http://localhost:8080/alert -H 'Content-Type: application/json' -d @minimal/test-alerts/alert1.json
```

Key alert fields: `correlationId` (for deduplication), `routeKey` (for channel routing), `severity`, `autoResolveSeconds`.
