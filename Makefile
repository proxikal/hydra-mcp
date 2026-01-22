.PHONY: test test-race lint build mocks

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	golangci-lint run

build:
	go build -o bin/hydra cmd/hydra/main.go

mocks:
	mockery --all --dir=internal --output=internal/mocks
