package buffalo

import (
	"bytes"
	"github.com/synnaxlabs/x/atomic"
	"io"
)

type Buffer struct {
	refCount *atomic.Int32Counter
	internal *bytes.Buffer
	pool     *Pool
}

func (b *Buffer) Retain() {
	b.refCount.Add(1)
}

func (b *Buffer) Release() {
	c := b.refCount.Add(-1)
	if c < 0 {
		panic("ref count < 0")
	}
	if c == 0 && b.pool != nil {
		b.pool.release(b)
	}
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	return b.internal.Write(p)
}

func (b *Buffer) Resize(n int) {
	b.internal.Reset()
	b.internal.Grow(n)
}

func (b *Buffer) Reset() {
	b.internal.Reset()
}

func (b *Buffer) Cap() int {
	return b.internal.Cap()
}

func (b *Buffer) Len() int {
	return b.internal.Len()
}

func (b *Buffer) WriteTo(w io.Writer) (n int64, err error) {
	return b.internal.WriteTo(w)
}
