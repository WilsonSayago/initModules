.PHONY: fmt fmt-check test test-race vet lint lint-verify vuln example ci

# Language minimum Go 1.24.0; recommended toolchain go1.27.1.
# CI matrix: 1.24.13 and 1.27.1. Lint and vuln run on current stable.
GOLANGCI_LINT_VERSION := v2.13.2
GOVULNCHECK_VERSION := v1.8.0

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

lint-verify:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) config verify

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

vet:
	go vet ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

example:
	cd examples/standalone && GOWORK=off go test ./...

ci: fmt-check vet test test-race example lint-verify lint vuln
