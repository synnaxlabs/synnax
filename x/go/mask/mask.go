package mask

import (
	"context"
	"github.com/synnaxlabs/x/iter"
	"math"
)

type Uint16 struct {
	HiddenCount uint16
	v           [math.MaxUint16 / 64]uint64
}

func (m *Uint16) Hide(i uint16) {
	m.v[i/64] |= 1 << (i % 64)
	m.HiddenCount++
}

func (m *Uint16) VisibleCount() uint16 {
	return math.MaxUint16 - m.HiddenCount
}

func (m *Uint16) Visible(i uint16) bool {
	return m.v[i/64]&(1<<(i%64)) == 0
}

func (m *Uint16) Iter() iter.Nexter[uint16] {
	i := uint16(0)
	return iter.NexterFunc[uint16](func(_ context.Context) (uint16, bool) {
		for ; i < math.MaxUint16; i++ {
			if m.Visible(i) {
				return i, true
			}
		}
		return 0, false
	})
}
