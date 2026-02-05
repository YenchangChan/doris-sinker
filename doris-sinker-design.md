# Doris-Sinker 设计方案

## 1. 技术选型

### 1.1 编程语言：Go
**选择理由：**
- **高性能**：原生协程支持，高并发处理能力强
- **内存效率**：编译型语言，内存占用低，GC优化好
- **稳定性**：强类型系统，编译时错误检查
- **生态成熟**：
  - Kafka客户端：sarama（最成熟的Go Kafka客户端）
  - JSON处理：encoding/json（标准库）
  - HTTP客户端：net/http（标准库，性能优秀）
- **部署简单**：单一可执行文件，无运行时依赖

### 1.2 核心依赖库
```
github.com/twmb/franz-go/pkg/kgo       # Kafka客户端（高性能，比sarama快2-3倍）
github.com/bytedance/sonic             # JSON处理（字节跳动开源，比标准库快10倍+）
go.uber.org/zap                        # 日志库（uber开源，高性能结构化日志）
github.com/prometheus/client_golang    # Prometheus指标暴露
github.com/go-sql-driver/mysql         # MySQL驱动（Doris兼容MySQL协议）
gopkg.in/yaml.v3                       # 配置文件解析
net/http/pprof                         # 性能分析（标准库）
```

**性能优化说明：**
- **franz-go**: 
  - 零拷贝设计，内存分配更少
  - 更好的批处理支持
  - 更低的延迟和更高的吞吐量
  - 原生支持 Kafka 3.x 新特性
  
- **sonic**: 
  - 使用 JIT 编译和 SIMD 指令加速
  - 支持按需解析（Get API），无需完整反序列化
  - 内存占用更低
  - 特别适合我们只需要提取特定字段的场景

- **zap**:
  - 零内存分配的结构化日志
  - 比标准库快10倍，比logrus快4-5倍
  - 支持日志采样，避免高频日志影响性能
  - 类型安全的字段记录

- **prometheus**:
  - 业界标准的监控指标暴露
  - 支持Counter、Gauge、Histogram、Summary等指标类型
  - 与Grafana完美集成

- **pprof**:
  - Go标准库的性能分析工具
  - 支持CPU、内存、goroutine、阻塞等分析
  - 通过HTTP接口实时查看性能数据

## 2. 架构设计

### 2.1 整体架构
```
┌─────────────────────────────────────────────────────────────┐
│                      Doris-Sinker                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────┐ │
│  │   Kafka      │      │   Batcher    │      │  Doris   │ │
│  │  Consumer    │─────▶│   (攒批)     │─────▶│  Writer  │ │
│  └──────────────┘      └──────────────┘      └──────────┘ │
│         │                      │                    │      │
│         │                      │                    │      │
│  ┌──────▼──────┐      ┌───────▼──────┐    ┌────────▼────┐ │
│  │   Message   │      │   Batch      │    │  Stream     │ │
│  │   Queue     │      │   Manager    │    │  Load API   │ │
│  └─────────────┘      └──────────────┘    └─────────────┘ │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │           Schema Mapper (字段映射)                   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │           Config Manager (配置管理)                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 核心模块

#### 2.2.1 Kafka Consumer（消费者模块）
- 使用 franz-go 实现高性能消费
- 支持消费者组管理和自动rebalance
- 支持offset管理（自动提交/手动提交）
- 可配置拉取参数（max_fetch_records, max_fetch_bytes）
- 零拷贝读取，减少内存分配

#### 2.2.2 Schema Mapper（字段映射模块）
- **自动Schema发现**：启动时从 Doris 查询表结构（DESCRIBE TABLE）
- **智能字段映射**：根据列名自动从 JSON 中提取对应字段
- 使用 sonic.Get() API 按需提取字段，无需完整反序列化
- 字段缺失时填充零值（根据数据类型）
- 忽略 JSON 中多余的字段
- 零内存拷贝的字段提取
- **支持两种模式**：
  - 自动模式：从 Doris 获取 schema（推荐）
  - 手动模式：从配置文件读取 schema（用于无法连接 Doris 的场景）

#### 2.2.3 Batcher（攒批模块）
- 在内存中攒批数据
- 支持三种触发条件：
  - max_batch_rows：最大行数
  - max_batch_size：最大字节数
  - max_batch_interval：最大时间间隔
- 先达到任一条件即触发提交

#### 2.2.4 Doris Writer（写入模块）
- 通过Stream Load API写入Doris
- 支持重试机制
- 错误处理和日志记录
- 支持事务保证

## 3. 配置设计

### 3.1 配置文件格式（YAML）
```yaml
# Kafka配置
kafka:
  brokers:
    - "localhost:9092"
    - "localhost:9093"
  topic: "event_topic"
  group_id: "doris-sinker-group"
  # 是否从最早开始消费（true: earliest, false: latest）
  from_earliest: true
  # 每次拉取最大条数
  max_fetch_records: 1000
  # 每次拉取最大字节数（单位：字节）
  max_fetch_bytes: 1048576  # 1MB
  # 消费超时时间（秒）
  session_timeout: 30
  # 心跳间隔（秒）
  heartbeat_interval: 3

