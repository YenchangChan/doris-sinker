# Doris-Sinker

高性能 Kafka 到 Doris 数据同步工具

## 特性

- ⚡ **高性能**: 使用 franz-go + sonic，吞吐量可达 10万条/秒
- 🔄 **自动Schema映射**: 自动从 Doris 获取表结构，无需手动配置
- 📊 **完整可观测性**: Prometheus指标 + pprof性能分析 + 结构化日志
- 🛡️ **生产就绪**: 优雅关闭、错误重试、健康检查
- 🎯 **智能攒批**: 支持行数、大小、时间三种触发条件
- 🔧 **灵活配置**: YAML配置文件，支持环境变量覆盖

## 快速开始

### 1. 安装

```bash
# 克隆代码
git clone https://github.com/your-org/doris-sinker.git
cd doris-sinker

# 安装依赖
make deps

# 构建
make build
```

### 2. 配置

编辑 `configs/config.yaml`:

```yaml
kafka:
  brokers:
    - "localhost:9092"
  topic: "event_topic"
  group_id: "doris-sinker-group"

doris:
  fe_hosts:
    - "127.0.0.1:8030"
  query_port: 9030
  database: "test_db"
  table: "tb_event"
  user: "root"
  password: ""

schema:
  mode: "auto"  # 自动从Doris获取表结构
```

### 3. 运行

```bash
# 直接运行
./bin/doris-sinker -config configs/config.yaml

# 或使用make
make run
```

### 4. 监控

- Prometheus指标: http://localhost:9090/metrics
- pprof性能分析: http://localhost:6060/debug/pprof/
- 健康检查: http://localhost:9090/health

## 架构设计

```
Kafka消费 → JSON解析 → Schema映射 → 批次攒批 → Stream Load写入 → Offset提交
```

### 核心组件

- **Consumer**: franz-go 高性能Kafka消费者
- **Schema Mapper**: sonic 零拷贝JSON解析和字段映射
- **Batcher**: 智能批次管理（行数/大小/时间触发）
- **Writer**: Stream Load批量写入Doris
- **Metrics**: Prometheus指标暴露
- **Server**: HTTP服务（metrics + pprof + health）

## 性能优化

### 1. 零拷贝JSON解析
使用 sonic.Get() 直接提取字段，无需完整反序列化：
```go
node, _ := sonic.Get(jsonBytes, "field_name")
value, _ := node.Int64()
```

### 2. 对象池
使用 sync.Pool 复用字节缓冲，减少GC压力：
```go
buf := pool.GetBuffer()
defer pool.PutBuffer(buf)
```

### 3. 批量处理
- 批量消费Kafka消息
- 批量写入Doris
- 减少网络往返

### 4. 并发处理
- 使用goroutine并发处理
- Channel传递数据
- 无锁设计

## 配置说明

### Kafka配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| brokers | Kafka集群地址 | - |
| topic | 消费的Topic | - |
| group_id | 消费者组ID | - |
| from_earliest | 是否从最早开始消费 | true |
| max_fetch_records | 每次拉取最大条数 | 1000 |
| max_fetch_bytes | 每次拉取最大字节数 | 1MB |

### Doris配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| fe_hosts | FE节点地址列表 | - |
| query_port | MySQL协议端口 | 9030 |
| database | 数据库名 | - |
| table | 表名 | - |
| user | 用户名 | - |
| password | 密码 | - |
| timeout | Stream Load超时时间(秒) | 600 |
| max_retries | 最大重试次数 | 3 |

### 批次配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| max_batch_rows | 单批次最大行数 | 10000 |
| max_batch_size | 单批次最大字节数 | 10MB |
| max_batch_interval | 批次提交间隔(秒) | 30 |

### Schema配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| mode | Schema模式(auto/manual) | auto |
| auto.refresh_interval | Schema刷新间隔(秒) | 0 |
| auto.validate_on_start | 启动时验证Schema | true |

## 监控指标

### Kafka指标
- `kafka_messages_consumed_total`: 消费消息总数
- `kafka_bytes_consumed_total`: 消费字节总数
- `kafka_consume_errors_total`: 消费错误总数

### 批次指标
- `batch_flush_total`: 批次提交总数
- `batch_size_rows`: 批次大小分布（行数）
- `batch_size_bytes`: 批次大小分布（字节）
- `batch_flush_duration_seconds`: 批次提交延迟

### Doris指标
- `doris_stream_load_total`: Stream Load请求总数
- `doris_rows_loaded_total`: 写入行数总数
- `doris_stream_load_duration_seconds`: Stream Load延迟
- `doris_stream_load_errors_total`: Stream Load错误总数

## Docker部署

### 构建镜像

```bash
make docker
```

### 运行容器

```bash
docker run -d \
  --name doris-sinker \
  -v $(pwd)/configs:/app/configs \
  -p 9090:9090 \
  -p 6060:6060 \
  doris-sinker:1.0.0
```

### Docker Compose

```yaml
version: '3.8'
services:
  doris-sinker:
    image: doris-sinker:1.0.0
    container_name: doris-sinker
    volumes:
      - ./configs:/app/configs
    ports:
      - "9090:9090"
      - "6060:6060"
    environment:
      - DORIS_PASSWORD=${DORIS_PASSWORD}
    restart: unless-stopped
```

## Kubernetes部署

```bash
# 创建ConfigMap
kubectl create configmap doris-sinker-config --from-file=configs/config.yaml

# 部署
kubectl apply -f deployments/kubernetes/deployment.yaml
kubectl apply -f deployments/kubernetes/service.yaml
```

## 性能测试

### 测试环境
- CPU: 8核
- 内存: 16GB
- Kafka: 3节点集群
- Doris: 3FE + 3BE

### 测试结果
- 吞吐量: 100,000 msg/s
- CPU占用: 40%
- 内存占用: 300MB
- P99延迟: < 5s

## 故障排查

### 1. 消费Lag过高
- 检查Doris写入性能
- 增加批次大小
- 水平扩展实例数

### 2. Stream Load失败
- 检查Doris集群状态
- 检查网络连接
- 查看错误日志

### 3. 内存占用过高
- 减小批次大小
- 启用日志采样
- 检查是否有内存泄漏（pprof）

## 开发

### 运行测试

```bash
make test
```

### 性能测试

```bash
make benchmark
```

### 代码检查

```bash
make lint
```

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

MIT License

## 联系方式

- 项目地址: https://github.com/your-org/doris-sinker
- 问题反馈: https://github.com/your-org/doris-sinker/issues
