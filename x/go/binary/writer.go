package binary

import (
	"encoding/binary"
	"fmt"
)

// Writer makes it easy to writer various primitive data types to binary using a given
// byte order.
type Writer struct {
	offset    int
	byteOrder binary.ByteOrder
	buf       []byte
}

// NewWriter creates a new writer with the given size, starting offset, and byte order.
func NewWriter(size int, offset int, order binary.ByteOrder) *Writer {
	if offset > size {
		panic(fmt.Sprintf("offset %v is greater than buffer allocation size %v", offset, size))
	}
	return &Writer{buf: make([]byte, size), offset: offset, byteOrder: order}
}

func (w *Writer) Reset() {
	w.offset = 0
}

func (w *Writer) Resize(size int) {
	if size < len(w.buf) {
		w.buf = w.buf[:size]
	} else if size > len(w.buf) {
		w.buf = make([]byte, size)
	}
}

// Uint8 writes a new Uint8 to the buffer. If the buffer is already at capacity, returns
// 0, otherwise returns 1.
func (w *Writer) Uint8(value uint8) int {
	if w.offset+1 > len(w.buf) {
		return 0
	}
	w.buf[w.offset] = value
	w.offset += 1
	return 1
}

// Uint32 writes a new Uint32 to the buffer. If the buffer is at capacity, returns 0,
// otherwise returns 1.
func (w *Writer) Uint32(value uint32) int {
	if w.offset+4 > len(w.buf) {
		return 0
	}
	w.byteOrder.PutUint32(w.buf[w.offset:w.offset+4], value)
	w.offset += 4
	return 4
}

// Uint64 writes a new Uint64 to the buffer. If the buffer is at capacity, returns 0,
// otherwise returns 8.
func (w *Writer) Uint64(value uint64) int {
	if w.offset+8 > len(w.buf) {
		return 0
	}
	w.byteOrder.PutUint64(w.buf[w.offset:w.offset+8], value)
	w.offset += 8
	return 8
}

// Write writes the given data to the buffer. Returns the number of bytes written,
// which is the min of len(data) and the remaining available buffer capacity.
func (w *Writer) Write(data []byte) int {
	count := copy(w.buf[w.offset:], data)
	w.offset += count
	return count
}

// Bytes returns the underlying binary buffer.
func (w *Writer) Bytes() []byte { return w.buf[:w.offset] }
