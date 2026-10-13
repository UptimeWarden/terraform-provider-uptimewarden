default: build

build:
	go build -v ./...

test:
	go test -v -count=1 -timeout 120s ./...

# Acceptance tests — require a live UptimeWarden API token.
# Set UW_API_TOKEN and optionally UW_API_URL before running.
testacc:
	TF_ACC=1 go test -v -count=1 -timeout 300s ./internal/provider/...

lint:
	golangci-lint run ./...

install: build
	go install .

generate:
	go generate ./...

.PHONY: build test testacc lint install generate
