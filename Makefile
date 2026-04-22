BINARY    := sageo
CMD       := ./cmd/sageo
BUILD_DIR := ./build
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build install clean tidy run test test-integration test-all check-tests lint fmt vet check precommit help release

## help: Show available make targets
help:
	@echo "Usage: make <target>"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build binary to ./build/sageo
build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD)

## install: Install binary to GOPATH/bin
install:
	go install $(LDFLAGS) $(CMD)

## run: Run without installing (ARGS="..." to pass arguments)
run:
	go run $(LDFLAGS) $(CMD) $(ARGS)

## tidy: Tidy and verify go modules
tidy:
	go mod tidy
	go mod verify

## check-tests: Lint test files — no unit test may hit the live internet
check-tests:
	@chmod +x scripts/check-no-live-tests.sh
	@./scripts/check-no-live-tests.sh

## test: Run unit tests only (no network, no cost, safe by default)
test: check-tests
	go test -race -coverprofile=coverage.out ./...

## test-integration: Run integration tests that hit live paid APIs (opt-in)
test-integration:
	@echo "⚠️  This will hit paid APIs. Approx cost: \$$1-3. Ctrl-C to abort. Continuing in 5s..."
	@sleep 5
	SAGEO_LIVE_TESTS=1 go test -race -tags integration ./...

## test-all: Run unit tests, then integration tests
test-all: test test-integration

## coverage: Open test coverage report in browser
coverage: test
	go tool cover -html=coverage.out

## fmt: Format all Go source files
fmt:
	go fmt ./...

## vet: Run go vet
vet:
	go vet ./...

## lint: Run golangci-lint (installs if missing)
lint:
	@which golangci-lint > /dev/null 2>&1 || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@LINT_BIN=$$(which golangci-lint 2>/dev/null || echo "$$(go env GOPATH)/bin/golangci-lint"); \
	$$LINT_BIN run ./...

## check: Run fmt, vet, and lint (pre-commit gate)
check: fmt vet lint

## precommit: Run all pre-commit checks (fmt, vet, test, lint)
precommit: fmt vet test lint

## release: Build release binaries for all platforms
release:
	@chmod +x scripts/release.sh
	@./scripts/release.sh $(VERSION)

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR) dist coverage.out
