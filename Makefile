.PHONY: build test coverage clean tidy lint docs docs-dev

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

docs:
	cd docs && npm run build

docs-dev:
	cd docs && npm run dev
