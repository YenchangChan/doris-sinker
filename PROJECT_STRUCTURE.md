# Doris-Sinker 项目结构设计

## 目录结构

```
doris-sinker/
├── cmd/
│   └── doris-sinker/
│       └── main.go                    # 程序入口，负责启动和协调各模块
│
├── internal/                          # 私有应用代码（不可被外部导入）
│   ├── config/
│   │   ├── config.go                  # 配置结构定义
│   │   ├── loader.go                  # 配置加载器（从文件/环境变量）
│   │   └── validator.go               # 配置验证器
│   │
│   ├── consumer/
│   │   ├── consumer.go                # Kafka消费者接口定义
│   │   ├── franz.go                   # franz-go实现
│   │   └── metrics.go                 # 消费者相关指标
│   │
│   ├── schema/
│   │   ├── schema.go                  # Schema结构定义
│   │   ├── fetcher.go                 # 从Doris获取schema
│   │   ├── mapper.go                  # JSON到Row的映射器
│   │   ├── types.go                   # 类型转换和零值处理
│   │   └── cache.go                   # Schema缓存（可选）
│   │
│   ├── batcher/
│   │   ├── batcher.go                 # 批次管理器接口
│   │   ├── memory_batcher.go          # 内存批次实现
│   │   ├── trigger.go                 # 触发条件检查（rows/size/interval）
│   │   └── metrics.go                 # 批次相关指标
│   │
│   ├── writer/
│   │   ├── writer.go                  # Doris写入器接口
│   │   ├── stream_load.go             # Stream Load实现
│   │   ├── csv_builder.go             # CSV格式构建器
│   │   ├── retry.go                   # 重试逻辑
│   │   └── metrics.go                 # 写入相关指标
│   │
│   ├── pipeline/
│   │   ├── pipeline.go                # 数据处理流水线（协调各模块）
│   │   ├── worker.go                  # 工作协程池
│   │   └── context.go                 # 处理上下文（携带trace信息）
│   │
│   ├── metrics/
│   │   ├── registry.go                # Prometheus指标注册中心
│   │   ├── kafka_metrics.go           # Kafka相关指标定义
│   │   ├── batch_metrics.go           # 批次相关指标定义
│   │   ├── doris_metrics.go           # Doris相关指标定义
│   │   └── system_metrics.go          # 系统相关指标定义
│   │
│   ├── server/
│   │   ├── server.go                  # HTTP服务器（metrics + pprof + health）
│   │   ├── metrics_handler.go         # Prometheus指标端点
│   │   ├── pprof_handler.go           # pprof端点
│   │   └── health_handler.go          # 健康检查端点
│   │
│   └── signal/
│       └── signal.go                  # 优雅关闭信号处理
│
├── pkg/                               # 公共库代码（可被外部导入）
│   ├── logger/
│   │   ├── logger.go                  # zap日志封装
│   │   ├── config.go                  # 日志配置
│   │   └── sampling.go                # 日志采样配置
│   │
│   ├── errors/
│   │   ├── errors.go                  # 自定义错误类型
│   │   └── codes.go                   # 错误码定义
│   │
│   ├── pool/
│   │   └── buffer_pool.go             # 字节缓冲池（减少GC）
│   │
│   └── utils/
│       ├── retry.go                   # 通用重试工具
│       ├── backoff.go                 # 退避策略
│       └── converter.go               # 类型转换工具
│
├── configs/                           # 配置文件示例
│   ├── config.yaml                    # 默认配置
│   ├── config.dev.yaml                # 开发环境配置
│   └── config.prod.yaml               # 生产环境配置
│
├── deployments/                       # 部署相关文件
│   ├── docker/
│   │   ├── Dockerfile                 # Docker镜像构建
│   │   └── docker-compose.yaml        # 本地测试环境
│   │
│   ├── kubernetes/
│   │   ├── deployment.yaml            # K8s部署配置
│   │   ├── service.yaml               # K8s服务配置
│   │   ├── configmap.yaml             # K8s配置映射
│   │   └── hpa.yaml                   # 水平扩展配置
│   │
│   ├── grafana/
│   │   └── dashboard.json             # Grafana监控面板
│   │
│   └── prometheus/
│       └── alerts.yml                 # Prometheus告警规则
│
├── scripts/                           # 脚本工具
│   ├── build.sh                       # 编译脚本
│   ├── test.sh                        # 测试脚本
│   └── benchmark.sh                   # 性能测试脚本
│
├── test/                              # 测试文件
│   ├── integration/
│   │   ├── kafka_test.go              # Kafka集成测试
│   │   ├── doris_test.go              # Doris集成测试
│   │   └── e2e_test.go                # 端到端测试
│   │
│   ├── benchmark/
│   │   ├── mapper_bench_test.go       # 映射性能测试
│   │   └── batcher_bench_test.go      # 批次性能测试
│   │
│   └── testdata/
│       ├── sample.json                # 测试数据样本
│       └── schema.json                # 测试schema
│
├── docs/                              # 文档
│   ├── architecture.md                # 架构设计文档
│   ├── performance.md                 # 性能优化文档
│   └── troubleshooting.md             # 故障排查文档
│
├── .gitignore
├── go.mod                             # Go模块依赖
├── go.sum                             # 依赖校验和
├── Makefile                           # 构建任务
├── README.md                          # 项目说明
└── LICENSE                            # 开源协议

```

