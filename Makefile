.PHONY: fmt test build tidy smoke

fmt:
	gofmt -w ./cmd ./internal

test:
	go test ./...

build:
	go build -o bin/mcp2cli ./cmd/mcp2cli

tidy:
	go mod tidy

smoke:
	bash scripts/smoke.sh
