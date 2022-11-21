package buffalo

import (
	"bytes"
	"github.com/synnaxlabs/x/atomic"
	"sync"
)

type Pool struct {
	wrapped *sync.Pool
}

func NewPool(cap int) *Pool {
	return &Pool{
		wrapped: &sync.Pool{
			New: func() interface{} {
				return &Buffer{
					refCount: &atomic.Int32Counter{},
					wrapped:  bytes.NewBuffer(make([]byte, 0, cap)),
				}
			},
		},
	}
}

func (p *Pool) Acquire() *Buffer {
	b := p.wrapped.Get().(*Buffer)
	b.Retain()
	return b
}

func (p *Pool) release(b *Buffer) { p.wrapped.Put(b) }