# Doris配置
doris:
  # FE地址列表（支持多个FE高可用）
  fe_hosts:
    - "127.0.0.1:8030"
  # MySQL协议端口（用于查询表结构，通常是9030）
  query_port: 9030
  # 数据库名
  database: "test_db"
  # 表名
  table: "tb_event"
  # 用户名
  user: "root"
  # 密码
  password: ""
  # Stream Load超时时间（秒）
  timeout: 600
  # 最大重试次数
  max_retries: 3

# 攒批配置
batch:
  # 单批次最大条数
  max_batch_rows: 10000
  # 单批次最大字节数（单位：字节）
  max_batch_size: 10485760  # 10MB
  # 批次提交间隔（秒）
  max_batch_interval: 30

# 表结构配置
schema:
  # Schema获取模式：auto（自动从Doris获取）, manual（手动配置）
  mode: "auto"
  
  # 自动模式配置（mode=auto时生效）
  auto:
    # Schema刷新间隔（秒），0表示只在启动时获取一次
    refresh_interval: 0
    # 是否在启动时验证schema
    validate_on_start: true
  
  # 手动模式配置（mode=manual时生效）
  # 当无法连接Doris或需要自定义字段顺序时使用
  manual:
    columns:
      - name: "id"
        type: "BIGINT"
      - name: "integration_id"
        type: "BIGINT"
      - name: "arrive_time"
        type: "BIGINT"
      # ... 其他字段省略，实际使用时需要完整配置
      type: "VARCHAR"
    # ... extend3 到 extend50 省略，实际配置需要全部列出

# 日志配置
log:
  # 日志级别：debug, info, warn, error
  level: "info"
  # 日志输出：stdout, file, both
  output: "stdout"
  # 日志文件路径（当output为file或both时）
  file_path: "/var/log/doris-sinker.log"
  # 日志格式：json, console
  format: "json"
  # 是否启用日志采样（高频日志只记录部分，避免影响性能）
  enable_sampling: true
  # 日志文件最大大小（MB）
  max_size: 100
  # 日志文件最大保留天数
  max_age: 7
  # 日志文件最大备份数
  max_backups: 10

# 监控配置
metrics:
  # 是否启用Prometheus指标
  enabled: true
  # 指标暴露端口
  port: 9090
  # 指标路径
  path: "/metrics"

# 性能分析配置
pprof:
  # 是否启用pprof
  enabled: true
  # pprof端口
  port: 6060
```

## 4. 核心流程设计

### 4.1 启动流程
```
1. 加载配置文件
2. 初始化日志系统
3. 验证配置有效性
4. 初始化Kafka消费者
5. 初始化Schema映射器
   5.1 如果是auto模式：从Doris查询表结构（DESCRIBE TABLE）
   5.2 如果是manual模式：从配置文件加载schema
   5.3 构建字段映射关系（列名 -> 索引位置）
   5.4 缓存字段类型信息（用于零值填充）
