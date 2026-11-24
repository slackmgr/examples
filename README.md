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

The `flexible` example adds production-grade infrastructure, including a Redis-backed per-channel alert rate limiter shared by all API instances. Start the dependencies first:

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
| `API_AUTH_MODE` | `off` | REST API authentication: `off`, `apikeys`, `oidc` or `both` (see [Authentication](#authentication)) |
| `API_AUTH_PERMISSIVE` | `false` | Let requests without credentials through anonymously (while migrating clients) |
| `API_CLIENT_BINDING` | `fill` | Set the alert `client` to the authenticated client ID: `fill`, `override` or `off` |
| `OIDC_ISSUER_URL` | — | Issuer of OIDC/OAuth2 access tokens (modes `oidc` and `both`) |
| `OIDC_AUDIENCE` | `slackmgr-api` | Required audience of access tokens |

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

## Authentication

The REST API of the flexible example is open by default (`API_AUTH_MODE=off`). Authentication is wired in [`flexible/auth.go`](flexible/auth.go). When it is enabled, all endpoints except `GET /ping` and `GET /openapi.yaml` require credentials with a scope: `alerts:write` (send alerts), `alerts:test` (`POST /alerts-test`), `config:read` (`GET /mappings`, `GET /channels`) or `*` (all).

### API keys

```bash
cd flexible
make apikey                 # prints a random key and its SHA-256 hash
```

Add the hash to `api-settings.yaml` (see the commented `apiKeys` section in `_sample_api-settings.yaml`), and start with `API_AUTH_MODE=apikeys`. Clients send the key as a bearer token (or as the password of HTTP basic authentication):

```bash
# No credentials: 401
curl -i -X POST http://localhost:8080/alerts -H 'Content-Type: application/json' -d @test-alerts/alert1.json

# With an API key: 202, or 403 if the key lacks the alerts:write scope or may not use the alert channel
curl -i -X POST http://localhost:8080/alerts -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $API_KEY" -d @test-alerts/alert1.json
```

The settings file is hot-reloaded, so keys can be added, rotated and revoked without a restart.

### OIDC/OAuth2 access tokens

The example includes a local [Keycloak](https://www.keycloak.org/) with a `slackmgr` realm ([`flexible/keycloak/slackmgr-realm.json`](flexible/keycloak/slackmgr-realm.json)) and two client credentials clients: `alertmanager` (scope `alerts:write`, secret `alertmanager-demo-secret`) and `slackmgr-admin` (all scopes, secret `admin-demo-secret`). The secrets and the Keycloak admin login (`admin`/`admin`, at http://localhost:8081) are public: use it for local development only.

```bash
cd flexible
docker compose --profile oidc up -d                  # Redis, Postgres and Keycloak
make run-with-redis-queues-and-postgres API_AUTH_MODE=oidc

# In another terminal: get an access token and send an alert
TOKEN=$(make -s token)                               # or: make -s token CLIENT=slackmgr-admin SECRET=admin-demo-secret
curl -i -X POST http://localhost:8080/alerts -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" -d @test-alerts/alert1.json
```

Tokens are validated by the [`oidc` plugin](https://github.com/slackmgr/plugins/tree/main/oidc): signature, issuer (`OIDC_ISSUER_URL`), audience (`OIDC_AUDIENCE`), expiry and scopes. The client ID (`azp`) with the prefix `oidc:` becomes the principal ID (e.g. `oidc:alertmanager`), so it never collides with an API key ID. With `API_AUTH_MODE=both`, API keys and access tokens are both accepted.

### Channel access rules and client binding

`accessRules` in `api-settings.yaml` restrict the Slack channels each authenticated client (API key ID, or `oidc:` + OIDC client ID; `oidc:*` matches all OIDC clients) may send alerts to, including escalation `moveToChannel` targets. Clients without a matching rule may use all channels, unless `defaultChannelAccess: deny`. With `API_CLIENT_BINDING=fill` (default), alerts without a `client` get the client ID, so insights show who sent them; `override` always replaces it.

### Rolling out authentication

To enable authentication without breaking existing clients, start with `API_AUTH_PERMISSIVE=true`: requests with credentials are checked, requests without are let through. Move the clients to credentials one by one, and set `API_AUTH_PERMISSIVE=false` when the `slackmgr_http_auth_requests_total{result="anonymous"}` metric (and the "Anonymous Requests" panel of the core Grafana dashboard) stays at zero.

See the [core README](https://github.com/slackmgr/core#authentication) for all details.

## Related

- [slackmgr/core](https://github.com/slackmgr/core) — the core library embedded by these examples
- [slackmgr/go-client](https://github.com/slackmgr/go-client) — Go client for sending alerts to a running instance
- [slackmgr/plugins](https://github.com/slackmgr/plugins) — database and queue plugins (SQS, DynamoDB, Postgres, Pub/Sub) and the OIDC authenticator
- [slackmgr/types](https://github.com/slackmgr/types) — shared interfaces and domain types

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

Copyright (c) 2026 Peter Aglen
