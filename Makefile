.PHONY: build test lint fmt vet install clean coverage

# Build the binary
build:
	go build -o alza .

# Run tests
test:
	go test -v ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	gofmt -w .

# Check formatting (for CI)
fmt-check:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Code is not formatted. Run 'make fmt' to fix."; \
		gofmt -d .; \
		exit 1; \
	fi

# Run go vet
vet:
	go vet ./...

# Run all linting checks
lint: fmt-check vet

# Install binary to GOPATH/bin
install:
	go install .

# Clean build artifacts
clean:
	rm -f alza coverage.out coverage.html

# Run all checks (for CI)
ci: lint test build

# Help
help:
	@echo "Available targets:"
	@echo "  build     - Build the binary"
	@echo "  test      - Run tests"
	@echo "  coverage  - Run tests with coverage report"
	@echo "  fmt       - Format code with gofmt"
	@echo "  fmt-check - Check if code is formatted"
	@echo "  vet       - Run go vet"
	@echo "  lint      - Run all linting checks"
	@echo "  install   - Install to GOPATH/bin"
	@echo "  clean     - Remove build artifacts"
	@echo "  ci        - Run all CI checks"