6. 初始化Batcher
7. 初始化Doris Writer
8. 启动监控服务（Prometheus + pprof）
9. 启动消费循环
```

### 4.2 消费流程
```
┌─────────────────┐
│  Kafka Consumer │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Parse JSON     │ ◄─── 解析JSON消息
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Schema Mapping │ ◄─── 字段映射 + 零值填充
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Add to Batch   │ ◄─── 添加到批次
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │ 是否触发 │ ◄─── 检查触发条件
    │ 提交？  │      (rows/size/interval)
    └───┬────┘
        │
    Yes │
        ▼
┌─────────────────┐
│  Flush Batch    │ ◄─── 提交批次到Doris
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Commit Offset  │ ◄─── 提交Kafka offset
└─────────────────┘
```

### 4.3 自动Schema获取流程
```go
// 伪代码 - 从Doris自动获取表结构
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql" // Doris兼容MySQL协议
)

// 从Doris获取表结构
func FetchSchemaFromDoris(host, database, table, user, password string) ([]Column, error) {
    // 1. 连接Doris（使用MySQL协议，端口通常是9030）
    dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", user, password, host, database)
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    defer db.Close()
    
    // 2. 执行DESCRIBE TABLE查询
    query := fmt.Sprintf("DESCRIBE %s.%s", database, table)
    rows, err := db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    // 3. 解析结果
    var columns []Column
    for rows.Next() {
        var field, dataType, null, key, defaultVal, extra string
        err := rows.Scan(&field, &dataType, &null, &key, &defaultVal, &extra)
        if err != nil {
            return nil, err
        }
        
        // 4. 映射Doris类型到内部类型
        col := Column{
            Name: field,
            Type: mapDorisType(dataType),
        }
        columns = append(columns, col)
    }
    
    return columns, nil
}

// 映射Doris数据类型
func mapDorisType(dorisType string) string {
    // 提取基础类型（去除长度等信息）
    baseType := strings.ToUpper(strings.Split(dorisType, "(")[0])
    
    switch baseType {
    case "TINYINT", "SMALLINT", "INT", "INTEGER":
        return "INT"
    case "BIGINT":
        return "BIGINT"
    case "BOOLEAN", "BOOL":
        return "BOOLEAN"
    case "FLOAT", "DOUBLE", "DECIMAL":
        return "FLOAT"
    case "CHAR", "VARCHAR":
        return "VARCHAR"
    case "STRING", "TEXT", "MEDIUMTEXT", "LONGTEXT":
        return "STRING"
    case "DATE":
        return "DATE"
    case "DATETIME", "TIMESTAMP":
        return "DATETIME"
    default:
        return "STRING" // 默认按字符串处理
    }
}

// Schema示例输出：
// [
//   {Name: "id", Type: "BIGINT"},
//   {Name: "integration_id", Type: "BIGINT"},
//   {Name: "arrive_time", Type: "BIGINT"},
//   {Name: "event_time", Type: "BIGINT"},
//   ...
//   {Name: "extend1", Type: "VARCHAR"},
//   {Name: "extend2", Type: "VARCHAR"},
//   ...
// ]
```

### 4.4 字段映射逻辑（使用 sonic 优化）
```go
// 伪代码 - 使用 sonic 按需提取字段
import "github.com/bytedance/sonic"

func MapJSONToRow(jsonBytes []byte, schema []Column) ([]interface{}, error) {
    row := make([]interface{}, len(schema))
    
    for i, col := range schema {
        // 使用 sonic.Get() 直接提取字段，无需完整解析
        // 假设JSON的key名与表字段名一致
        node, err := sonic.Get(jsonBytes, col.Name)
        
        if err != nil || !node.Exists() {
            // 字段不存在，填充零值
            row[i] = getZeroValue(col.Type)
            continue
        }
        
        // 根据类型提取值
        switch col.Type {
        case "BIGINT", "INT":
            val, _ := node.Int64()
            row[i] = val
        case "BOOLEAN":
            val, _ := node.Bool()
            row[i] = val
        case "FLOAT":
            val, _ := node.Float64()
            row[i] = val
        case "VARCHAR", "STRING":
            val, _ := node.String()
            row[i] = val
        case "DATE", "DATETIME":
            val, _ := node.String()
            row[i] = val
        default:
            row[i] = getZeroValue(col.Type)
        }
    }
    
    return row, nil
}

