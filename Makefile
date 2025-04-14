# Project name
PROJECT_NAME := nca

# Go command
GO := go

# Output directory
OUTPUT_DIR := bin

# Main source file
MAIN_FILE := cmd/nca/main.go

# Target platforms
PLATFORMS := linux darwin windows

# Target architectures
ARCHITECTURES := amd64 arm64

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d %H:%M:%S')
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Compilation flags with version info - using single quotes for shell compatibility
LDFLAGS = -ldflags '-X "main.Version=$(VERSION)" -X "main.BuildTime=$(BUILD_TIME)" -X "main.CommitHash=$(COMMIT_HASH)"'

# Default target
.PHONY: all
all: build

# Initialize module
.PHONY: init
init:
	$(GO) mod init github.com/pederhe/nca
	$(GO) mod tidy

# Build binary for current platform
.PHONY: build
build:
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build $(LDFLAGS) -o $(OUTPUT_DIR)/$(PROJECT_NAME) $(MAIN_FILE)
	@echo "Built $(PROJECT_NAME) to $(OUTPUT_DIR)/$(PROJECT_NAME)"

# Build toolstest utility
.PHONY: toolstest
toolstest:
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build -o $(OUTPUT_DIR)/toolstest cmd/toolstest/main.go
	@echo "Built toolstest to $(OUTPUT_DIR)/toolstest"

# Build binaries for all platforms
.PHONY: build-all
build-all:
	@mkdir -p $(OUTPUT_DIR)
	@for platform in $(PLATFORMS); do \
		for arch in $(ARCHITECTURES); do \
			output_name=$(PROJECT_NAME); \
			if [ $$platform = "windows" ]; then \
				output_name=$(PROJECT_NAME).exe; \
			fi; \
			echo "Building $$platform/$$arch..."; \
			GOOS=$$platform GOARCH=$$arch $(GO) build $(LDFLAGS) -o $(OUTPUT_DIR)/$(PROJECT_NAME)_$${platform}_$${arch}/$$output_name $(MAIN_FILE); \
		done; \
	done
	@echo "All platform builds completed"

# Run tests
.PHONY: test
test:
	$(GO) test -v ./...

# Run code checks
.PHONY: lint
lint:
	$(GO) vet ./...
	@if command -v staticcheck > /dev/null; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed. Run: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi

# Clean build artifacts
.PHONY: clean
clean:
	@rm -rf $(OUTPUT_DIR)
	@echo "Build artifacts cleaned"

# Install to GOPATH
.PHONY: install
install:
	$(GO) install $(LDFLAGS) ./...
	@echo "$(PROJECT_NAME) installed"

# Run program
.PHONY: run
run:
	$(GO) run $(MAIN_FILE)

# Help information
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make init       - Initialize Go module"
	@echo "  make build      - Build binary for current platform"
	@echo "  make build-all  - Build binaries for all supported platforms"
	@echo "  make toolstest  - Build the tools testing utility"
	@echo "  make test       - Run tests"
	@echo "  make lint       - Run code checks"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make install    - Install to GOPATH"
	@echo "  make run        - Run program"
	@echo "  make help       - Show this help information" 