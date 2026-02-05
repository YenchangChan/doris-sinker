package consumer

import (
	"context"
)

// Message Kafka消息
type Message struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Timestamp int64
}

// Consumer Kafka消费者接口
type Consumer interface {
	Start(ctx context.Context) error
	Messages() <-chan *Message
	Commit(ctx context.Context, offset int64) error
	Close() error
}
