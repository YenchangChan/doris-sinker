package schema

import (
	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"

	"github.com/doris-sinker/doris-sinker/pkg/errors"
	"github.com/doris-sinker/doris-sinker/pkg/logger"
	"go.uber.org/zap"
)

// Mapper JSON到Row的映射器
type Mapper struct {
	schema *Schema
}

// NewMapper 创建Mapper
func NewMapper(schema *Schema) *Mapper {
	return &Mapper{schema: schema}
}

// MapJSONToRow 将JSON字节数组映射为Row
func (m *Mapper) MapJSONToRow(jsonBytes []byte) ([]interface{}, error) {
	row := make([]interface{}, m.schema.ColumnCount())

	for i, col := range m.schema.Columns {
		// 使用sonic.Get()直接提取字段，无需完整解析
		node, err := sonic.Get(jsonBytes, col.Name)

		if err != nil || !node.Exists() {
			// 字段不存在，填充零值
			row[i] = GetZeroValue(col.Type)
			continue
		}

		// 根据类型提取值
		value, err := m.extractValue(node, col.Type, col.Name)
		if err != nil {
			// 类型转换失败，使用零值并记录警告
			logger.Warn("field type conversion failed, using zero value",
				zap.String("field", col.Name),
				zap.String("type", col.Type),
				zap.Error(err),
			)
			row[i] = GetZeroValue(col.Type)
			continue
		}

		row[i] = value
	}

	return row, nil
}

// extractValue 提取并转换值
func (m *Mapper) extractValue(node ast.Node, dataType string, fieldName string) (interface{}, error) {
	switch dataType {
	case "INT", "BIGINT":
		val, err := node.Int64()
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeTypeConvert, "failed to convert to int", err)
		}
		return val, nil

	case "BOOLEAN":
		val, err := node.Bool()
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeTypeConvert, "failed to convert to boolean", err)
		}
		return val, nil

	case "FLOAT":
		val, err := node.Number()
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeTypeConvert, "failed to convert to float", err)
		}
		return val, nil

	case "VARCHAR", "STRING", "DATE", "DATETIME":
		val, err := node.String()
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeTypeConvert, "failed to convert to string", err)
		}
		return val, nil

	default:
		// 未知类型，尝试转换为字符串
		val, err := node.String()
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeTypeConvert, "failed to convert unknown type", err)
		}
		return val, nil
	}
}

// GetSchema 获取Schema
func (m *Mapper) GetSchema() *Schema {
	return m.schema
}
