package io

import (
	"encoding/binary"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/counter"
	"io"
)

type Int32Counter struct {
	err error
	counter.Int32
	f   ReaderAtWriterAtCloser
	buf []byte
}

func NewInt32Counter(f ReaderAtWriterAtCloser, base counter.Int32) (*Int32Counter, error) {
	i := &Int32Counter{
		Int32: base,
		f:     f,
		buf:   make([]byte, 4),
	}
	i.load()
	return i, i.Error()
}

func (c *Int32Counter) load() int32 {
	_, err := c.f.ReadAt(c.buf, 0)
	if !errors.Is(err, io.EOF) {
		c.err = err
	}
	return int32(binary.LittleEndian.Uint32(c.buf))
}

func (c *Int32Counter) Add(delta ...int32) int32 {
	v := c.Int32.Add(delta...)
	binary.LittleEndian.PutUint32(c.buf, uint32(v))
	_, c.err = c.f.WriteAt(c.buf, 0)
	return v
}

func (c *Int32Counter) Error() error { return c.err }

func (c *Int32Counter) Close() error { return c.f.Close() }
