.PHONY: build clean test run

BINARY_NAME=dloom
BUILD_DIR=bin

build:
	@echo "Building $(BINARY_NAME) in $(BUILD_DIR)..."
	@mkdir -p $(BUILD_DIR)
	@echo "Created build dir $(BUILD_DIR)..."
	@echo "Invoking go build -o $(BUILD_DIR)/$(BINARY_NAME)..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Completed go build to $(BUILD_DIR)/$(BINARY_NAME)..."
	@echo "Directory listing for $(BUILD_DIR)..."
	@ls -la $(BUILD_DIR)
	@echo "Checking if $(BUILD_DIR)/$(BINARY_NAME) exists..."
	@ls -la $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Binary $(BUILD_DIR)/$(BINARY_NAME) exists..."
	@echo "Making $(BUILD_DIR)/$(BINARY_NAME) executable..."
	@chmod +x $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Marked $(BUILD_DIR)/$(BINARY_NAME) executable..."
	@echo "Completed build of $(BINARY_NAME) in $(BUILD_DIR)..."

clean:
	@echo "Cleaning $(BUILD_DIR)..."
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned $(BUILD_DIR)..."

test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "Completed running tests..."

run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)