func getZeroValue(dataType string) interface{} {
    switch dataType {
    case "BIGINT", "INT":
        return 0
    case "BOOLEAN":
        return false
    case "FLOAT":
        return 0.0
    case "VARCHAR", "STRING", "DATE", "DATETIME":
        return ""
    default:
        return nil
    }
}

// 性能优势：
// 1. 无需将整个JSON反序列化为map[string]interface{}
// 2. 只提取需要的字段，跳过不需要的字段
// 3. 零内存拷贝，直接在原始字节上操作
// 4. 对于大JSON（如包含大量extend字段），性能提升明显
// 5. 自动适配表结构变化，无需手动维护配置
```
```

### 4.5 Stream Load写入流程
```go
// 伪代码
func WriteToDoris(batch [][]interface{}) error {
    // 1. 构造CSV格式数据
    csvData := convertToCSV(batch)
    
    // 2. 构造HTTP请求
    url := fmt.Sprintf("http://%s/api/%s/%s/_stream_load", 
                       feHost, database, table)
    
    req := &http.Request{
        Method: "PUT",
        URL:    url,
        Body:   csvData,
        Header: {
            "Authorization": basicAuth(user, password),
            "Expect": "100-continue",
            "Content-Type": "text/plain",
            "format": "csv",
            "column_separator": ",",
        },
    }
    
    // 3. 发送请求
    resp := httpClient.Do(req)
    
    // 4. 处理响应
    if resp.StatusCode == 200 {
        // 解析响应，检查导入状态
        result := parseStreamLoadResult(resp.Body)
        if result.Status == "Success" {
            return nil
        }
    }
    
    return errors.New("stream load failed")
}
```

## 5. 容错设计

### 5.1 Kafka消费容错
- 使用消费者组，支持自动rebalance
- 手动提交offset，确保数据不丢失
- 消费失败时记录日志，继续处理下一条

### 5.2 Doris写入容错
- Stream Load失败时自动重试（可配置重试次数）
- 重试失败后将数据写入失败队列或文件
- 支持断点续传（通过offset管理）

### 5.3 字段映射容错
- JSON解析失败时跳过该条消息，记录错误日志
- 字段类型转换失败时使用零值
- 未知字段自动忽略

### 5.4 优雅关闭
- 捕获SIGINT/SIGTERM信号
- 停止消费新消息
- 等待当前批次处理完成
- 提交offset
- 关闭连接

## 6. 可观测性设计

### 6.1 Prometheus 指标

#### 6.1.1 Kafka消费指标
```go
// 消费消息总数（Counter）
kafka_messages_consumed_total{topic="event_topic"}

// 消费字节总数（Counter）
kafka_bytes_consumed_total{topic="event_topic"}

// 消费延迟/Lag（Gauge）
kafka_consumer_lag{topic="event_topic", partition="0"}

// 消费速率（通过rate计算）
rate(kafka_messages_consumed_total[1m])

// 消费错误总数（Counter）
kafka_consume_errors_total{topic="event_topic", error_type="parse_error"}
```

#### 6.1.2 批次处理指标
```go
// 批次提交总数（Counter）
batch_flush_total{status="success"}
batch_flush_total{status="failed"}

// 批次大小分布（Histogram）
batch_size_rows{quantile="0.5"}  # 中位数
batch_size_rows{quantile="0.95"} # P95
batch_size_rows{quantile="0.99"} # P99

// 批次字节数分布（Histogram）
batch_size_bytes{quantile="0.5"}
batch_size_bytes{quantile="0.95"}
batch_size_bytes{quantile="0.99"}

// 批次处理延迟（Histogram，单位：秒）
batch_flush_duration_seconds{quantile="0.5"}
batch_flush_duration_seconds{quantile="0.95"}
batch_flush_duration_seconds{quantile="0.99"}

// 当前批次行数（Gauge）
batch_current_rows

// 当前批次字节数（Gauge）
batch_current_bytes
```

