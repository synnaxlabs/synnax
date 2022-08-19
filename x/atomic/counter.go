package atomic

import (
	"github.com/arya-analytics/x/types"
	"sync/atomic"
)

// Int32Counter is an int32 counter  that is safe for concurrent use.
type Int32Counter struct{ value int32 }

func (c *Int32Counter) Add(delta ...int32) int32 {
	return atomic.AddInt32(&c.value, parseDelta(delta...))
}

func (c *Int32Counter) Value() int32 { return atomic.LoadInt32(&c.value) }

// Int64Counter is an int64 counter  that is safe for concurrent use.
type Int64Counter struct{ value int64 }

func (c *Int64Counter) Add(delta ...int64) int64 {
	return atomic.AddInt64(&c.value, parseDelta(delta...))
}

func (c *Int64Counter) Value() int64 { return atomic.LoadInt64(&c.value) }

func parseDelta[V types.Numeric](delta ...V) V {
	var (
		_delta V
		ld     = len(delta)
	)
	if ld == 0 {
		_delta = 1
	} else if ld == 1 {
		_delta = delta[0]
	} else {
		for _, v := range delta {
			_delta += v
		}
	}
	return _delta
}
