package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/doris-sinker/doris-sinker/internal/config"
	"github.com/doris-sinker/doris-sinker/pkg/errors"
	"github.com/doris-sinker/doris-sinker/pkg/logger"
	"go.uber.org/zap"
)

// Fetcher Schema获取器
type Fetcher struct {
	cfg config.DorisConfig
}

// NewFetcher 创建Fetcher
func NewFetcher(cfg config.DorisConfig) *Fetcher {
	return &Fetcher{cfg: cfg}
}

// FetchFromDoris 从Doris获取表结构
func (f *Fetcher) FetchFromDoris(ctx context.Context) (*Schema, error) {
	// 连接Doris（使用MySQL协议）
	host := strings.Split(f.cfg.FEHosts[0], ":")[0]
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=10s",
		f.cfg.User,
		f.cfg.Password,
		host,
		f.cfg.QueryPort,
		f.cfg.Database,
	)

	logger.Info("connecting to doris to fetch schema",
		zap.String("host", host),
		zap.Int("port", f.cfg.QueryPort),
		zap.String("database", f.cfg.Database),
		zap.String("table", f.cfg.Table),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeSchemaFetch, "failed to connect to doris", err)
	}
	defer db.Close()

	// 设置连接池参数
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Wrap(errors.ErrCodeSchemaFetch, "failed to ping doris", err)
	}

	// 执行DESCRIBE TABLE查询
	query := fmt.Sprintf("DESCRIBE `%s`.`%s`", f.cfg.Database, f.cfg.Table)
	logger.Debug("executing describe query", zap.String("query", query))

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeSchemaFetch, "failed to describe table", err)
	}
	defer rows.Close()

	// 解析结果
	var columns []Column
	for rows.Next() {
		var field, dataType, null, key, defaultVal, extra sql.NullString
		err := rows.Scan(&field, &dataType, &null, &key, &defaultVal, &extra)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeSchemaFetch, "failed to scan row", err)
		}

		if !field.Valid || !dataType.Valid {
			continue
		}

		col := Column{
			Name: field.String,
			Type: MapDorisType(dataType.String),
		}
		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(errors.ErrCodeSchemaFetch, "error iterating rows", err)
	}

	if len(columns) == 0 {
		return nil, errors.New(errors.ErrCodeSchemaFetch, "no columns found in table")
	}

	schema := NewSchema(columns)
	logger.Info("schema fetched successfully",
		zap.Int("column_count", len(columns)),
		zap.String("schema", schema.String()),
	)

	return schema, nil
}

// FetchFromConfig 从配置文件获取表结构
func FetchFromConfig(cfg config.ManualSchemaConfig) (*Schema, error) {
	if len(cfg.Columns) == 0 {
		return nil, errors.New(errors.ErrCodeSchemaFetch, "no columns in manual config")
	}

	columns := make([]Column, len(cfg.Columns))
	for i, col := range cfg.Columns {
		columns[i] = Column{
			Name: col.Name,
			Type: col.Type,
		}
	}

	schema := NewSchema(columns)
	logger.Info("schema loaded from config",
		zap.Int("column_count", len(columns)),
	)

	return schema, nil
}