#### 6.1.3 Doris写入指标
```go
// Stream Load请求总数（Counter）
doris_stream_load_total{status="success"}
doris_stream_load_total{status="failed"}

// Stream Load写入行数（Counter）
doris_rows_loaded_total

// Stream Load过滤行数（Counter）
doris_rows_filtered_total

// Stream Load延迟（Histogram，单位：秒）
doris_stream_load_duration_seconds{quantile="0.5"}
doris_stream_load_duration_seconds{quantile="0.95"}
doris_stream_load_duration_seconds{quantile="0.99"}

// Stream Load重试次数（Counter）
doris_stream_load_retries_total

// Stream Load错误（Counter）
doris_stream_load_errors_total{error_type="timeout"}
doris_stream_load_errors_total{error_type="network"}
```

#### 6.1.4 字段映射指标
```go
// JSON解析错误总数（Counter）
json_parse_errors_total{field="priority"}

// 字段缺失总数（Counter）
field_missing_total{field="extend10"}

// 字段类型转换错误（Counter）
field_type_conversion_errors_total{field="severity", from_type="string", to_type="int"}
```

#### 6.1.5 系统指标
```go
// Go运行时指标（自动采集）
go_goroutines                    # goroutine数量
go_threads                       # 线程数量
go_memstats_alloc_bytes         # 已分配内存
go_memstats_heap_inuse_bytes    # 堆内存使用
go_gc_duration_seconds          # GC耗时
process_cpu_seconds_total       # CPU使用时间
process_resident_memory_bytes   # 常驻内存
```

#### 6.1.6 指标暴露示例
```go
// 伪代码
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    // Counter示例
    messagesConsumed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kafka_messages_consumed_total",
            Help: "Total number of messages consumed from Kafka",
        },
        []string{"topic"},
    )
    
    // Gauge示例
    consumerLag = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "kafka_consumer_lag",
            Help: "Current consumer lag",
        },
        []string{"topic", "partition"},
    )
    
    // Histogram示例
    batchFlushDuration = promauto.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "batch_flush_duration_seconds",
            Help:    "Batch flush duration in seconds",
            Buckets: prometheus.DefBuckets, // 或自定义: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
        },
    )
)

// 启动metrics服务
func StartMetricsServer(port int) {
    http.Handle("/metrics", promhttp.Handler())
    go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
```

### 6.2 pprof 性能分析

#### 6.2.1 pprof端点
```bash
# CPU性能分析（采样30秒）
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# 内存分配分析
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# goroutine分析
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof

# 阻塞分析
curl http://localhost:6060/debug/pprof/block > block.prof
go tool pprof block.prof

# 互斥锁分析
curl http://localhost:6060/debug/pprof/mutex > mutex.prof
go tool pprof mutex.prof

# 实时查看（Web界面）
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap
```

#### 6.2.2 pprof集成代码
```go
// 伪代码
import (
    _ "net/http/pprof"
    "net/http"
)

// 启动pprof服务
func StartPprofServer(port int) {
    go func() {
        log.Info("Starting pprof server", zap.Int("port", port))
        if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
            log.Error("pprof server failed", zap.Error(err))
        }
    }()
}
```

### 6.3 日志输出（使用 zap 结构化日志）

### 6.4 日志输出（使用 zap 结构化日志）
```json
// 消费日志
{"level":"info","ts":1709567890.123,"msg":"consumed messages","count":1000,"offset":123456,"lag":50}

// 批次提交日志
{"level":"info","ts":1709567891.456,"msg":"batch flushed","rows":10000,"size_mb":5.2,"duration_ms":1200}

// 写入成功日志
{"level":"info","ts":1709567892.789,"msg":"stream load success","rows":10000,"filtered":0,"loaded":10000,"load_time_ms":800}

// 写入失败日志
{"level":"error","ts":1709567893.012,"msg":"stream load failed","error":"connection timeout","retry":1,"max_retries":3}

// JSON解析错误日志
{"level":"warn","ts":1709567894.345,"msg":"json parse error","field":"priority","value":"invalid","offset":123457}
```

