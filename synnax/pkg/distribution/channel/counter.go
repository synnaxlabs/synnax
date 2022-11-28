package channel

import (
	"github.com/synnaxlabs/x/counter"
	"github.com/synnaxlabs/x/kv"
)

const counterKey = "distribution.channel.counter"

type keyCounter struct {
	err      error
	internal *kv.PersistedCounter
}

func (c *keyCounter) Add(delta ...uint16) uint16 {
	var total int64
	for _, d := range delta {
		total += int64(d)
	}
	total, c.err = c.internal.Add(int64(total))
	return uint16(total)
}

func (c *keyCounter) Value() uint16 { return uint16(c.internal.Value()) }

func (c *keyCounter) Error() error { return c.err }

func openCounter(kve kv.DB) (counter.Uint16Error, error) {
	c, err := kv.OpenCounter(kve, []byte(counterKey))
	if err != nil {
		return nil, err
	}
	return &keyCounter{internal: c}, nil
}
