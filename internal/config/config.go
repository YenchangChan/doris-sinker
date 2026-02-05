package config

import (
	"github.com/doris-sinker/doris-sinker/pkg/logger"
)

// Config 全局配置
type Config struct {
	Kafka   KafkaConfig   `yaml:"kafka"`
	Doris   DorisConfig   `yaml:"doris"`
	Batch   BatchConfig   `yaml:"batch"`
	Schema  SchemaConfig  `yaml:"schema"`
	Log     logger.Config `yaml:"log"`
	Metrics MetricsConfig `yaml:"metrics"`
	Pprof   PprofConfig   `yaml:"pprof"`
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers           []string `yaml:"brokers"`
	Topic             string   `yaml:"topic"`
	GroupID           string   `yaml:"group_id"`
	FromEarliest      bool     `yaml:"from_earliest"`
	MaxFetchRecords   int      `yaml:"max_fetch_records"`
	MaxFetchBytes     int      `yaml:"max_fetch_bytes"`
	SessionTimeout    int      `yaml:"session_timeout"`
	HeartbeatInterval int      `yaml:"heartbeat_interval"`
}

// DorisConfig Doris配置
type DorisConfig struct {
	FEHosts    []string `yaml:"fe_hosts"`
	QueryPort  int      `yaml:"query_port"`
	Database   string   `yaml:"database"`
	Table      string   `yaml:"table"`
	User       string   `yaml:"user"`
	Password   string   `yaml:"password"`
	Timeout    int      `yaml:"timeout"`
	MaxRetries int      `yaml:"max_retries"`
}

// BatchConfig 批次配置
type BatchConfig struct {
	MaxBatchRows     int `yaml:"max_batch_rows"`
	MaxBatchSize     int `yaml:"max_batch_size"`
	MaxBatchInterval int `yaml:"max_batch_interval"`
	FlushWorkerCount int `yaml:"flush_worker_count"` // 并发flush工作器数量
}

// SchemaConfig Schema配置
type SchemaConfig struct {
	Mode   string             `yaml:"mode"` // auto, manual
	Auto   AutoSchemaConfig   `yaml:"auto"`
	Manual ManualSchemaConfig `yaml:"manual"`
}

// AutoSchemaConfig 自动Schema配置
type AutoSchemaConfig struct {
	RefreshInterval int  `yaml:"refresh_interval"`
	ValidateOnStart bool `yaml:"validate_on_start"`
}

// ManualSchemaConfig 手动Schema配置
type ManualSchemaConfig struct {
	Columns []ColumnConfig `yaml:"columns"`
}

// ColumnConfig 列配置
type ColumnConfig struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// MetricsConfig 监控配置
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// PprofConfig pprof配置
type PprofConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Kafka: KafkaConfig{
			Brokers:           []string{"localhost:9092"},
			Topic:             "event_topic",
			GroupID:           "doris-sinker-group",
			FromEarliest:      true,
			MaxFetchRecords:   1000,
			MaxFetchBytes:     1048576, // 1MB
			SessionTimeout:    30,
			HeartbeatInterval: 3,
		},
		Doris: DorisConfig{
			FEHosts:    []string{"127.0.0.1:8030"},
			QueryPort:  9030,
			Database:   "test_db",
			Table:      "tb_event",
			User:       "root",
			Password:   "",
			Timeout:    600,
			MaxRetries: 3,
		},
		Batch: BatchConfig{
			MaxBatchRows:     10000,
			MaxBatchSize:     10485760, // 10MB
			MaxBatchInterval: 30,
			FlushWorkerCount: 4, // 默认4个并发flush工作器
		},
		Schema: SchemaConfig{
			Mode: "auto",
			Auto: AutoSchemaConfig{
				RefreshInterval: 0,
				ValidateOnStart: true,
			},
		},
		Log: logger.Config{
			Level:          "info",
			Output:         "stdout",
			Format:         "json",
			EnableSampling: true,
			MaxSize:        100,
			MaxAge:         7,
			MaxBackups:     10,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Port:    9090,
			Path:    "/metrics",
		},
		Pprof: PprofConfig{
			Enabled: true,
			Port:    6060,
		},
	}
}