**zap 日志优势：**
- 结构化输出，便于日志分析和监控
- 零内存分配，不影响性能
- 支持日志采样（高频日志只记录部分）
- 类型安全，避免格式化错误

### 6.5 Grafana 监控面板

#### 6.5.1 推荐监控面板布局
```
┌─────────────────────────────────────────────────────────┐
│                    Doris-Sinker 监控                     │
├─────────────────────────────────────────────────────────┤
│  消费速率 │ 消费Lag │ 批次大小 │ 写入成功率 │ 错误率    │
├─────────────────────────────────────────────────────────┤
│  消费趋势图（折线图）                                     │
│  - rate(kafka_messages_consumed_total[1m])              │
├─────────────────────────────────────────────────────────┤
│  Lag趋势图（折线图）                                      │
│  - kafka_consumer_lag                                   │
├─────────────────────────────────────────────────────────┤
│  批次延迟分布（热力图）                                   │
│  - batch_flush_duration_seconds                         │
├─────────────────────────────────────────────────────────┤
│  Doris写入延迟（折线图）                                  │
│  - doris_stream_load_duration_seconds                   │
├─────────────────────────────────────────────────────────┤
│  系统资源（CPU/内存/Goroutine）                          │
│  - process_cpu_seconds_total                            │
│  - go_memstats_heap_inuse_bytes                         │
│  - go_goroutines                                        │
└─────────────────────────────────────────────────────────┘
```

#### 6.5.2 关键告警规则
```yaml
# Prometheus告警规则示例
groups:
  - name: doris_sinker_alerts
    rules:
      # 消费Lag过高
      - alert: HighConsumerLag
        expr: kafka_consumer_lag > 100000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Kafka consumer lag is too high"
          description: "Consumer lag is {{ $value }} for topic {{ $labels.topic }}"
      
      # 写入失败率过高
      - alert: HighStreamLoadFailureRate
        expr: rate(doris_stream_load_total{status="failed"}[5m]) / rate(doris_stream_load_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Stream load failure rate is too high"
          description: "Failure rate is {{ $value | humanizePercentage }}"
      
      # 批次延迟过高
      - alert: HighBatchFlushLatency
        expr: histogram_quantile(0.99, batch_flush_duration_seconds) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Batch flush latency is too high"
          description: "P99 latency is {{ $value }}s"
      
      # 内存使用过高
      - alert: HighMemoryUsage
        expr: go_memstats_heap_inuse_bytes / 1024 / 1024 / 1024 > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Memory usage is too high"
          description: "Heap memory usage is {{ $value }}GB"
      
      # Goroutine泄漏
      - alert: GoroutineLeak
        expr: go_goroutines > 1000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Possible goroutine leak"
          description: "Goroutine count is {{ $value }}"
```

### 6.6 可观测性最佳实践

1. **指标采集频率**：Prometheus默认15秒抓取一次
2. **指标保留时间**：建议至少保留15天
3. **日志采样**：高频日志（如每条消息）启用采样，避免影响性能
4. **pprof使用**：
   - 生产环境建议启用，但限制访问权限
   - 定期采样分析，发现性能瓶颈
   - 内存泄漏排查时重点关注heap和goroutine
5. **告警配置**：
   - 设置合理的阈值，避免告警疲劳
   - 关键指标（Lag、失败率）设置多级告警
   - 告警通知接入企业IM（钉钉/企微/飞书）

## 7. 性能优化

### 7.1 并发处理
- 使用goroutine并发消费
- 批量写入减少网络开销
- 连接池复用HTTP连接
- franz-go 的零拷贝消费

### 7.2 内存优化
- 限制批次大小，避免OOM
- 及时释放已处理数据
- 使用对象池减少GC压力
- sonic 按需解析，避免完整反序列化
- 直接在字节数组上操作，减少内存分配

### 7.3 网络优化
- 使用HTTP Keep-Alive
- 压缩传输数据（gzip）
- 合理设置超时时间
- franz-go 的批量拉取优化

