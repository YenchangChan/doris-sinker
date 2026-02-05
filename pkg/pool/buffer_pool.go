package pool

import (
	"bytes"
	"sync"
)

// BufferPool 字节缓冲池
var BufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GetBuffer 从池中获取buffer
func GetBuffer() *bytes.Buffer {
	buf := BufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer 将buffer放回池中
func PutBuffer(buf *bytes.Buffer) {
	if buf != nil {
		BufferPool.Put(buf)
	}
}
