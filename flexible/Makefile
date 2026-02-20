include .env

APP=slack-manager

init: modules

modules:
	go mod tidy

test:
	gosec ./...
	go fmt ./...
	go test ./... -timeout 5s --cover
	go vet ./...

compile:
	go build -ldflags "-X main.AppName=$(APP) -X main.Version=vDEV" -o bin/$(APP)

build: test compile

lint:
	golangci-lint run ./...

bump-common-lib:
	go get github.com/slackmgr/types@latest
	go mod tidy

run-with-in-memory-queues-and-postgres: compile
	LOG_JSON=false \
	VERBOSE=true \
	ENCRYPTION_KEY=$(ENCRYPTION_KEY) \
	SLACK_APP_TOKEN=$(SLACK_APP_TOKEN) \
	SLACK_BOT_TOKEN=$(SLACK_BOT_TOKEN) \
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
	VERBOSE=true \
	ENCRYPTION_KEY=$(ENCRYPTION_KEY) \
	SLACK_APP_TOKEN=$(SLACK_APP_TOKEN) \
	SLACK_BOT_TOKEN=$(SLACK_BOT_TOKEN) \
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
	REDIS_ADDR=localhost:6379 \
	QUEUE_MODE=sqs \
	DATABASE_MODE=dynamodb \
	AWS_REGION=eu-north-1 \
	AWS_DYNAMODB_TABLE_NAME=slack-manager \
	AWS_SQS_ALERT_QUEUE_NAME=slack-manager-alerts.fifo \
	AWS_SQS_COMMAND_QUEUE_NAME=slack-manager-commands.fifo \
	./bin/$(APP)
