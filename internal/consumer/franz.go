package consumer

import (
	"context"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/doris-sinker/doris-sinker/internal/config"
	"github.com/doris-sinker/doris-sinker/pkg/errors"
	"github.com/doris-sinker/doris-sinker/pkg/logger"
	"go.uber.org/zap"
)

// FranzConsumer franz-go消费者实现
type FranzConsumer struct {
	cfg        config.KafkaConfig
	client     *kgo.Client
	msgChan    chan *Message
	commitChan chan int64
	closed     bool
}

// NewFranzConsumer 创建franz-go消费者
func NewFranzConsumer(cfg config.KafkaConfig) (*FranzConsumer, error) {
	// 配置选项
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.ConsumeTopics(cfg.Topic),
		kgo.FetchMaxBytes(int32(cfg.MaxFetchBytes)),
		kgo.FetchMinBytes(1),
		kgo.SessionTimeout(time.Duration(cfg.SessionTimeout) * time.Second),
		kgo.HeartbeatInterval(time.Duration(cfg.HeartbeatInterval) * time.Second),
		kgo.DisableAutoCommit(), // 手动提交offset
	}

	// 设置消费起始位置
	if cfg.FromEarliest {
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	} else {
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
	}

	// 创建客户端
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeKafkaConnect, "failed to create kafka client", err)
	}

	logger.Info("kafka consumer created",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("topic", cfg.Topic),
		zap.String("group_id", cfg.GroupID),
	)

	return &FranzConsumer{
		cfg:        cfg,
		client:     client,
		msgChan:    make(chan *Message, 1000),
		commitChan: make(chan int64, 100),
		closed:     false,
	}, nil
}

// Start 启动消费
func (c *FranzConsumer) Start(ctx context.Context) error {
	// 测试连接
	if err := c.client.Ping(ctx); err != nil {
		return errors.Wrap(errors.ErrCodeKafkaConnect, "failed to ping kafka", err)
	}

	logger.Info("kafka consumer started")

	// 启动消费循环
	go c.consumeLoop(ctx)

	// 启动提交循环
	go c.commitLoop(ctx)

	return nil
}

// consumeLoop 消费循环
func (c *FranzConsumer) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			logger.Info("consume loop stopped")
			return
		default:
		}

		// 拉取消息
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			logger.Info("kafka client closed")
			return
		}

		// 处理错误
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, err := range errs {
				logger.Error("fetch error",
					zap.String("topic", err.Topic),
					zap.Int32("partition", err.Partition),
					zap.Error(err.Err),
				)
			}
			continue
		}

		// 处理消息
		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()

			msg := &Message{
				Topic:     record.Topic,
				Partition: record.Partition,
				Offset:    record.Offset,
				Key:       record.Key,
				Value:     record.Value,
				Timestamp: record.Timestamp.UnixMilli(),
			}

			select {
			case c.msgChan <- msg:
			case <-ctx.Done():
				return
			}
		}
	}
}

// commitLoop 提交循环
func (c *FranzConsumer) commitLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastOffset int64 = -1

	for {
		select {
		case <-ctx.Done():
			// 最后提交一次
			if lastOffset >= 0 {
				c.doCommit(ctx, lastOffset)
			}
			logger.Info("commit loop stopped")
			return

		case offset := <-c.commitChan:
			lastOffset = offset

		case <-ticker.C:
			if lastOffset >= 0 {
				c.doCommit(ctx, lastOffset)
				lastOffset = -1
			}
		}
	}
}

// doCommit 执行提交
func (c *FranzConsumer) doCommit(ctx context.Context, offset int64) {
	if err := c.client.CommitMarkedOffsets(ctx); err != nil {
		logger.Error("failed to commit offset",
			zap.Int64("offset", offset),
			zap.Error(err),
		)
	} else {
		logger.Info("offset committed",
			zap.Int64("offset", offset),
		)
	}
}

// Messages 返回消息channel
func (c *FranzConsumer) Messages() <-chan *Message {
	return c.msgChan
}

// Commit 提交offset
func (c *FranzConsumer) Commit(ctx context.Context, offset int64) error {
	select {
	case c.commitChan <- offset:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close 关闭消费者
func (c *FranzConsumer) Close() error {
	if c.closed {
		return nil
	}

	c.closed = true
	close(c.msgChan)
	close(c.commitChan)

	c.client.Close()
	logger.Info("kafka consumer closed")

	return nil
}
