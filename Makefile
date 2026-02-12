.PHONY: build install test coverage clean tidy lint website website-dev

BINARY := healthsync

build:
	go build -o $(BINARY) .

install:
	go build -o $(shell go env GOPATH)/bin/$(BINARY) .

test:
	go test ./... -v -count=1

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

clean:
	rm -f $(BINARY) coverage.out coverage.html

tidy:
	go mod tidy

lint:
	go vet ./...

website:
	cd website && hugo --minify

website-dev:
	cd website && hugo server -D
