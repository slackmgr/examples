include .env # This is where the Slack app and bot tokens go, as well as the Slack channel ID where all alerts will be sent

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

run: compile
	SLACK_APP_TOKEN=$(SLACK_APP_TOKEN) \
	SLACK_BOT_TOKEN=$(SLACK_BOT_TOKEN) \
	ALERT_CHANNEL_ID=$(ALERT_CHANNEL_ID) \
	./bin/$(APP)
