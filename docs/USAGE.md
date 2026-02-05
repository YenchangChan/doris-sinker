# Doris-Sinker 使用指南

## 目录
1. [快速开始](#快速开始)
2. [配置详解](#配置详解)
3. [运行模式](#运行模式)
4. [监控告警](#监控告警)
5. [故障排查](#故障排查)
6. [最佳实践](#最佳实践)

## 快速开始

### 前置条件
- Go 1.21+
- Kafka集群
- Doris集群
- 已创建目标表

### 安装步骤

1. **下载代码**
```bash
git clone https://github.com/your-org/doris-sinker.git
cd doris-sinker
```

2. **安装依赖**
```bash
make deps
```

3. **配置文件**
```bash
cp configs/config.yaml configs/config.prod.yaml
vim configs/config.prod.yaml
```

4. **构建**
```bash
make build
```

5. **运行**
```bash
./bin/doris-sinker -config configs/config.prod.yaml
```

## 配置详解

### Schema自动映射

**推荐使用auto模式**，程序会自动从Doris获取表结构：

```yaml
schema:
  mode: "auto"
  auto:
    refresh_interval: 0  # 0表示只在启动时获取一次
    validate_on_start: true
```

**手动模式**（当无法连接Doris时使用）：

```yaml
schema:
  mode: "manual"
  manual:
    columns:
      - name: "id"
        type: "BIGINT"
      - name: "name"
        type: "VARCHAR"
      # ... 其他字段
```

### 批次调优

根据数据特点调整批次参数：

**小消息高频场景**（如日志）：
```yaml
batch:
  max_batch_rows: 50000
  max_batch_size: 52428800  # 50MB
  max_batch_interval: 60
```

**大消息低频场景**（如订单）：
```yaml
batch:
  max_batch_rows: 1000
  max_batch_size: 10485760  # 10MB
  max_batch_interval: 10
```

### Kafka调优

**高吞吐场景**：
```yaml
kafka:
  max_fetch_records: 5000
  max_fetch_bytes: 10485760  # 10MB
```

**低延迟场景**：
```yaml
kafka:
  max_fetch_records: 100
  max_fetch_bytes: 102400  # 100KB
```

## 运行模式

### 开发模式

```bash
# 使用开发配置
./bin/doris-sinker -config configs/config.dev.yaml

# 日志级别设为debug
# 批次大小较小，便于调试
```

### 生产模式

```bash
# 使用生产配置
./bin/doris-sinker -config configs/config.prod.yaml

# 日志级别设为info
# 启用日志采样
# 批次大小优化
```

### Docker模式

```bash
docker run -d \
  --name doris-sinker \
  -v $(pwd)/configs:/app/configs \
  -e DORIS_PASSWORD=your_password \
  -p 9090:9090 \
  doris-sinker:1.0.0
```

### Kubernetes模式

```bash
# 创建Secret
kubectl create secret generic doris-sinker-secret \
  --from-literal=password=your_password

# 部署
kubectl apply -f deployments/kubernetes/
```

## 监控告警

### Prometheus指标

访问 http://localhost:9090/metrics 查看所有指标

**关键指标**：
- `kafka_messages_consumed_total`: 消费速率
- `batch_flush_duration_seconds`: 批次延迟
- `doris_stream_load_total{status="failed"}`: 写入失败数

### Grafana面板

导入 `deployments/grafana/dashboard.json` 到Grafana

**面板包含**：
- 消费速率趋势
- Lag监控
- 批次大小分布
- 写入成功率
- 系统资源使用

### 告警规则

在Prometheus中配置告警：

```yaml
groups:
  - name: doris_sinker
    rules:
      # 写入失败率过高
      - alert: HighFailureRate
        expr: rate(doris_stream_load_total{status="failed"}[5m]) > 0.05
        for: 5m
        annotations:
          summary: "Stream load failure rate > 5%"
      
      # 批次延迟过高
      - alert: HighLatency
        expr: histogram_quantile(0.99, batch_flush_duration_seconds) > 10
        for: 5m
        annotations:
          summary: "P99 batch flush latency > 10s"
```

### pprof性能分析

**CPU分析**：
```bash
# 采样30秒
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# 分析
go tool pprof cpu.prof
```

**内存分析**：
```bash
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**Web界面**：
```bash
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap
```

## 故障排查

### 1. 消费Lag持续增长

**原因**：
- Doris写入慢
- 批次配置不合理
- 网络问题

**排查步骤**：
1. 检查Doris集群负载
2. 查看Stream Load延迟指标
3. 增大批次大小
4. 水平扩展实例

**解决方案**：
```yaml
# 增大批次
batch:
  max_batch_rows: 50000
  max_batch_size: 52428800
```

### 2. Stream Load频繁失败

**原因**：
- Doris集群故障
- 网络不稳定
- 数据格式错误

**排查步骤**：
1. 查看错误日志
2. 检查Doris集群状态
3. 验证数据格式
4. 检查网络连接

**解决方案**：
```bash
# 查看详细错误
grep "stream load failed" /var/log/doris-sinker.log

# 检查Doris
mysql -h 127.0.0.1 -P 9030 -u root -e "SHOW BACKENDS"
```

### 3. 内存占用过高

**原因**：
- 批次过大
- 内存泄漏
- 日志过多

**排查步骤**：
1. 使用pprof分析内存
2. 检查批次配置
3. 启用日志采样

**解决方案**：
```yaml
# 减小批次
batch:
  max_batch_rows: 5000
  max_batch_size: 5242880  # 5MB

# 启用日志采样
log:
  enable_sampling: true
```

### 4. JSON解析错误

**原因**：
- JSON格式不正确
- 字段类型不匹配
- 编码问题

**排查步骤**：
1. 查看错误日志中的字段名
2. 检查JSON样本
3. 验证Schema映射

**解决方案**：
```bash
# 查看解析错误
grep "json parse error" /var/log/doris-sinker.log

# 检查Prometheus指标
curl http://localhost:9090/metrics | grep json_parse_errors
```

## 最佳实践

### 1. 批次大小设置

**经验公式**：
- 行数：根据消息大小，通常 5000-50000
- 字节数：10MB-50MB
- 时间间隔：30-60秒

**示例**：
```yaml
# 小消息（< 1KB）
batch:
  max_batch_rows: 50000
  max_batch_size: 52428800
  max_batch_interval: 60

# 大消息（> 10KB）
batch:
  max_batch_rows: 1000
  max_batch_size: 10485760
  max_batch_interval: 30
```

### 2. 水平扩展

**单实例性能瓶颈时**：

1. 增加Kafka分区数
2. 部署多个实例（相同group_id）
3. 每个实例消费不同分区

```bash
# 部署3个实例
for i in 1 2 3; do
  ./bin/doris-sinker -config configs/config.yaml &
done
```

### 3. 监控告警

**必须监控的指标**：
- 消费Lag
- 写入失败率
- 批次延迟
- 内存使用

**告警阈值建议**：
- Lag > 100万：警告
- 失败率 > 5%：严重
- P99延迟 > 10s：警告
- 内存 > 2GB：警告

### 4. 日志管理

**生产环境配置**：
```yaml
log:
  level: "info"
  output: "both"  # 同时输出到stdout和文件
  file_path: "/var/log/doris-sinker.log"
  format: "json"
  enable_sampling: true  # 启用采样
  max_size: 100
  max_age: 7
  max_backups: 10
```

### 5. 优雅关闭

**正确的停止方式**：
```bash
# 发送SIGTERM信号
kill -TERM <pid>

# 等待30秒让程序优雅关闭
# 程序会：
# 1. 停止消费新消息
# 2. 处理完当前批次
# 3. 提交offset
# 4. 关闭连接
```

### 6. 数据一致性

**At-Least-Once语义**：
- 手动提交offset
- 批次成功后才提交
- 可能有重复数据

**去重方案**：
- 使用Doris的UNIQUE KEY模型
- 设置主键（如id）
- Doris自动去重

### 7. 性能调优

**CPU密集型**：
- 增加goroutine数量
- 使用更快的JSON库（sonic）
- 减少日志输出

**IO密集型**：
- 增大批次大小
- 使用连接池
- 启用HTTP Keep-Alive

**内存密集型**：
- 使用对象池
- 及时释放资源
- 启用日志采样

## 常见问题

### Q1: 如何验证数据是否写入成功？

```sql
-- 在Doris中查询
SELECT COUNT(*) FROM test_db.tb_event;

-- 查看最新数据
SELECT * FROM test_db.tb_event ORDER BY id DESC LIMIT 10;
```

### Q2: 如何处理Schema变更？

**自动模式**：
1. 在Doris中修改表结构
2. 重启doris-sinker
3. 程序会自动获取新Schema

**手动模式**：
1. 修改配置文件中的columns
2. 重启程序

### Q3: 如何提高吞吐量？

1. 增大批次大小
2. 增加Kafka分区数
3. 水平扩展实例
4. 优化Doris集群

### Q4: 如何保证数据不丢失？

1. 使用手动提交offset
2. 批次成功后才提交
3. 配置重试机制
4. 监控消费Lag

### Q5: 如何处理敏感信息？

```bash
# 使用环境变量
export DORIS_PASSWORD=your_password

# 配置文件中不写密码
doris:
  password: ""  # 从环境变量读取
```

## 参考资料

- [Doris Stream Load文档](https://doris.apache.org/docs/data-operate/import/import-way/stream-load-manual)
- [franz-go文档](https://github.com/twmb/franz-go)
- [sonic文档](https://github.com/bytedance/sonic)
- [Prometheus最佳实践](https://prometheus.io/docs/practices/)
