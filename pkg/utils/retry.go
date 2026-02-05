package utils

import (
	"context"
	"time"
)

// RetryFunc 重试函数类型
type RetryFunc func() error

// Retry 重试执行函数
func Retry(ctx context.Context, maxRetries int, initialBackoff time.Duration, fn RetryFunc) error {
	var err error
	backoff := initialBackoff

	for i := 0; i <= maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		// 最后一次重试失败，直接返回
		if i == maxRetries {
			return err
		}

		// 等待后重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// 指数退避
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}

	return err
}
