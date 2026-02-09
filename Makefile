BINARY_NAME=wabisaby-node
BUILD_DIR=bin
CMD_DIR=cmd/node

.PHONY: build clean test tidy

## build: Build the node binary
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## test: Run all tests
test:
	go test ./...

## tidy: Run go mod tidy
tidy:
	go mod tidy

## help: Show this help
help:
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'
