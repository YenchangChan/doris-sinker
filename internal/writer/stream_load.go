package writer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/doris-sinker/doris-sinker/internal/config"
	"github.com/doris-sinker/doris-sinker/pkg/errors"
	"github.com/doris-sinker/doris-sinker/pkg/logger"
	"github.com/doris-sinker/doris-sinker/pkg/pool"
	"go.uber.org/zap"
)

// StreamLoadWriter Stream Load写入器
type StreamLoadWriter struct {
	cfg        config.DorisConfig
	httpClient *http.Client
	authHeader string
	columns    []string
}

// NewStreamLoadWriter 创建Stream Load写入器
func NewStreamLoadWriter(cfg config.DorisConfig, columns []string) *StreamLoadWriter {
	// 创建HTTP客户端，优化连接池以支持并发请求
	client := &http.Client{
		Timeout: time.Duration(cfg.Timeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100, // 增加最大空闲连接数
			MaxIdleConnsPerHost: 50,  // 每个主机最大空闲连接数
			MaxConnsPerHost:     100, // 每个主机最大连接数
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false, // 启用压缩
			DisableKeepAlives:   false, // 启用keep-alive
			ForceAttemptHTTP2:   false, // Doris使用HTTP/1.1
			// 增加缓冲区大小以提高大文件传输性能
			WriteBufferSize: 32 * 1024, // 32KB写缓冲
			ReadBufferSize:  32 * 1024, // 32KB读缓冲
		},
	}

	// 构建Basic Auth header
	auth := base64.StdEncoding.EncodeToString([]byte(cfg.User + ":" + cfg.Password))
	authHeader := "Basic " + auth

	return &StreamLoadWriter{
		cfg:        cfg,
		httpClient: client,
		authHeader: authHeader,
		columns:    columns,
	}
}

// Write 写入数据到Doris
func (w *StreamLoadWriter) Write(ctx context.Context, batch [][]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	startTime := time.Now()

	// 构建JSON Lines数据
	jsonBuf := BuildJSONLines(batch, w.columns)
	defer pool.PutBuffer(jsonBuf)

	// 重试写入
	var lastErr error
	for i := 0; i <= w.cfg.MaxRetries; i++ {
		err := w.doStreamLoad(ctx, jsonBuf.Bytes(), len(batch))
		if err == nil {
			duration := time.Since(startTime)
			logger.Info("stream load success",
				zap.Int("rows", len(batch)),
				zap.Int("size_bytes", jsonBuf.Len()),
				zap.Duration("duration", duration),
				zap.Int("retry", i),
			)
			return nil
		}

		lastErr = err
		logger.Warn("stream load failed, retrying",
			zap.Error(err),
			zap.Int("retry", i),
			zap.Int("max_retries", w.cfg.MaxRetries),
		)

		// 最后一次重试失败，直接返回
		if i == w.cfg.MaxRetries {
			break
		}

		// 等待后重试
		backoff := time.Duration(i+1) * time.Second
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}

	return errors.Wrap(errors.ErrCodeDorisStreamLoad, "stream load failed after retries", lastErr)
}

// doStreamLoad 执行Stream Load
func (w *StreamLoadWriter) doStreamLoad(ctx context.Context, data []byte, rowCount int) error {
	// 选择FE节点（简单轮询，实际可以实现更复杂的负载均衡）
	feHost := w.cfg.FEHosts[0]

	// 构建URL
	url := fmt.Sprintf("http://%s/api/%s/%s/_stream_load",
		feHost,
		w.cfg.Database,
		w.cfg.Table,
	)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(errors.ErrCodeDorisStreamLoad, "failed to create request", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", w.authHeader)
	req.Header.Set("Expect", "100-continue")
	// req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("format", "json")
	req.Header.Set("read_json_by_line", "true")
	//req.Header.Set("strip_outer_array", "false")
	req.Header.Set("max_filter_ratio", "0.1") // 允许10%的错误率

	logger.Debug("sending stream load request",
		zap.String("url", url),
		zap.Int("rows", rowCount),
		zap.Int("size_bytes", len(data)),
	)

	// 发送请求
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(errors.ErrCodeDorisStreamLoad, "failed to send request", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(errors.ErrCodeDorisStreamLoad, "failed to read response", err)
	}

	// 解析响应
	var result StreamLoadResult
	if err := json.Unmarshal(body, &result); err != nil {
		return errors.Wrap(errors.ErrCodeDorisStreamLoad, "failed to parse response", err)
	}

	// 检查结果
	if result.Status != "Success" {
		logger.Error("stream load failed", zap.Any("result", result))
		return errors.New(errors.ErrCodeDorisStreamLoad,
			fmt.Sprintf("stream load failed: %s, message: %s", result.Status, result.Message))
	}

	logger.Info("stream load response",
		zap.String("status", result.Status),
		zap.Int64("loaded_rows", result.NumberLoadedRows),
		zap.Int64("filtered_rows", result.NumberFilteredRows),
		zap.Int("load_time_ms", result.LoadTimeMs),
	)

	return nil
}

// Close 关闭写入器
func (w *StreamLoadWriter) Close() error {
	w.httpClient.CloseIdleConnections()
	return nil
}

// StreamLoadResult Stream Load响应结果
type StreamLoadResult struct {
	TxnID                  int64  `json:"TxnId"`
	Label                  string `json:"Label"`
	Status                 string `json:"Status"`
	Message                string `json:"Message"`
	NumberTotalRows        int64  `json:"NumberTotalRows"`
	NumberLoadedRows       int64  `json:"NumberLoadedRows"`
	NumberFilteredRows     int64  `json:"NumberFilteredRows"`
	NumberUnselectedRows   int64  `json:"NumberUnselectedRows"`
	LoadBytes              int64  `json:"LoadBytes"`
	LoadTimeMs             int    `json:"LoadTimeMs"`
	BeginTxnTimeMs         int    `json:"BeginTxnTimeMs"`
	StreamLoadPutTimeMs    int    `json:"StreamLoadPutTimeMs"`
	ReadDataTimeMs         int    `json:"ReadDataTimeMs"`
	WriteDataTimeMs        int    `json:"WriteDataTimeMs"`
	CommitAndPublishTimeMs int    `json:"CommitAndPublishTimeMs"`
	ErrorURL               string `json:"ErrorURL"`
}
