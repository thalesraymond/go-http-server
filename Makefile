.PHONY: build test vet fmt fmt-check run sqlc migrate-up

build:
	go build ./...

test:
	go tool gotestsum --format standard-verbose -- -v -race -cover ./...

fmt:
	gofmt -w .

fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not gofmt formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

vet:
	go vet ./... && golangci-lint run

run:
	go run .

sqlc:
	sqlc generate

migrate-up:
	./goose.sh up
