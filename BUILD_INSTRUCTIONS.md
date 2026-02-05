# 构建说明

## 修复编译错误后的构建步骤

### 1. 初始化依赖

由于我们修复了一些API兼容性问题，需要重新下载依赖：

```bash
cd doris-sinker

# 清理旧的依赖
go clean -modcache

# 下载依赖
go mod download

# 整理依赖
go mod tidy
```

### 2. 已修复的问题

#### 问题1: zapcore.Second undefined
**原因**: zapcore 没有 Second 常量
**修复**: 使用 `time.Second` 替代

#### 问题2: sonic.Node undefined  
**原因**: sonic 的 API 变更，Node 类型不再直接暴露
**修复**: 使用 `sonic.Get()` 返回 `[]byte`，然后用 `ast.Node` 解析

#### 问题3: kgo.FetchMaxRecords undefined
**原因**: franz-go 的新版本移除了 FetchMaxRecords 选项
**修复**: 移除该选项，使用 FetchMaxBytes 和 FetchMinBytes 控制

### 3. 构建

```bash
# 构建二进制
make build

# 或直接使用 go build
go build -o bin/doris-sinker cmd/doris-sinker/main.go
```

### 4. 验证

```bash
# 查看版本
./bin/doris-sinker -h

# 测试运行（需要先配置 Kafka 和 Doris）
./bin/doris-sinker -config configs/config.dev.yaml
```

## 依赖版本说明

当前使用的主要依赖版本：

- `github.com/bytedance/sonic`: v1.11.2
- `github.com/twmb/franz-go`: v1.16.1
- `go.uber.org/zap`: v1.26.0
- `github.com/prometheus/client_golang`: v1.18.0
- `github.com/go-sql-driver/mysql`: v1.7.1

## 常见问题

### Q1: go mod download 失败

**解决方案**:
```bash
# 设置 GOPROXY
export GOPROXY=https://goproxy.cn,direct

# 或使用其他代理
export GOPROXY=https://goproxy.io,direct
```

### Q2: 编译时提示找不到包

**解决方案**:
```bash
# 确保 go.mod 中的 module 路径正确
# 当前为: github.com/doris-sinker/doris-sinker

# 如果需要修改，全局替换所有 import 路径
```

### Q3: sonic 相关错误

**解决方案**:
```bash
# sonic 需要 Go 1.16+ 和 amd64/arm64 架构
# 确保你的环境满足要求

go version  # 检查 Go 版本
go env GOARCH  # 检查架构
```

## 开发环境设置

### VSCode

创建 `.vscode/settings.json`:
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.formatTool": "goimports"
}
```

### GoLand

1. 打开项目
2. 设置 Go SDK 为 1.21+
3. 启用 Go Modules
4. 设置 GOPROXY

## 测试

```bash
# 运行所有测试
make test

# 运行特定包的测试
go test -v ./internal/schema/...

# 运行性能测试
make benchmark
```

## Docker 构建

```bash
# 构建镜像
make docker

# 或手动构建
docker build -t doris-sinker:1.0.0 -f deployments/docker/Dockerfile .
```

## 下一步

1. 配置 Kafka 和 Doris 连接信息
2. 创建目标表
3. 启动程序
4. 监控指标和日志

详细使用说明请参考 [docs/USAGE.md](docs/USAGE.md)
