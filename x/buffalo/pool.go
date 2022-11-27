package buffalo

import (
	"bytes"
	"github.com/synnaxlabs/x/atomic"
	"sync"
)

type Pool struct {
	internal *sync.Pool
}

func NewPool(cap int) *Pool {
	return &Pool{
		internal: &sync.Pool{
			New: func() interface{} {
				return &Buffer{
					refCount: &atomic.Int32Counter{},
					internal: bytes.NewBuffer(make([]byte, 0, cap)),
				}
			},
		},
	}
}

func (p *Pool) Acquire() *Buffer {
	b := p.internal.Get().(*Buffer)
	b.Retain()
	return b
}

func (p *Pool) release(b *Buffer) { p.internal.Put(b) }
