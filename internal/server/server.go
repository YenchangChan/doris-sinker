package server

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/doris-sinker/doris-sinker/internal/config"
	"github.com/doris-sinker/doris-sinker/pkg/logger"
	"go.uber.org/zap"
)

// Server HTTP服务器
type Server struct {
	metricsServer *http.Server
	pprofServer   *http.Server
}

// NewServer 创建HTTP服务器
func NewServer(cfg config.Config) *Server {
	s := &Server{}

	// Metrics服务器
	if cfg.Metrics.Enabled {
		mux := http.NewServeMux()
		mux.Handle(cfg.Metrics.Path, promhttp.Handler())
		mux.HandleFunc("/health", healthHandler)
		mux.HandleFunc("/ready", readyHandler)

		s.metricsServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Metrics.Port),
			Handler: mux,
		}
	}

	// Pprof服务器
	if cfg.Pprof.Enabled {
		s.pprofServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Pprof.Port),
			Handler: http.DefaultServeMux, // pprof已自动注册到DefaultServeMux
		}
	}

	return s
}

// Start 启动服务器
func (s *Server) Start() error {
	// 启动Metrics服务器
	if s.metricsServer != nil {
		go func() {
			logger.Info("starting metrics server", zap.String("addr", s.metricsServer.Addr))
			if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("metrics server error", zap.Error(err))
			}
		}()
	}

	// 启动Pprof服务器
	if s.pprofServer != nil {
		go func() {
			logger.Info("starting pprof server", zap.String("addr", s.pprofServer.Addr))
			if err := s.pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("pprof server error", zap.Error(err))
			}
		}()
	}

	return nil
}

// Stop 停止服务器
func (s *Server) Stop(ctx context.Context) error {
	if s.metricsServer != nil {
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown metrics server", zap.Error(err))
		}
	}

	if s.pprofServer != nil {
		if err := s.pprofServer.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown pprof server", zap.Error(err))
		}
	}

	return nil
}

// healthHandler 健康检查
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// readyHandler 就绪检查
func readyHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: 检查依赖服务（Kafka、Doris）是否就绪
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}
