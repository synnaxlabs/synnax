package index

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/array"
	"github.com/synnaxlabs/x/telem"
)

type Buffered struct {
	Wrapped  Writer
	searcher CompoundSearcher
	buffers  [][]Alignment
}

var _ Index = (*Buffered)(nil)

func (b *Buffered) Key() core.ChannelKey {
	return b.Wrapped.Key()
}

func (b *Buffered) Release() error {
	return b.Wrapped.Release()
}

func (b *Buffered) Write(alignments []Alignment) error {
	b.buffers = append(b.buffers, alignments)
	idx := &BinarySearch{
		Array: array.Searchable[Alignment]{Array: array.Wrap[Alignment](alignments)},
	}
	b.searcher = append(b.searcher, idx)
	return nil
}

func (b *Buffered) SearchP(stamp telem.TimeStamp, approx position.Approximation) (position.Approximation, error) {
	return b.searcher.SearchP(stamp, approx)
}

func (b *Buffered) SearchTS(pos position.Position, approx telem.Approximation) (telem.Approximation, error) {
	return b.searcher.SearchTS(pos, approx)
}

func (b *Buffered) Commit(pos position.Position) error {
	flushThreshold := -1
	for i := len(b.buffers) - 1; i >= 0; i-- {
		buf := b.buffers[i]
		if len(buf) == 0 {
			continue
		}
		last := buf[len(buf)-1]
		if last.Pos <= pos {
			flushThreshold = i
			break
		}
	}
	if flushThreshold == -1 {
		return nil
	}
	for i := 0; i <= flushThreshold; i++ {
		buf := b.buffers[i]
		if len(buf) == 0 {
			continue
		}
		if err := b.Wrapped.Write(buf); err != nil {
			return err
		}
	}
	b.buffers = b.buffers[flushThreshold+1:]
	b.searcher = b.searcher[flushThreshold+1:]
	return nil
}
