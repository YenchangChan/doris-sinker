package config

import (
	"fmt"

	"github.com/doris-sinker/doris-sinker/pkg/errors"
)

// Validate 验证配置
func Validate(cfg *Config) error {
	// 验证Kafka配置
	if len(cfg.Kafka.Brokers) == 0 {
		return errors.New(errors.ErrCodeConfigValidate, "kafka.brokers is required")
	}
	if cfg.Kafka.Topic == "" {
		return errors.New(errors.ErrCodeConfigValidate, "kafka.topic is required")
	}
	if cfg.Kafka.GroupID == "" {
		return errors.New(errors.ErrCodeConfigValidate, "kafka.group_id is required")
	}
	if cfg.Kafka.MaxFetchRecords <= 0 {
		return errors.New(errors.ErrCodeConfigValidate, "kafka.max_fetch_records must be positive")
	}
	if cfg.Kafka.MaxFetchBytes <= 0 {
		return errors.New(errors.ErrCodeConfigValidate, "kafka.max_fetch_bytes must be positive")
	}

	// 验证Doris配置
	if len(cfg.Doris.FEHosts) == 0 {
		return errors.New(errors.ErrCodeConfigValidate, "doris.fe_hosts is required")
	}
	if cfg.Doris.Database == "" {
		return errors.New(errors.ErrCodeConfigValidate, "doris.database is required")
	}
	if cfg.Doris.Table == "" {
		return errors.New(errors.ErrCodeConfigValidate, "doris.table is required")
	}
	if cfg.Doris.User == "" {
		return errors.New(errors.ErrCodeConfigValidate, "doris.user is required")
	}
	if cfg.Doris.QueryPort <= 0 {
		cfg.Doris.QueryPort = 9030 // 默认值
	}

	// 验证批次配置
	if cfg.Batch.MaxBatchRows <= 0 {
		return errors.New(errors.ErrCodeConfigValidate, "batch.max_batch_rows must be positive")
	}
	if cfg.Batch.MaxBatchSize <= 0 {
		return errors.New(errors.ErrCodeConfigValidate, "batch.max_batch_size must be positive")
	}
	if cfg.Batch.MaxBatchInterval <= 0 {
		return errors.New(errors.ErrCodeConfigValidate, "batch.max_batch_interval must be positive")
	}

	// 验证Schema配置
	if cfg.Schema.Mode != "auto" && cfg.Schema.Mode != "manual" {
		return errors.New(errors.ErrCodeConfigValidate, "schema.mode must be 'auto' or 'manual'")
	}
	if cfg.Schema.Mode == "manual" && len(cfg.Schema.Manual.Columns) == 0 {
		return errors.New(errors.ErrCodeConfigValidate, "schema.manual.columns is required when mode is 'manual'")
	}

	// 验证监控配置
	if cfg.Metrics.Enabled && cfg.Metrics.Port <= 0 {
		return errors.New(errors.ErrCodeConfigValidate, "metrics.port must be positive when enabled")
	}

	// 验证pprof配置
	if cfg.Pprof.Enabled && cfg.Pprof.Port <= 0 {
		return errors.New(errors.ErrCodeConfigValidate, "pprof.port must be positive when enabled")
	}

	// 验证端口冲突
	if cfg.Metrics.Enabled && cfg.Pprof.Enabled && cfg.Metrics.Port == cfg.Pprof.Port {
		return errors.New(errors.ErrCodeConfigValidate, "metrics.port and pprof.port cannot be the same")
	}

	return nil
}

// String 返回配置的字符串表示（隐藏敏感信息）
func (c *Config) String() string {
	return fmt.Sprintf("Config{Kafka: %v, Doris: %s, Batch: %v, Schema: %s}",
		c.Kafka.Brokers,
		c.Doris.Database+"."+c.Doris.Table,
		c.Batch.MaxBatchRows,
		c.Schema.Mode,
	)
}
