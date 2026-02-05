.PHONY: build test clean run docker

# 变量定义
BINARY_NAME=doris-sinker
VERSION=1.0.0
BUILD_TIME=$(shell date +%Y-%m-%d_%H:%M:%S)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# 构建
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/doris-sinker/main.go
	@echo "Build complete: bin/$(BINARY_NAME)"

# 运行
run: build
	@echo "Running $(BINARY_NAME)..."
	./bin/$(BINARY_NAME) -config configs/config.yaml

# 测试
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

# 基准测试
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# 代码检查
lint:
	@echo "Running linters..."
	golangci-lint run

# 清理
clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

# Docker构建
docker:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) -f deployments/docker/Dockerfile .

# 安装依赖
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# 格式化代码
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 生成文档
docs:
	@echo "Generating documentation..."
	godoc -http=:6060

# 帮助
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  run        - Build and run the application"
	@echo "  test       - Run tests"
	@echo "  benchmark  - Run benchmarks"
	@echo "  lint       - Run linters"
	@echo "  clean      - Clean build artifacts"
	@echo "  docker     - Build Docker image"
	@echo "  deps       - Install dependencies"
	@echo "  fmt        - Format code"
	@echo "  docs       - Generate documentation"
