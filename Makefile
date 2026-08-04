.PHONY: build test vet run sqlc migrate-up

build:
	go build ./...

test:
	go tool gotestsum --format standard-verbose -- -v -race -cover ./...

vet:
	go vet ./... && golangci-lint run

run:
	go run .

sqlc:
	sqlc generate

migrate-up:
	./goose.sh up
