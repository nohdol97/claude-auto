.PHONY: build test run clean install lint

BINARY_NAME=claude-auto
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=cmd/claude-auto/main.go

build:
	go build -o $(BINARY_PATH) $(MAIN_PATH)

test:
	go test -v -cover ./...

run:
	go run $(MAIN_PATH)

clean:
	rm -rf bin/
	go clean

install:
	go install ./cmd/claude-auto

lint:
	golangci-lint run

deps:
	go mod download
	go mod tidy

dev: deps build
	@echo "Development environment ready"

.DEFAULT_GOAL := build