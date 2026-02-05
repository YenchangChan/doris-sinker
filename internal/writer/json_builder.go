package writer

import (
	"bytes"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/doris-sinker/doris-sinker/pkg/pool"
)

// BuildJSONLines 构建JSON Lines格式数据（每行一个JSON对象）
// 优化版本：预分配缓冲区，减少内存分配
func BuildJSONLines(batch [][]interface{}, columns []string) *bytes.Buffer {
	if len(batch) == 0 {
		return pool.GetBuffer()
	}

	// 预估算大小：每行约200字节
	estimatedSize := len(batch) * 200
	buf := pool.GetBuffer()
	buf.Grow(estimatedSize)

	for i, row := range batch {
		// 直接序列化为JSON，避免创建map
		buf.WriteByte('{')

		for j, col := range columns {
			if j < len(row) {
				if j > 0 {
					buf.WriteByte(',')
				}

				// 写入字段名
				buf.WriteByte('"')
				buf.WriteString(col)
				buf.WriteByte('"')
				buf.WriteByte(':')

				// 写入值
				writeValue(buf, row[j])
			}
		}

		buf.WriteByte('}')

		// 最后一行不加换行符
		if i < len(batch)-1 {
			buf.WriteByte('\n')
		}
	}

	return buf
}

// writeValue 将值写入buffer（避免反射开销）
func writeValue(buf *bytes.Buffer, val interface{}) {
	switch v := val.(type) {
	case string:
		buf.WriteByte('"')
		buf.WriteString(v)
		buf.WriteByte('"')
	case int:
		buf.WriteString(strconv.Itoa(v))
	case int64:
		buf.WriteString(strconv.FormatInt(v, 10))
	case float64:
		buf.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
	case bool:
		if v {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case nil:
		buf.WriteString("null")
	default:
		// 其他类型使用sonic序列化
		jsonBytes, err := sonic.Marshal(v)
		if err != nil {
			buf.WriteString("null")
		} else {
			buf.Write(jsonBytes)
		}
	}
}
