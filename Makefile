.PHONY: all build test test-cover test-integration lint fmt tidy deps check install uninstall release snapshot smoke clean

BINARY := slaw
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w \
	-X github.com/open-cli-collective/slack-app-web/internal/version.Version=$(VERSION) \
	-X github.com/open-cli-collective/slack-app-web/internal/version.Commit=$(COMMIT) \
	-X github.com/open-cli-collective/slack-app-web/internal/version.Date=$(DATE)"

all: build

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/slaw

test:
	go test -race ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Runs the browser flows against a signed-in profile: SLAW_PROFILE_DIR=<dir>.
test-integration:
	go test -tags integration -count=1 ./internal/slackweb/...

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

deps:
	go mod download
	go mod verify

check: tidy fmt lint test build

smoke: build
	./bin/$(BINARY) version
	./bin/$(BINARY) --help > /dev/null
	for grp in app-tokens apps bot-tokens browser config-tokens icons login me; do \
		./bin/$(BINARY) $$grp --help > /dev/null 2>&1 || exit 1; \
	done

install: build
	mkdir -p $(HOME)/.local/bin
	cp bin/$(BINARY) $(HOME)/.local/bin/

uninstall:
	rm -f $(HOME)/.local/bin/$(BINARY)

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin/ dist/
	rm -f coverage.out coverage.html
