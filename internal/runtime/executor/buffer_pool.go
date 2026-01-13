// Package executor provides runtime execution capabilities for various AI service providers.
// This file implements buffer pooling for efficient memory reuse.
package executor

import (
	"bytes"
	"sync"
)

// BufferPool provides a pool of reusable byte buffers to reduce GC pressure.
// It uses sync.Pool for efficient concurrent access.
type BufferPool struct {
	pool sync.Pool
	size int
}

// NewBufferPool creates a new buffer pool with the specified initial buffer size.
func NewBufferPool(initialSize int) *BufferPool {
	if initialSize <= 0 {
		initialSize = 64 * 1024 // 64KB default
	}
	return &BufferPool{
		size: initialSize,
		pool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, initialSize))
			},
		},
	}
}

// Get retrieves a buffer from the pool.
// The buffer is reset and ready for use.
func (p *BufferPool) Get() *bytes.Buffer {
	buf := p.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns a buffer to the pool for reuse.
// The buffer is reset before being returned to the pool.
func (p *BufferPool) Put(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	// Avoid keeping very large buffers in the pool
	if buf.Cap() > p.size*4 {
		return
	}
	buf.Reset()
	p.pool.Put(buf)
}

// ByteSlicePool provides a pool of reusable byte slices.
type ByteSlicePool struct {
	pool sync.Pool
	size int
}

// NewByteSlicePool creates a new byte slice pool with the specified size.
func NewByteSlicePool(size int) *ByteSlicePool {
	if size <= 0 {
		size = 32 * 1024 // 32KB default
	}
	return &ByteSlicePool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				b := make([]byte, size)
				return &b
			},
		},
	}
}

// Get retrieves a byte slice from the pool.
func (p *ByteSlicePool) Get() []byte {
	return *p.pool.Get().(*[]byte)
}

// Put returns a byte slice to the pool.
func (p *ByteSlicePool) Put(b []byte) {
	if cap(b) < p.size {
		return
	}
	b = b[:p.size]
	p.pool.Put(&b)
}

// Global buffer pools for common use cases
var (
	// RequestBufferPool is used for request body buffering
	RequestBufferPool = NewBufferPool(64 * 1024) // 64KB

	// ResponseBufferPool is used for response body buffering
	ResponseBufferPool = NewBufferPool(128 * 1024) // 128KB

	// StreamChunkPool is used for streaming chunk buffering
	StreamChunkPool = NewByteSlicePool(16 * 1024) // 16KB
)

// GetRequestBuffer gets a buffer from the request pool.
func GetRequestBuffer() *bytes.Buffer {
	return RequestBufferPool.Get()
}

// PutRequestBuffer returns a buffer to the request pool.
func PutRequestBuffer(buf *bytes.Buffer) {
	RequestBufferPool.Put(buf)
}

// GetResponseBuffer gets a buffer from the response pool.
func GetResponseBuffer() *bytes.Buffer {
	return ResponseBufferPool.Get()
}

// PutResponseBuffer returns a buffer to the response pool.
func PutResponseBuffer(buf *bytes.Buffer) {
	ResponseBufferPool.Put(buf)
}
