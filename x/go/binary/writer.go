package binary

import (
	"encoding/binary"
)

type Writer struct {
	offset    int
	byteOrder binary.ByteOrder
	buf       []byte
}

func NewWriter(size int, offset int) *Writer {
	return &Writer{buf: make([]byte, size), offset: offset, byteOrder: binary.LittleEndian}
}

func (w *Writer) Uint8(value uint8) {
	w.buf[w.offset] = value
	w.offset += 1
}

func (w *Writer) Uint32(value uint32) {
	w.byteOrder.PutUint32(w.buf[w.offset:w.offset+4], value)
	w.offset += 4
}

func (w *Writer) Uint64(value uint64) {
	w.byteOrder.PutUint64(w.buf[w.offset:w.offset+8], value)
	w.offset += 8
}

func (w *Writer) Write(data []byte) {
	copy(w.buf[w.offset:w.offset+len(data)], data)
	w.offset += len(data)
}

func (w *Writer) Bytes() []byte {
	return w.buf
}
