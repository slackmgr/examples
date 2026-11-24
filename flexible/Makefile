include .env

APP=slack-manager

# REST API authentication (see auth.go). Override in .env or on the command line, e.g. make run-with-redis-queues-and-postgres API_AUTH_MODE=apikeys
API_AUTH_MODE ?= off
API_AUTH_PERMISSIVE ?= false
API_CLIENT_BINDING ?= fill
OIDC_ISSUER_URL ?= http://localhost:8081/realms/slackmgr
OIDC_AUDIENCE ?= slackmgr-api

init: modules

modules:
	go mod tidy

test:
	gosec ./...
	go fmt ./...
	go test ./... -race -timeout 5s --cover
	go vet ./...

compile:
	go build -ldflags "-X main.AppName=$(APP) -X main.Version=vDEV" -o bin/$(APP)

build: test compile

lint:
	golangci-lint run ./...

# Generate a random API key and its SHA-256 hash. Give the key to the client, and put the hash in api-settings.yaml (apiKeys).
apikey:
	@key=$$(openssl rand -hex 32); \
	echo "API key (give to the client): $$key"; \
	echo "keySha256 (api-settings.yaml): $$(printf '%s' "$$key" | openssl dgst -sha256 -r | cut -d' ' -f1)"

# Get an access token for a demo client of the local Keycloak (docker compose --profile oidc up -d), e.g. make token CLIENT=slackmgr-admin SECRET=admin-demo-secret
CLIENT ?= alertmanager
SECRET ?= alertmanager-demo-secret
token:
	@curl -sf -d grant_type=client_credentials -u $(CLIENT):$(SECRET) $(OIDC_ISSUER_URL)/protocol/openid-connect/token | sed -E 's/.*"access_token":"([^"]+)".*/\1/'

bump-common-lib:
	go get github.com/slackmgr/types@latest
	go mod tidy

run-with-in-memory-queues-and-postgres: compile
	LOG_JSON=false \
	VERBOSE=true \
	ENCRYPTION_KEY=$(ENCRYPTION_KEY) \
	SLACK_APP_TOKEN=$(SLACK_APP_TOKEN) \
	SLACK_BOT_TOKEN=$(SLACK_BOT_TOKEN) \
	API_AUTH_MODE=$(API_AUTH_MODE) \
	API_AUTH_PERMISSIVE=$(API_AUTH_PERMISSIVE) \
	API_CLIENT_BINDING=$(API_CLIENT_BINDING) \
	OIDC_ISSUER_URL=$(OIDC_ISSUER_URL) \
	OIDC_AUDIENCE=$(OIDC_AUDIENCE) \
	API_ALERTS_PER_SECOND=3 \
	API_ALLOWED_BURST=20 \
	REDIS_ADDR=localhost:6379 \
	QUEUE_MODE=in-memory \
	DATABASE_MODE=postgres \
	POSTGRES_HOST=localhost \
	POSTGRES_PORT=5432 \
	POSTGRES_USER=postgres \
	POSTGRES_PASSWORD=qwerty \
	POSTGRES_DATABASE=slack_manager \
	./bin/$(APP)

run-with-redis-queues-and-postgres: compile
	LOG_JSON=false \
	VERBOSE=false \
	ENCRYPTION_KEY=$(ENCRYPTION_KEY) \
	SLACK_APP_TOKEN=$(SLACK_APP_TOKEN) \
	SLACK_BOT_TOKEN=$(SLACK_BOT_TOKEN) \
	API_AUTH_MODE=$(API_AUTH_MODE) \
	API_AUTH_PERMISSIVE=$(API_AUTH_PERMISSIVE) \
	API_CLIENT_BINDING=$(API_CLIENT_BINDING) \
	OIDC_ISSUER_URL=$(OIDC_ISSUER_URL) \
	OIDC_AUDIENCE=$(OIDC_AUDIENCE) \
	API_ALERTS_PER_SECOND=0.5 \
	API_ALLOWED_BURST=8 \
	REDIS_ADDR=localhost:6379 \
	QUEUE_MODE=redis \
	DATABASE_MODE=postgres \
	POSTGRES_HOST=localhost \
	POSTGRES_PORT=5432 \
	POSTGRES_USER=postgres \
	POSTGRES_PASSWORD=qwerty \
	POSTGRES_DATABASE=slack_manager \
	./bin/$(APP)

run-with-sqs-queues-and-dynamodb: compile
	LOG_JSON=false \
	VERBOSE=true \
	ENCRYPTION_KEY=$(ENCRYPTION_KEY) \
	SLACK_APP_TOKEN=$(SLACK_APP_TOKEN) \
	SLACK_BOT_TOKEN=$(SLACK_BOT_TOKEN) \
	API_AUTH_MODE=$(API_AUTH_MODE) \
	API_AUTH_PERMISSIVE=$(API_AUTH_PERMISSIVE) \
	API_CLIENT_BINDING=$(API_CLIENT_BINDING) \
	OIDC_ISSUER_URL=$(OIDC_ISSUER_URL) \
	OIDC_AUDIENCE=$(OIDC_AUDIENCE) \
	API_ALERTS_PER_SECOND=3 \
	API_ALLOWED_BURST=20 \
	REDIS_ADDR=localhost:6379 \
	QUEUE_MODE=sqs \
	DATABASE_MODE=dynamodb \
	AWS_REGION=eu-north-1 \
	AWS_DYNAMODB_TABLE_NAME=slack-manager \
	AWS_SQS_ALERT_QUEUE_NAME=slack-manager-alerts.fifo \
	AWS_SQS_COMMAND_QUEUE_NAME=slack-manager-commands.fifo \
	./bin/$(APP)