### 7.4 性能对比预估
```
标准方案 (sarama + encoding/json):
- 吞吐量: ~30,000 msg/s
- CPU占用: ~60%
- 内存占用: ~500MB

优化方案 (franz-go + sonic):
- 吞吐量: ~100,000 msg/s (提升3倍+)
- CPU占用: ~40% (降低33%)
- 内存占用: ~300MB (降低40%)
- GC暂停: 减少50%+
```

## 8. 部署方案

### 8.1 单机部署
```bash
# 1. 编译
go build -o doris-sinker main.go

# 2. 配置
vim config.yaml

# 3. 运行
./doris-sinker -config config.yaml

# 4. 后台运行
nohup ./doris-sinker -config config.yaml > sinker.log 2>&1 &
```

### 8.2 Docker部署
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o doris-sinker main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/doris-sinker .
COPY config.yaml .
CMD ["./doris-sinker", "-config", "config.yaml"]
```

### 8.3 Kubernetes部署
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: doris-sinker
spec:
  replicas: 3
  selector:
    matchLabels:
      app: doris-sinker
  template:
    metadata:
      labels:
        app: doris-sinker
    spec:
      containers:
      - name: doris-sinker
        image: doris-sinker:latest
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
      volumes:
      - name: config
        configMap:
          name: doris-sinker-config
```

## 9. 测试方案

### 9.1 单元测试
- Schema映射逻辑测试
- 零值填充测试
- 批次管理测试
- CSV格式转换测试

### 9.2 集成测试
- Kafka消费测试
- Doris写入测试
- 端到端流程测试

### 9.3 性能测试
- 吞吐量测试（目标：10万条/秒）
- 延迟测试（目标：P99 < 5s）
- 稳定性测试（7x24小时运行）

## 10. 项目结构

```
doris-sinker/
├── cmd/
│   └── main.go              # 程序入口
├── config/
│   ├── config.go            # 配置结构定义
│   └── config.yaml          # 配置文件示例
├── internal/
│   ├── consumer/
│   │   └── kafka.go         # Kafka消费者
│   ├── mapper/
│   │   └── schema.go        # Schema映射器
│   ├── batcher/
│   │   └── batcher.go       # 攒批管理器
│   ├── writer/
│   │   └── doris.go         # Doris写入器
│   ├── metrics/
│   │   └── metrics.go       # Prometheus指标定义
│   └── model/
│       └── types.go         # 数据类型定义
├── pkg/
│   ├── logger/
│   │   └── logger.go        # 日志工具
│   └── server/
│       └── server.go        # HTTP服务器（metrics + pprof）
├── deployments/
│   ├── grafana/
│   │   └── dashboard.json   # Grafana监控面板
│   └── prometheus/
│       └── alerts.yml       # Prometheus告警规则
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── README.md
```

## 11. 开发计划

### Phase 1: 核心功能（3天）
- Day 1: 项目框架搭建 + 配置管理 + Kafka消费者
- Day 2: Schema映射 + 攒批管理
- Day 3: Doris写入 + 端到端联调

### Phase 2: 容错与优化（2天）
- Day 4: 错误处理 + 重试机制 + 优雅关闭
- Day 5: 性能优化 + 日志系统

### Phase 3: 可观测性（1天）
- Day 6: Prometheus指标 + pprof集成 + Grafana面板

### Phase 4: 测试与文档（1天）
- Day 7: 单元测试 + 集成测试 + 使用文档

## 12. 风险与挑战

### 12.1 潜在风险
- Kafka消费速度跟不上生产速度，导致lag增大
- Doris写入性能瓶颈
- 内存占用过高导致OOM
- 网络抖动导致写入失败

### 12.2 应对措施
- 支持水平扩展（多实例部署）
- 合理配置批次大小
- 实现背压机制
- 完善的重试和降级策略

## 13. 后续扩展

- 支持多表写入
- 支持数据转换（UDF）
- 支持动态配置更新
- 支持exactly-once语义
- 支持数据去重
- 支持分布式追踪（OpenTelemetry）
- 支持告警通知（钉钉/企微/飞书）
- 支持健康检查接口（/health, /ready）
