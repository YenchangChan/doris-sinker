package batcher

import (
	"context"
	"sync"
	"time"

	"github.com/doris-sinker/doris-sinker/internal/config"
	"github.com/doris-sinker/doris-sinker/pkg/logger"
	"go.uber.org/zap"
)

// Batcher 批次管理器接口
type Batcher interface {
	Add(row []interface{}) error
	Flush() ([][]interface{}, error)
	ShouldFlush() bool
	Size() int
	Start(ctx context.Context)
	Close() error
}

// MemoryBatcher 内存批次实现
type MemoryBatcher struct {
	cfg           config.BatchConfig
	rows          [][]interface{}
	currentSize   int
	mu            sync.Mutex
	flushChan     chan struct{}
	lastFlushTime time.Time
	closed        bool
}

// NewMemoryBatcher 创建内存批次管理器
func NewMemoryBatcher(cfg config.BatchConfig) *MemoryBatcher {
	return &MemoryBatcher{
		cfg:           cfg,
		rows:          make([][]interface{}, 0, cfg.MaxBatchRows),
		currentSize:   0,
		flushChan:     make(chan struct{}, 1),
		lastFlushTime: time.Now(),
		closed:        false,
	}
}

// Add 添加行到批次
func (b *MemoryBatcher) Add(row []interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	// 估算行大小（粗略估算）
	rowSize := estimateRowSize(row)

	b.rows = append(b.rows, row)
	b.currentSize += rowSize

	// 检查是否需要flush
	if b.shouldFlushLocked() {
		select {
		case b.flushChan <- struct{}{}:
		default:
			// channel已满，说明有flush正在进行
		}
	}

	return nil
}

// Flush 刷新批次
func (b *MemoryBatcher) Flush() ([][]interface{}, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.rows) == 0 {
		return nil, nil
	}

	// 复制数据
	batch := make([][]interface{}, len(b.rows))
	copy(batch, b.rows)

	// 重置
	b.rows = b.rows[:0]
	b.currentSize = 0
	b.lastFlushTime = time.Now()

	logger.Debug("batch flushed",
		zap.Int("rows", len(batch)),
		zap.Int("size_bytes", b.currentSize),
	)

	return batch, nil
}

// ShouldFlush 检查是否应该flush
func (b *MemoryBatcher) ShouldFlush() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.shouldFlushLocked()
}

// shouldFlushLocked 检查是否应该flush（需要持有锁）
func (b *MemoryBatcher) shouldFlushLocked() bool {
	// 检查行数
	if len(b.rows) >= b.cfg.MaxBatchRows {
		return true
	}

	// 检查大小
	if b.currentSize >= b.cfg.MaxBatchSize {
		return true
	}

	// 检查时间间隔
	if time.Since(b.lastFlushTime) >= time.Duration(b.cfg.MaxBatchInterval)*time.Second {
		return len(b.rows) > 0
	}

	return false
}

// Size 返回当前批次大小
func (b *MemoryBatcher) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.rows)
}

// Start 启动定时flush
func (b *MemoryBatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if b.ShouldFlush() {
				select {
				case b.flushChan <- struct{}{}:
				default:
				}
			}
		}
	}
}

// FlushChan 返回flush通知channel
func (b *MemoryBatcher) FlushChan() <-chan struct{} {
	return b.flushChan
}

// Close 关闭批次管理器
func (b *MemoryBatcher) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	close(b.flushChan)

	return nil
}

// estimateRowSize 估算行大小
func estimateRowSize(row []interface{}) int {
	size := 0
	for _, v := range row {
		switch val := v.(type) {
		case string:
			size += len(val)
		case int, int64, float64, bool:
			size += 8
		default:
			size += 8
		}
	}
	return size
}
