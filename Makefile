.PHONY: fmt fmt-check test test-race vet lint example ci

FORMAT_FILES := main.go prop_test.go once_test.go initinstance_test.go app_test.go

fmt:
	gofmt -w $(FORMAT_FILES)

fmt-check:
	test -z "$$(gofmt -l *.go examples/standalone/*.go)"

test:
	go test -count=20 ./...

test-race:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	test -s coverage.out

lint:
	golangci-lint run ./...

vet:
	go vet ./...

example:
	cd examples/standalone && GOWORK=off go test ./...

ci: fmt-check vet test test-race example lint
