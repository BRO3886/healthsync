.PHONY: build test coverage clean run-parse run-server

BINARY := healthsync

build:
	go build -o $(BINARY) .

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
