package errors

import (
	"fmt"
)

// ErrorCode 错误码类型
type ErrorCode int

const (
	// Kafka相关错误 1xxx
	ErrCodeKafkaConnect ErrorCode = 1001
	ErrCodeKafkaConsume ErrorCode = 1002
	ErrCodeKafkaCommit  ErrorCode = 1003

	// Schema相关错误 2xxx
	ErrCodeSchemaFetch  ErrorCode = 2001
	ErrCodeJSONParse    ErrorCode = 2002
	ErrCodeFieldMapping ErrorCode = 2003
	ErrCodeTypeConvert  ErrorCode = 2004

	// Batch相关错误 3xxx
	ErrCodeBatchFull  ErrorCode = 3001
	ErrCodeBatchFlush ErrorCode = 3002

	// Doris相关错误 4xxx
	ErrCodeDorisConnect    ErrorCode = 4001
	ErrCodeDorisStreamLoad ErrorCode = 4002
	ErrCodeDorisQuery      ErrorCode = 4003

	// 配置相关错误 5xxx
	ErrCodeConfigLoad     ErrorCode = 5001
	ErrCodeConfigValidate ErrorCode = 5002
)

// SinkerError 自定义错误类型
type SinkerError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *SinkerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *SinkerError) Unwrap() error {
	return e.Err
}

// New 创建新错误
func New(code ErrorCode, message string) *SinkerError {
	return &SinkerError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误
func Wrap(code ErrorCode, message string, err error) *SinkerError {
	return &SinkerError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if se, ok := err.(*SinkerError); ok {
		switch se.Code {
		case ErrCodeKafkaConnect, ErrCodeDorisConnect, ErrCodeDorisStreamLoad:
			return true
		default:
			return false
		}
	}

	return false
}
