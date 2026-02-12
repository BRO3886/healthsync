.PHONY: build install test coverage clean tidy lint website website-dev release

BINARY  := healthsync
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

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

release: ## Build cross-platform binaries for release
	@mkdir -p bin
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} ; \
		output=bin/$(BINARY)-$${GOOS}-$${GOARCH} ; \
		if [ "$${GOOS}" = "windows" ]; then output=$${output}.exe; fi ; \
		echo "Building $${output}..." ; \
		GOOS=$${GOOS} GOARCH=$${GOARCH} go build $(LDFLAGS) -o $${output} . || exit 1 ; \
	done

website: ## Build the Hugo website
	cd website && hugo --minify

website-dev: ## Start Hugo dev server
	cd website && hugo server -D

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
