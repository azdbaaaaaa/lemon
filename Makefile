.PHONY: all build run dev test lint lint-fix fmt clean deps tools docker-build docker-run docker-stop wire coverage help init-admin

# 变量
APP_NAME := lemon
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X lemon/cmd.Version=$(VERSION) -X lemon/cmd.GitCommit=$(GIT_COMMIT) -X lemon/cmd.BuildTime=$(BUILD_TIME)"

# Go 相关
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod

# 默认目标
all: lint test build

# ==================== 依赖管理 ====================

# 安装依赖
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# 安装开发工具
tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/cosmtrek/air@latest
	@echo "Done!"

# ==================== 构建 ====================

# 构建
build:
	$(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME) .

# 构建 Linux 版本
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 .

# 构建所有平台
build-all: build build-linux
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME)-windows-amd64.exe .

# ==================== 运行 ====================

# 开发模式运行
dev:
	go run . serve --mode debug --log-level debug --log-format console

# 使用 air 热重载运行
dev-watch:
	air -c .air.toml

# 运行
run:
	go run . serve

# ==================== 测试 ====================

# 运行测试
test:
	$(GOTEST) -v -race ./...

# 运行测试 (简短输出)
test-short:
	$(GOTEST) -race ./...

# 测试覆盖率
coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# ==================== 代码质量 ====================

# 代码检查
lint:
	golangci-lint run ./...

# 代码检查并自动修复
lint-fix:
	golangci-lint run --fix ./...

# 格式化
fmt:
	$(GOCMD) fmt ./...
	goimports -w -local lemon .

# 检查格式化
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

# 安全检查
sec:
	gosec ./...

# ==================== 清理 ====================

# 清理构建产物
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(GOCMD) clean -cache -testcache

# ==================== Docker ====================

# Docker 构建
docker-build:
	docker build -t $(APP_NAME):$(VERSION) -f deployments/Dockerfile .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

# Docker 运行
docker-run:
	docker-compose -f deployments/docker-compose.yaml up -d

# Docker 停止
docker-stop:
	docker-compose -f deployments/docker-compose.yaml down

# Docker 日志
docker-logs:
	docker-compose -f deployments/docker-compose.yaml logs -f

# ==================== 其他 ====================

# 生成 wire 依赖注入代码
wire:
	wire ./internal/...

# 生成 Swagger 文档
swagger:
	@echo "Generating Swagger documentation..."
	swag init -g main.go -o docs/swagger
	@echo "Swagger documentation generated successfully!"

# 初始化 git 仓库
git-init:
	git init
	git add .
	git commit -m "Initial commit: Lemon AI API Service"

# 帮助
help:
	@echo "Lemon - AI-powered API Service"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Dependency Management:"
	@echo "  deps          Download and tidy dependencies"
	@echo "  tools         Install development tools (golangci-lint, goimports, air)"
	@echo ""
	@echo "Build:"
	@echo "  build         Build the application"
	@echo "  build-linux   Build for Linux"
	@echo "  build-all     Build for all platforms"
	@echo ""
	@echo "Run:"
	@echo "  dev           Run in development mode"
	@echo "  dev-watch     Run with hot reload (requires air)"
	@echo "  run           Run the application"
	@echo ""
	@echo "Test:"
	@echo "  test          Run tests with verbose output"
	@echo "  test-short    Run tests with short output"
	@echo "  coverage      Generate test coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint          Run golangci-lint"
	@echo "  lint-fix      Run golangci-lint with auto-fix"
	@echo "  fmt           Format code"
	@echo "  fmt-check     Check code formatting"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build  Build Docker image"
	@echo "  docker-run    Run with Docker Compose"
	@echo "  docker-stop   Stop Docker Compose"
	@echo "  docker-logs   View Docker Compose logs"
	@echo ""
	@echo "Other:"
	@echo "  clean         Clean build artifacts"
	@echo "  git-init      Initialize git repository"
	@echo "  help          Show this help"

## 初始化管理员账号（默认 admin / admin123）
init-admin:
	INIT_ADMIN_USERNAME=admin INIT_ADMIN_PASSWORD=admin123 go run ./scripts/init_admin.go
