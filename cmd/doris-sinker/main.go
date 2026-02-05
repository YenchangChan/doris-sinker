package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/doris-sinker/doris-sinker/internal/batcher"
	"github.com/doris-sinker/doris-sinker/internal/config"
	"github.com/doris-sinker/doris-sinker/internal/consumer"
	"github.com/doris-sinker/doris-sinker/internal/pipeline"
	"github.com/doris-sinker/doris-sinker/internal/schema"
	"github.com/doris-sinker/doris-sinker/internal/server"
	"github.com/doris-sinker/doris-sinker/internal/signal"
	"github.com/doris-sinker/doris-sinker/internal/writer"
	"github.com/doris-sinker/doris-sinker/pkg/logger"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "configs/config.yaml", "config file path")
	version    = "1.0.0"
)

func main() {
	flag.Parse()

	fmt.Printf("Doris-Sinker v%s\n", version)
	fmt.Printf("Loading config from: %s\n", *configPath)

	// 1. 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	if err := logger.Init(cfg.Log); err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("doris-sinker starting",
		zap.String("version", version),
		zap.String("config", cfg.String()),
	)

	// 3. 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. 初始化Schema
	var schemaObj *schema.Schema
	if cfg.Schema.Mode == "auto" {
		logger.Info("fetching schema from doris")
		fetcher := schema.NewFetcher(cfg.Doris)
		schemaObj, err = fetcher.FetchFromDoris(ctx)
		if err != nil {
			logger.Fatal("failed to fetch schema from doris", zap.Error(err))
		}
	} else {
		logger.Info("loading schema from config")
		schemaObj, err = schema.FetchFromConfig(cfg.Schema.Manual)
		if err != nil {
			logger.Fatal("failed to load schema from config", zap.Error(err))
		}
	}

	logger.Info("schema initialized",
		zap.Int("columns", schemaObj.ColumnCount()),
	)

	// 5. 创建Mapper
	mapper := schema.NewMapper(schemaObj)

	// 6. 创建Consumer
	kafkaConsumer, err := consumer.NewFranzConsumer(cfg.Kafka)
	if err != nil {
		logger.Fatal("failed to create kafka consumer", zap.Error(err))
	}

	// 7. 创建Batcher
	batcherObj := batcher.NewMemoryBatcher(cfg.Batch)

	// 8. 创建Writer
	columnNames := make([]string, len(schemaObj.Columns))
	for i, col := range schemaObj.Columns {
		columnNames[i] = col.Name
	}
	writerObj := writer.NewStreamLoadWriter(cfg.Doris, columnNames)

	// 9. 创建Pipeline
	pipelineObj := pipeline.NewPipeline(kafkaConsumer, mapper, batcherObj, writerObj, cfg.Batch.FlushWorkerCount)

	// 10. 启动HTTP服务器
	srv := server.NewServer(*cfg)
	if err := srv.Start(); err != nil {
		logger.Fatal("failed to start server", zap.Error(err))
	}

	// 11. 启动Pipeline
	if err := pipelineObj.Start(ctx); err != nil {
		logger.Fatal("failed to start pipeline", zap.Error(err))
	}

	logger.Info("doris-sinker started successfully")

	// 12. 等待关闭信号
	signal.WaitForShutdown(ctx, cancel)

	logger.Info("shutting down gracefully...")

	// 13. 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 停止Pipeline
	if err := pipelineObj.Stop(shutdownCtx); err != nil {
		logger.Error("failed to stop pipeline", zap.Error(err))
	}

	// 停止HTTP服务器
	if err := srv.Stop(shutdownCtx); err != nil {
		logger.Error("failed to stop server", zap.Error(err))
	}

	logger.Info("doris-sinker stopped")
}