## 核心模块职责说明

### 1. cmd/doris-sinker/main.go
**职责**：程序入口，负责启动流程
- 加载配置
- 初始化日志
- 初始化各个模块
- 启动HTTP服务器（metrics + pprof）
- 启动数据处理流水线
- 监听退出信号，优雅关闭

**设计原则**：薄层，只做协调，不包含业务逻辑

---

### 2. internal/config/
**职责**：配置管理
- `config.go`：定义完整的配置结构体
- `loader.go`：从YAML文件或环境变量加载配置
- `validator.go`：验证配置合法性（必填项、范围检查等）

**设计原则**：
- 使用结构体标签（yaml, validate）
- 支持配置热加载（可选）
- 敏感信息（密码）支持环境变量覆盖

---

### 3. internal/consumer/
**职责**：Kafka消息消费
- `consumer.go`：定义Consumer接口（便于测试和替换实现）
- `franz.go`：franz-go的具体实现
- `metrics.go`：消费相关指标（消费速率、Lag等）

**接口设计**：
```go
type Consumer interface {
    Start(ctx context.Context) error
    Subscribe(topic string) error
    Consume() <-chan *Message
    Commit(offset int64) error
    Close() error
}
```

**设计原则**：
- 面向接口编程
- 错误通过channel返回
- 支持优雅关闭

---

### 4. internal/schema/
**职责**：Schema管理和字段映射
- `schema.go`：Schema结构定义（Column列表）
- `fetcher.go`：从Doris查询表结构（DESCRIBE TABLE）
- `mapper.go`：JSON字节数组到Row的映射（使用sonic）
- `types.go`：类型转换、零值填充
- `cache.go`：Schema缓存（避免频繁查询）

**核心方法**：
```go
// 获取Schema
func FetchSchema(ctx context.Context, cfg DorisConfig) (*Schema, error)

// 映射JSON到Row
func (s *Schema) MapJSONToRow(jsonBytes []byte) ([]interface{}, error)

// 获取零值
func GetZeroValue(dataType string) interface{}
```

**设计原则**：
- Schema不可变（immutable）
- 映射过程零内存分配
- 错误详细记录（哪个字段、什么错误）

---

### 5. internal/batcher/
**职责**：批次管理和触发
- `batcher.go`：Batcher接口定义
- `memory_batcher.go`：内存批次实现
- `trigger.go`：触发条件检查（rows/size/interval三者之一）
- `metrics.go`：批次指标（大小分布、延迟等）

**接口设计**：
```go
type Batcher interface {
    Add(row []interface{}) error
    Flush() ([][]interface{}, error)
    ShouldFlush() bool
    Size() int
    Close() error
}
```

**设计原则**：
- 线程安全（使用mutex或channel）
- 自动触发（定时器）
- 支持手动flush

---

### 6. internal/writer/
**职责**：Doris数据写入
- `writer.go`：Writer接口定义
- `stream_load.go`：Stream Load API实现
- `csv_builder.go`：将Row数组转换为CSV格式
- `retry.go`：重试逻辑（指数退避）
- `metrics.go`：写入指标（成功率、延迟等）

**接口设计**：
```go
type Writer interface {
    Write(ctx context.Context, batch [][]interface{}) error
    Close() error
}
```

**设计原则**：
- HTTP连接池复用
- 支持重试和超时
- 详细的错误日志

---

### 7. internal/pipeline/
**职责**：数据处理流水线（核心协调层）
- `pipeline.go`：流水线主逻辑
- `worker.go`：工作协程池（并发处理）
- `context.go`：处理上下文（携带trace_id等）

**流程**：
```
Kafka消费 → Schema映射 → 批次攒批 → Doris写入 → Offset提交
```

**设计原则**：
- 使用channel连接各模块
- 支持背压（backpressure）
- 错误隔离（一条消息失败不影响其他）

---

### 8. internal/metrics/
**职责**：Prometheus指标管理
- `registry.go`：全局指标注册中心
- `kafka_metrics.go`：Kafka指标（消费速率、Lag）
- `batch_metrics.go`：批次指标（大小、延迟）
- `doris_metrics.go`：Doris指标（写入成功率、延迟）
- `system_metrics.go`：系统指标（goroutine、内存）

