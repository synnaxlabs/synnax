package seeker

import (
	"github.com/synnaxlabs/x/iter"
)

type alignment struct {
	pos  int
	root int
}

type RollingMemoryIndex struct {
	values *iter.RollingArray[alignment]
}

func NewMemoryIndex(cap int) *RollingMemoryIndex {
	return &RollingMemoryIndex{
		values: iter.NewRollingArray[alignment](cap),
	}
}

func (o *RollingMemoryIndex) Seek(pos int) (int, error) {
	start := 0
	end := o.values.Size - 1
	for start <= end {
		mid := (start + end) / 2
		g := o.values.Get(mid)
		if g.pos == pos {
			return g.root, nil
		} else if g.pos < pos {
			start = mid + 1
		} else {
			end = mid - 1
		}
	}
	if end == -1 {
		return -1, nil
	}
	return o.values.Get(end).root, nil
}

func (o *RollingMemoryIndex) Add(pos int, root int) {
	o.values.Append(alignment{pos: pos, root: root})
}

type compoundMemoryIndexEntry struct {
	capacity int
	index    *RollingMemoryIndex
}

type CompoundMemoryIndex struct {
	rootSize int
	indexes  []compoundMemoryIndexEntry
}

func (c *CompoundMemoryIndex) Seek(pos int) (int, error) {
	for _, i := range c.indexes {
		r, err := i.Seek(pos)
		if err == nil {
			return r, nil
		}
	}
	return -1, nil
}

func (c *CompoundMemoryIndex) Add(pos int, root int) {

}
