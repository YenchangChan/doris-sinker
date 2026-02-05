package pipeline

import (
	"context"
	"time"

	"github.com/doris-sinker/doris-sinker/internal/batcher"
	"github.com/doris-sinker/doris-sinker/internal/consumer"
	"github.com/doris-sinker/doris-sinker/internal/metrics"
	"github.com/doris-sinker/doris-sinker/internal/schema"
	"github.com/doris-sinker/doris-sinker/internal/writer"
	"github.com/doris-sinker/doris-sinker/pkg/errors"
	"github.com/doris-sinker/doris-sinker/pkg/logger"
	"go.uber.org/zap"
)

// Pipeline 数据处理流水线
type Pipeline struct {
	consumer    consumer.Consumer
	mapper      *schema.Mapper
	batcher     batcher.Batcher
	writer      writer.Writer
	flushWorker *FlushWorker
}

// NewPipeline 创建流水线
func NewPipeline(
	consumer consumer.Consumer,
	mapper *schema.Mapper,
	batcher batcher.Batcher,
	writer writer.Writer,
	flushWorkerCount int,
) *Pipeline {
	p := &Pipeline{
		consumer: consumer,
		mapper:   mapper,
		batcher:  batcher,
		writer:   writer,
	}

	// 创建并发flush工作器
	if flushWorkerCount > 1 {
		p.flushWorker = NewFlushWorker(p, flushWorkerCount)
	}

	return p
}

// Start 启动流水线
func (p *Pipeline) Start(ctx context.Context) error {
	logger.Info("starting pipeline")

	// 启动消费者
	if err := p.consumer.Start(ctx); err != nil {
		return errors.Wrap(errors.ErrCodeKafkaConsume, "failed to start consumer", err)
	}

	// 启动批次管理器
	go p.batcher.Start(ctx)

	// 启动flush工作器（如果启用）
	if p.flushWorker != nil {
		p.flushWorker.Start(ctx)
	}

	// 启动消息处理循环
	go p.processMessages(ctx)

	// 启动批次刷新循环
	go p.flushBatches(ctx)

	logger.Info("pipeline started")
	return nil
}

// processMessages 处理消息
func (p *Pipeline) processMessages(ctx context.Context) {
	msgChan := p.consumer.Messages()

	for {
		select {
		case <-ctx.Done():
			logger.Info("message processing stopped")
			return

		case msg, ok := <-msgChan:
			if !ok {
				logger.Info("message channel closed")
				return
			}

			// 处理消息
			if err := p.processMessage(ctx, msg); err != nil {
				logger.Error("failed to process message",
					zap.Error(err),
					zap.Int64("offset", msg.Offset),
				)
				metrics.KafkaConsumeErrors.WithLabelValues(msg.Topic, "process_error").Inc()
			} else {
				// 更新指标
				metrics.KafkaMessagesConsumed.WithLabelValues(msg.Topic).Inc()
				metrics.KafkaBytesConsumed.WithLabelValues(msg.Topic).Add(float64(len(msg.Value)))
			}
		}
	}
}

// processMessage 处理单条消息
func (p *Pipeline) processMessage(ctx context.Context, msg *consumer.Message) error {
	// JSON映射为Row
	row, err := p.mapper.MapJSONToRow(msg.Value)
	if err != nil {
		return errors.Wrap(errors.ErrCodeFieldMapping, "failed to map json to row", err)
	}

	// 添加到批次
	if err := p.batcher.Add(row); err != nil {
		return errors.Wrap(errors.ErrCodeBatchFull, "failed to add to batch", err)
	}

	// 更新批次指标
	metrics.BatchCurrentRows.Set(float64(p.batcher.Size()))

	return nil
}

// flushBatches 刷新批次
func (p *Pipeline) flushBatches(ctx context.Context) {
	// 获取批次管理器的flush channel
	memBatcher, ok := p.batcher.(*batcher.MemoryBatcher)
	if !ok {
		logger.Error("batcher is not MemoryBatcher")
		return
	}

	flushChan := memBatcher.FlushChan()

	for {
		select {
		case <-ctx.Done():
			// 最后flush一次
			p.doFlush(ctx)
			logger.Info("batch flushing stopped")
			return

		case <-flushChan:
			p.doFlush(ctx)
		}
	}
}

// doFlush 执行flush
func (p *Pipeline) doFlush(ctx context.Context) {
	startTime := time.Now()

	// 获取批次
	batch, err := p.batcher.Flush()
	if err != nil {
		logger.Error("failed to flush batch", zap.Error(err))
		metrics.BatchFlushTotal.WithLabelValues("error").Inc()
		return
	}

	if len(batch) == 0 {
		return
	}

	// 计算批次大小
	batchSize := 0
	for _, row := range batch {
		for _, val := range row {
			if str, ok := val.(string); ok {
				batchSize += len(str)
			} else {
				batchSize += 8
			}
		}
	}

	logger.Info("flushing batch",
		zap.Int("rows", len(batch)),
		zap.Int("size_bytes", batchSize),
	)

	// 如果启用并发flush，提交给worker处理
	if p.flushWorker != nil {
		p.flushWorker.Submit(batch)
		// 并发模式下，指标由worker更新，这里只记录提交
		metrics.BatchFlushTotal.WithLabelValues("submitted").Inc()
		return
	}

	// 同步flush
	if err := p.writer.Write(ctx, batch); err != nil {
		logger.Error("failed to write to doris", zap.Error(err))
		metrics.BatchFlushTotal.WithLabelValues("failed").Inc()
		metrics.DorisStreamLoadTotal.WithLabelValues("failed").Inc()
		metrics.DorisStreamLoadErrors.WithLabelValues("write_error").Inc()
		return
	}

	// 更新指标
	duration := time.Since(startTime)
	metrics.BatchFlushTotal.WithLabelValues("success").Inc()
	metrics.BatchSizeRows.Observe(float64(len(batch)))
	metrics.BatchSizeBytes.Observe(float64(batchSize))
	metrics.BatchFlushDuration.Observe(duration.Seconds())
	metrics.DorisStreamLoadTotal.WithLabelValues("success").Inc()
	metrics.DorisRowsLoaded.Add(float64(len(batch)))
	metrics.DorisStreamLoadDuration.Observe(duration.Seconds())

	// 重置批次指标
	metrics.BatchCurrentRows.Set(0)
	metrics.BatchCurrentBytes.Set(0)

	logger.Info("batch flushed successfully",
		zap.Int("rows", len(batch)),
		zap.Duration("duration", duration),
	)
}

// Stop 停止流水线
func (p *Pipeline) Stop(ctx context.Context) error {
	logger.Info("stopping pipeline")

	// 关闭flush工作器（如果启用）
	if p.flushWorker != nil {
		p.flushWorker.Stop()
	}

	// 关闭消费者
	if err := p.consumer.Close(); err != nil {
		logger.Error("failed to close consumer", zap.Error(err))
	}

	// 关闭批次管理器
	if err := p.batcher.Close(); err != nil {
		logger.Error("failed to close batcher", zap.Error(err))
	}

	// 关闭写入器
	if err := p.writer.Close(); err != nil {
		logger.Error("failed to close writer", zap.Error(err))
	}

	logger.Info("pipeline stopped")
	return nil
}
