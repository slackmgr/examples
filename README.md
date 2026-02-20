# slackmgr/examples

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![CI (minimal)](https://github.com/slackmgr/examples/workflows/CI%20(minimal)/badge.svg)](https://github.com/slackmgr/examples/actions/workflows/ci-minimal.yml)
[![CI (flexible)](https://github.com/slackmgr/examples/workflows/CI%20(flexible)/badge.svg)](https://github.com/slackmgr/examples/actions/workflows/ci-flexible.yml)

Example host applications for [Slack Manager](https://github.com/slackmgr/core). Each example is a runnable program that wires up the core library with a specific set of infrastructure backends.

## Examples

| Example | Description |
|---------|-------------|
| [minimal](./minimal/) | In-memory queue, in-memory DB, no cache. For local testing and development only. |
| [flexible](./flexible/) | Redis queue or SQS, Postgres or DynamoDB, Redis cache, Prometheus metrics. Production-ready starting point. |

## Quick start

The `minimal` example is the fastest way to get something running. You only need a Slack app.

**1. Create a Slack app**

Use the `manifest.json` from the [core](https://github.com/slackmgr/core) repository at [api.slack.com/apps](https://api.slack.com/apps). Two tokens are required:

- **Bot token** (`xoxb-...`) — Slack API calls
- **App-level token** (`xapp-...`) — Socket Mode

**2. Configure**

```bash
cd minimal
cp _sample.env .env
# Edit .env: set SLACK_BOT_TOKEN, SLACK_APP_TOKEN, ALERT_CHANNEL_ID
```

**3. Run**

```bash
make run
```

**4. Send a test alert**

```bash
curl -X POST http://localhost:8080/alert \
  -H 'Content-Type: application/json' \
  -d @test-alerts/alert1.json
```

## The flexible example

The `flexible` example adds production-grade infrastructure. Start the dependencies first:

```bash
cd flexible
docker compose up -d          # starts Postgres + Redis
cp _sample_api-settings.yaml api-settings.yaml
cp _sample_manager-settings.yaml manager-settings.yaml
# Edit both yaml files and set environment variables (see config/config.go)
go run .
```

Key environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SLACK_BOT_TOKEN` | — | Slack bot token (`xoxb-...`) |
| `SLACK_APP_TOKEN` | — | Slack app-level token (`xapp-...`) |
| `QUEUE_MODE` | `redis` | `redis`, `sqs`, or `in-memory` |
| `DATABASE_MODE` | `postgres` | `postgres` or `dynamodb` |
| `REDIS_ADDR` | — | Redis address (e.g. `localhost:6379`) |
| `ENABLE_METRICS` | `true` | Expose Prometheus metrics on `METRICS_PORT` |
| `METRICS_PORT` | `9090` | Port for `/metrics` endpoint |
| `REST_PORT` | `8080` | Port for the alert ingestion REST API |
| `ENCRYPTION_KEY` | — | 32-char key for webhook payload encryption |

Settings files (`api-settings.yaml`, `manager-settings.yaml`) are hot-reloaded every 10 seconds — no restart needed when changing routing rules or admin lists.

## Alert routing

Alerts are routed to Slack channels via `routingRules` in `api-settings.yaml`. Rules match on the `routeKey` field of the alert using `equals`, `hasPrefix`, or `matchAll`. Always include a `matchAll` fallback rule.

```yaml
routingRules:
  - name: Team A
    hasPrefix: ["a-", "a/"]
    channel: CXXXXXXXXXXX
  - name: Fallback
    matchAll: true
    channel: CZZZZZZZZZZZ
```

## Related

- [slackmgr/core](https://github.com/slackmgr/core) — the core library embedded by these examples
- [slackmgr/go-client](https://github.com/slackmgr/go-client) — Go client for sending alerts to a running instance
- [slackmgr/plugins](https://github.com/slackmgr/plugins) — database and queue plugins (SQS, DynamoDB, Postgres, Pub/Sub)
- [slackmgr/types](https://github.com/slackmgr/types) — shared interfaces and domain types

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

Copyright (c) 2026 Peter Aglen
