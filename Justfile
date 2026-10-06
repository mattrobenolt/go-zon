TEST_FLAGS := "--packages=./... --format testname"

default:
    @just --list

# Format Go code
fmt: fmt-go

# Format Go code with golangci-lint
fmt-go:
    golangci-lint fmt

# Run linter
lint:
    golangci-lint run --show-stats

# Run tests
test:
    gotestsum {{ TEST_FLAGS }}

# Run tests with race detector
test-race:
    gotestsum {{ TEST_FLAGS }} -- -race

# Run tests in watch mode
test-watch:
    gotestsum --watch {{ TEST_FLAGS }}

# Start documentation server
docs:
    go doc -http

# Verify golden output parses under the system zig
verify-zig:
    go test {{ TEST_FLAGS }} -run TestZigParse