**设计原则**：
- 使用promauto自动注册
- 指标命名规范（前缀统一）
- 合理的label设计

---

### 9. internal/server/
**职责**：HTTP服务器
- `server.go`：HTTP服务器启动和管理
- `metrics_handler.go`：`/metrics` 端点
- `pprof_handler.go`：`/debug/pprof/*` 端点
- `health_handler.go`：`/health` 和 `/ready` 端点

**端点设计**：
```
GET /metrics          - Prometheus指标
GET /health           - 健康检查（存活探针）
GET /ready            - 就绪检查（就绪探针）
GET /debug/pprof/*    - 性能分析
```

**设计原则**：
- 独立的goroutine运行
- 支持优雅关闭
- 健康检查包含依赖检查（Kafka、Doris）

---

### 10. internal/signal/
**职责**：优雅关闭
- `signal.go`：监听SIGINT/SIGTERM信号

**流程**：
```
1. 接收信号
2. 停止消费新消息
3. 等待当前批次处理完成
4. 提交offset
5. 关闭所有连接
6. 退出程序
```

---

### 11. pkg/logger/
**职责**：日志封装
- `logger.go`：zap日志封装
- `config.go`：日志配置（级别、输出、格式）
- `sampling.go`：日志采样配置

**设计原则**：
- 全局单例logger
- 结构化日志
- 支持动态调整日志级别

---

### 12. pkg/errors/
**职责**：错误处理
- `errors.go`：自定义错误类型
- `codes.go`：错误码定义

**错误分类**：
```go
const (
    ErrCodeKafkaConsume = 1001
    ErrCodeJSONParse    = 2001
    ErrCodeSchemaMap    = 2002
    ErrCodeDorisWrite   = 3001
)
```

---

### 13. pkg/pool/
**职责**：对象池
- `buffer_pool.go`：字节缓冲池（减少GC）

**使用场景**：
- CSV构建时的字节缓冲
- JSON解析时的临时缓冲

---

## 模块依赖关系

```
main.go
  ├─> config (加载配置)
  ├─> logger (初始化日志)
  ├─> metrics (初始化指标)
  ├─> server (启动HTTP服务)
  └─> pipeline (启动数据流水线)
        ├─> consumer (消费Kafka)
        ├─> schema (字段映射)
        ├─> batcher (批次管理)
        └─> writer (写入Doris)
```

**依赖原则**：
- 高层模块不依赖低层模块，都依赖抽象（接口）
- 内部模块不依赖外部模块
- pkg可以被internal使用，反之不行

---

## 接口设计原则

### 1. 面向接口编程
所有核心模块都定义接口，便于：
- 单元测试（mock实现）
- 替换实现（如换成其他Kafka客户端）
- 解耦模块

### 2. 依赖注入
通过构造函数注入依赖：
```go
func NewPipeline(
    consumer consumer.Consumer,
    schema *schema.Schema,
    batcher batcher.Batcher,
    writer writer.Writer,
    logger *zap.Logger,
) *Pipeline
```

### 3. 错误处理
- 使用`error`返回错误
- 关键错误使用自定义错误类型
- 错误信息包含上下文

### 4. 上下文传递
所有IO操作接收`context.Context`：
- 支持超时控制
- 支持取消操作
- 传递trace信息

---

## 性能优化点

### 1. 零拷贝
- sonic直接在字节数组上操作
- 避免JSON完整反序列化

### 2. 对象池
- 使用sync.Pool复用对象
- 减少GC压力

### 3. 批量处理
- 批量消费Kafka
- 批量写入Doris

### 4. 并发处理
- 使用goroutine并发映射
- 使用channel传递数据

---

## 测试策略

### 1. 单元测试
- 每个模块独立测试
- 使用mock接口
- 覆盖率 > 80%

### 2. 集成测试
- 使用testcontainers启动Kafka和Doris
- 测试完整流程

### 3. 性能测试
- Benchmark测试关键路径
- 压测工具测试吞吐量

---

## 构建和部署

### Makefile任务
```makefile
build:        # 编译二进制
test:         # 运行测试
benchmark:    # 性能测试
lint:         # 代码检查
docker:       # 构建Docker镜像
deploy:       # 部署到K8s
```

### Docker镜像
- 多阶段构建
- 最小化镜像大小
- 非root用户运行

---

## 总结

这个项目结构的优势：
1. **清晰的分层**：cmd → internal → pkg
2. **高内聚低耦合**：每个模块职责单一
3. **易于测试**：面向接口，依赖注入
4. **易于扩展**：新增功能只需实现接口
5. **生产就绪**：包含监控、日志、优雅关闭
6. **符合Go规范**：遵循Go项目布局标准

准备好开始编码了吗？我会按照这个结构逐步实现！
