.PHONY: test lint vet ci example

test:
	go test -race -cover ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

ci: vet test lint

example:
	cd examples/standalone && go run .
