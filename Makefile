# Makefile for kirk-ai project

# Variables
BINARY_NAME=kirk-ai
BUILD_DIR=./build
GO_FILES=$(shell find . -name "*.go" -not -path "./vendor/*")

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          -  Build the binary"
	@echo "  clean          -  Remove build artifacts"
	@echo "  tests          -  Run tests"
	@echo "  format         -  Format Go code"
	@echo "  vet            -  Vet Go code"
	@echo "  lint           -  Lint Go code (requires golint)"
	@echo "  deps           -  Tidy and download dependencies"
	@echo "  install        -  Install the binary to GOPATH/bin"
	@echo "  run            -  Run the application"
	@echo "  benchmarks     -  Run benchmark tests"
	@echo "  models         -  List available Ollama models"
	@echo "  check-ollama   -  Check if Ollama is running"
	@echo "  uncrawl        -  Delete crawling data"
	@echo "  help           -  Show this help"

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	cp $(BUILD_DIR)/$(BINARY_NAME) .

# Delete crawling data
.PHONY: uncrawl
uncrawl:
	@echo "Deleting crawling data"
	rm -rf tpusa_crawl

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)
	go clean

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	gofmt -w $(GO_FILES)

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	go vet ./...

# Lint code (install golint if needed: go install golang.org/x/lint/golint@latest)
.PHONY: lint
lint:
	@echo "Linting code..."
	golint ./...

# Manage dependencies
.PHONY: deps
deps:
	@echo "Tidying and downloading dependencies..."
	go mod tidy
	go mod download

# Install the binary
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	go install .

# Run the application
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	go run .

# Run benchmark
.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	./$(BUILD_DIR)/$(BINARY_NAME) benchmark --quick

# List models
.PHONY: models
models:
	@echo "Listing available models..."
	./$(BUILD_DIR)/$(BINARY_NAME) models

# Check if Ollama is running
.PHONY: check-ollama
check-ollama:
	@echo "Checking Ollama status..."
	@if curl -s http://localhost:11434/api/tags > /dev/null; then \
	    echo "Ollama is running."; \
	else \
	    echo "Ollama is not running. Please start it with 'ollama serve'."; \
	    exit 1; \
	fi