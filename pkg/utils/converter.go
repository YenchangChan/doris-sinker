package utils

import (
	"fmt"
	"strconv"
)

// ToString 转换为字符串
func ToString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", val)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%f", val)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// EscapeCSV 转义CSV字段
func EscapeCSV(s string) string {
	// 如果包含逗号、引号或换行符，需要用引号包裹并转义引号
	needQuote := false
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			needQuote = true
			break
		}
	}

	if !needQuote {
		return s
	}

	// 转义引号
	result := ""
	for _, c := range s {
		if c == '"' {
			result += "\"\""
		} else {
			result += string(c)
		}
	}

	return "\"" + result + "\""
}
