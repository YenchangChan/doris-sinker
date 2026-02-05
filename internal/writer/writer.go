package writer

import (
	"context"
)

// Writer Doris写入器接口
type Writer interface {
	Write(ctx context.Context, batch [][]interface{}) error
	Close() error
}
