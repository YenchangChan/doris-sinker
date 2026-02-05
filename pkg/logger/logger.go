package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var globalLogger *zap.Logger

// Config 日志配置
type Config struct {
	Level          string `yaml:"level"`           // debug, info, warn, error
	Output         string `yaml:"output"`          // stdout, file, both
	FilePath       string `yaml:"file_path"`       // 日志文件路径
	Format         string `yaml:"format"`          // json, console
	EnableSampling bool   `yaml:"enable_sampling"` // 是否启用采样
	MaxSize        int    `yaml:"max_size"`        // 日志文件最大大小(MB)
	MaxAge         int    `yaml:"max_age"`         // 日志文件最大保留天数
	MaxBackups     int    `yaml:"max_backups"`     // 日志文件最大备份数
}

// Init 初始化日志
func Init(cfg Config) error {
	// 解析日志级别
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// 配置编码器
	var encoderConfig zapcore.EncoderConfig
	if cfg.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// 创建编码器
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 配置输出
	var cores []zapcore.Core

	// 标准输出
	if cfg.Output == "stdout" || cfg.Output == "both" {
		cores = append(cores, zapcore.NewCore(
			encoder,
			zapcore.AddSync(os.Stdout),
			level,
		))
	}

	// 文件输出
	if cfg.Output == "file" || cfg.Output == "both" {
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge,
			MaxBackups: cfg.MaxBackups,
			Compress:   true,
		}
		cores = append(cores, zapcore.NewCore(
			encoder,
			zapcore.AddSync(fileWriter),
			level,
		))
	}

	// 创建logger
	core := zapcore.NewTee(cores...)

	// 采样配置
	if cfg.EnableSampling {
		core = zapcore.NewSamplerWithOptions(
			core,
			time.Second,
			100,  // 每秒前100条日志全部记录
			1000, // 之后每1000条记录1条
		)
	}

	globalLogger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return nil
}

// Get 获取全局logger
func Get() *zap.Logger {
	if globalLogger == nil {
		// 如果未初始化，使用默认配置
		globalLogger, _ = zap.NewProduction()
	}
	return globalLogger
}

// Sync 同步日志
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// With 创建带字段的logger
func With(fields ...zap.Field) *zap.Logger {
	return Get().With(fields...)
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}
