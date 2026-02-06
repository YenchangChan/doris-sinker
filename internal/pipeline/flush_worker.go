package pipeline

import (
	"context"
	"sync"
	"time"

	"github.com/doris-sinker/doris-sinker/pkg/logger"
	"go.uber.org/zap"
)

// FlushWorker 并发flush工作器
type FlushWorker struct {
	pipeline    *Pipeline
	workerCount int
	flushChan   chan flushTask
	wg          sync.WaitGroup
	ctx         context.Context
}

type flushTask struct {
	batch [][]interface{}
}

// NewFlushWorker 创建flush工作器
func NewFlushWorker(pipeline *Pipeline, workerCount int) *FlushWorker {
	return &FlushWorker{
		pipeline:    pipeline,
		workerCount: workerCount,
		flushChan:   make(chan flushTask, workerCount*2),
	}
}

// Start 启动工作器
func (w *FlushWorker) Start(ctx context.Context) {
	w.ctx = ctx
	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}
	logger.Info("flush workers started", zap.Int("count", w.workerCount))
}

// worker flush工作协程
func (w *FlushWorker) worker(id int) {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			logger.Info("flush worker stopped", zap.Int("worker_id", id))
			return
		case task := <-w.flushChan:
			w.doFlush(task.batch, id)
		}
	}
}

// Submit 提交flush任务
func (w *FlushWorker) Submit(batch [][]interface{}) {
	task := flushTask{batch: batch}

	// 阻塞等待，确保数据不丢失
	w.flushChan <- task

	logger.Debug("flush task submitted", zap.Int("batch_size", len(batch)))
}

// doFlush 执行flush
func (w *FlushWorker) doFlush(batch [][]interface{}, workerID int) {
	if len(batch) == 0 {
		return
	}

	logger.Debug("flush worker processing batch",
		zap.Int("worker_id", workerID),
		zap.Int("rows", len(batch)),
	)

	// 复用pipeline的doFlush逻辑，但去掉指标更新（避免并发问题）
	startTime := time.Now()

	// 写入Doris
	if err := w.pipeline.writer.Write(w.ctx, batch); err != nil {
		logger.Error("flush worker failed to write to doris",
			zap.Int("worker_id", workerID),
			zap.Error(err),
		)
		return
	}

	duration := time.Since(startTime)
	logger.Info("flush worker completed",
		zap.Int("worker_id", workerID),
		zap.Int("rows", len(batch)),
		zap.Duration("duration", duration),
	)
}

// Stop 停止工作器
func (w *FlushWorker) Stop() {
	close(w.flushChan)
	w.wg.Wait()
	logger.Info("flush workers stopped")
}
