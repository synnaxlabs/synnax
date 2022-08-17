package atomic

import "sync/atomic"

// Int32Counter is an int32 counter  that is safe for concurrent use.
type Int32Counter struct{ value int32 }

func (c *Int32Counter) Add(delta int32) int32 { return atomic.AddInt32(&c.value, delta) }

func (c *Int32Counter) Value() int32 { return atomic.LoadInt32(&c.value) }
