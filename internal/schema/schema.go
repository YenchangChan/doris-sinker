package schema

import (
	"fmt"
	"strings"
)

// Column 列定义
type Column struct {
	Name string
	Type string
}

// Schema 表结构
type Schema struct {
	Columns []Column
	// 字段名到索引的映射，加速查找
	fieldIndex map[string]int
}

// NewSchema 创建Schema
func NewSchema(columns []Column) *Schema {
	fieldIndex := make(map[string]int, len(columns))
	for i, col := range columns {
		fieldIndex[col.Name] = i
	}

	return &Schema{
		Columns:    columns,
		fieldIndex: fieldIndex,
	}
}

// GetFieldIndex 获取字段索引
func (s *Schema) GetFieldIndex(fieldName string) (int, bool) {
	idx, ok := s.fieldIndex[fieldName]
	return idx, ok
}

// ColumnCount 返回列数
func (s *Schema) ColumnCount() int {
	return len(s.Columns)
}

// String 返回Schema的字符串表示
func (s *Schema) String() string {
	var sb strings.Builder
	sb.WriteString("Schema{")
	for i, col := range s.Columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%s:%s", col.Name, col.Type))
		if i >= 5 { // 只显示前5个字段
			sb.WriteString(fmt.Sprintf(", ... (%d more)", len(s.Columns)-5))
			break
		}
	}
	sb.WriteString("}")
	return sb.String()
}

// MapDorisType 映射Doris数据类型到内部类型
func MapDorisType(dorisType string) string {
	// 提取基础类型（去除长度等信息）
	baseType := strings.ToUpper(strings.Split(dorisType, "(")[0])
	baseType = strings.TrimSpace(baseType)

	switch baseType {
	case "TINYINT", "SMALLINT", "INT", "INTEGER":
		return "INT"
	case "BIGINT":
		return "BIGINT"
	case "BOOLEAN", "BOOL":
		return "BOOLEAN"
	case "FLOAT", "DOUBLE", "DECIMAL":
		return "FLOAT"
	case "CHAR", "VARCHAR":
		return "VARCHAR"
	case "STRING", "TEXT", "MEDIUMTEXT", "LONGTEXT":
		return "STRING"
	case "DATE":
		return "DATE"
	case "DATETIME", "TIMESTAMP":
		return "DATETIME"
	default:
		return "STRING" // 默认按字符串处理
	}
}

// GetZeroValue 获取类型的零值
func GetZeroValue(dataType string) interface{} {
	switch dataType {
	case "BIGINT", "INT":
		return int64(0)
	case "BOOLEAN":
		return false
	case "FLOAT":
		return float64(0)
	case "VARCHAR", "STRING", "DATE", "DATETIME":
		return ""
	default:
		return nil
	}
}
