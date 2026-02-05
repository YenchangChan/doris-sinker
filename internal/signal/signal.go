package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/doris-sinker/doris-sinker/pkg/logger"
	"go.uber.org/zap"
)

// WaitForShutdown 等待关闭信号
func WaitForShutdown(ctx context.Context, cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	case <-ctx.Done():
		logger.Info("context cancelled")
	}
}
