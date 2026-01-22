.PHONY: test test-race lint build mocks

test:
	CGO_ENABLED=0 go test ./...

test-race:
	CGO_ENABLED=0 go test -race ./...

lint:
	golangci-lint run

build:
	go build -o bin/hydra cmd/hydra/main.go

mocks:
	mockery --all --dir=internal --output=internal/mocks
