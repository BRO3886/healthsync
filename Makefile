.PHONY: build install test coverage clean tidy lint website website-dev release

BINARY  := healthsync
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

build: ## Build the binary
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY) .

install: ## Install to $GOPATH/bin
	go install $(LDFLAGS) .

test: ## Run tests
	go test ./... -v -count=1

coverage: ## Run tests with coverage
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

coverage-html: coverage ## Open coverage report in browser
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

clean: ## Remove build artifacts
	rm -rf bin/ coverage.out coverage.html

tidy: ## Tidy go modules
	go mod tidy

lint: ## Run go vet
	go vet ./...

release: ## Build release tarballs for GitHub upload (darwin arm64 + amd64)
	@mkdir -p bin
	@for arch in arm64 amd64; do \
		echo "Building $(BINARY)-darwin-$$arch..."; \
		GOOS=darwin GOARCH=$$arch go build $(LDFLAGS) -o bin/$(BINARY) . || exit 1; \
		chmod +x bin/$(BINARY); \
		tar -czf bin/$(BINARY)-darwin-$$arch.tar.gz -C bin $(BINARY); \
		rm bin/$(BINARY); \
	done
	@echo "Upload bin/$(BINARY)-darwin-{arm64,amd64}.tar.gz to GitHub Releases"

website: ## Build the Hugo website
	cd website && hugo --minify

website-dev: ## Start Hugo dev server
	cd website && hugo server -D

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
