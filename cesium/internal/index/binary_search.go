package index

import (
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/array"
	"github.com/synnaxlabs/x/compare"
	"github.com/synnaxlabs/x/telem"
	"sync"
)

// BinarySearch implements an in-memory index that uses binary search to resolve timestamps
// from positions.
type BinarySearch struct {
	// Every represents the separation between positions accepted upon writing to the
	// index. The larger the value of Every, the less precise the index, but the larger
	// span it occupies on the same memory footprint.
	Every position.Span
	// HWM is a high-water mark tracking the most recent position written to the index.
	HWM position.Position
	// Array is the underlying binary searchable array that the index uses as its backing
	// store.
	Array array.Searchable[Alignment]
	mu    sync.RWMutex
	nopReleaser
}

var _ Searcher = (*BinarySearch)(nil)

func (bsi *BinarySearch) Size() int { return bsi.Array.Size() }

// SearchP implements the PositionSearcher interface.
func (bsi *BinarySearch) SearchP(
	stamp telem.TimeStamp,
	approx position.Approximation,
) (position.Approximation, error) {
	return bsi.searchP(stamp, approx), nil
}

// SearchTS implements the StampSearcher interface.
func (bsi *BinarySearch) SearchTS(
	pos position.Position,
	approx telem.Approximation,
) (telem.Approximation, error) {
	return bsi.searchTS(pos, approx), nil
}

func (bsi *BinarySearch) searchP(
	stamp telem.TimeStamp,
	approx position.Approximation,
) position.Approximation {
	bsi.mu.RLock()
	defer bsi.mu.RUnlock()
	if bsi.Size() == 0 {
		return position.Uncertain
	}

	c := compare.NumericUnary(stamp)
	a, i := bsi.Array.Search(func(a Alignment) compare.Result { return c(a.Stamp) })

	// We've resolved the value with certainty.
	if a.Stamp == stamp {
		return position.ExactlyAt(a.Pos)
	}

	// We know the value is after the end of the index.
	if i == bsi.Array.Size() {
		return position.Between(a.Pos, approx.End)
	}

	// We know the value is before the start of the index.
	if i == -1 {
		return position.Between(approx.Start, a.Pos)
	}

	// We know the value is somewhere between these two.
	return position.Between(a.Pos, bsi.Array.Get(i+1).Pos)
}

func (bsi *BinarySearch) searchTS(
	pos position.Position,
	approx telem.Approximation,
) telem.Approximation {
	bsi.mu.RLock()
	defer bsi.mu.RUnlock()
	if bsi.Size() == 0 {
		return telem.Uncertain
	}

	c := compare.NumericUnary(pos)
	a, i := bsi.Array.Search(func(a Alignment) compare.Result { return c(a.Pos) })

	// We've resolved the value with  certainty.
	if a.Pos == pos {
		return telem.ExactlyAt(a.Stamp)
	}

	// We know the value is after the end of the index.
	if i == bsi.Array.Size() {
		return telem.Between(a.Stamp, approx.End)
	}

	// We know the value is before the start of the index.
	if i == -1 {
		return telem.Between(approx.Start, a.Stamp)
	}

	// We know the value is somewhere between these two.
	return telem.Between(a.Stamp, bsi.Array.Get(i+1).Stamp)
}

// Write implements the Writer interface.
func (bsi *BinarySearch) Write(alignments []Alignment) error {
	bsi.mu.Lock()
	defer bsi.mu.Unlock()
	for _, a := range alignments {
		if position.Span(a.Pos-bsi.HWM) >= bsi.Every {
			bsi.Array.Append(a)
			bsi.HWM = a.Pos
		}
	}
	return nil
}

func binarySearchP(
	stamp telem.TimeStamp,
	approx position.Approximation,
	alignments []Alignment,
) position.Approximation {
	bs := &BinarySearch{Array: array.Searchable[Alignment]{Array: array.Wrap(alignments)}}
	return bs.searchP(stamp, approx)
}

func binarySearchTS(
	pos position.Position,
	approx telem.Approximation,
	alignments []Alignment,
) telem.Approximation {
	bs := &BinarySearch{Array: array.Searchable[Alignment]{Array: array.Wrap(alignments)}}
	return bs.searchTS(pos, approx)
}
