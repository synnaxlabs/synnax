package accumulate

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

// Accumulator accumulates segments that satisfy a given position range.
// The Segments in an accumulator are always sorted by Alignment position.
type Accumulator struct {
	// Density sets the density of the accumulated segments.
	// This density is used to calculate the position range of a segment.
	Density telem.Density
	// Bounds sets the bounds for the accumulator. The accumulator will ignore any
	// segments not within these bounds. This value is altered on calls to Reset.
	// calls to Reset.
	Bounds position.Range
	// Merge optionally merges contiguous segments on disk and in position space.
	Merge bool
	// Slice optionally slices segments that overlap with the bounds to fit within them.
	Slice bool
	// Segments is a list of accumulated segments. The caller should not modify these
	// segments directly, and should instead call Accumulate. The Segments are guaranteed
	// to be in time order.
	Segments []core.SegmentMD
}

// Reset resets the accumulator, clearing all segments and resetting the bounds.
func (a *Accumulator) Reset(bounds position.Range) {
	a.Bounds = bounds
	a.Segments = nil
}

// Accumulate accumulates the given segment into the accumulator. If the segment is
// not within the bounds of the accumulator, it is ignored. If Accumulator.Slice is
// true, the segment is sliced to fit within the bounds of the accumulator. If
// Accumulator.Merge is true, the segment is merged with the first or last
// segment in the accumulator if possible (two segments will be merged if it is contiguous
// in position space and on disk).
func (a *Accumulator) Accumulate(smd core.SegmentMD) bool {
	rng := smd.Range(a.Density)
	if !rng.OverlapsWith(a.Bounds) {
		return false
	}
	smd = a.slice(smd)
	if len(a.Segments) == 0 {
		a.Segments = append(a.Segments, smd)
	} else if smd.Alignment.Before(a.Segments[0].Alignment) {
		a.mergeStart(smd)
	} else {
		a.mergeEnd(smd)
	}
	return true
}

// Satisfied returns true if the segments in the accumulator span its bounds i.e.
// the segments represent all possible data in the bounds.
func (a *Accumulator) Satisfied() bool {
	// get the start of the first segment and the end of the last segment
	// if the range between them is equal to the bounds, then we are satisfied
	if len(a.Segments) == 0 {
		return false
	}
	start := a.Segments[0].Alignment
	end := a.Segments[len(a.Segments)-1].End(a.Density)
	return a.Bounds.Equals(position.Range{Start: start, End: end})
}

// PartiallySatisfied returns true if the segments in the accumulator partially span it's bounds
// i.e. the segments represent some but not all possible data in the bounds. Returns
// false if the segments are empty.
func (a *Accumulator) PartiallySatisfied() bool { return len(a.Segments) > 0 }

func (a *Accumulator) slice(smd core.SegmentMD) core.SegmentMD {
	if a.Slice {
		smd = Slice(smd, a.Density, a.Bounds)
	}
	return smd
}

func (a *Accumulator) mergeStart(smd core.SegmentMD) {
	if !a.Merge {
		a.Segments = append([]core.SegmentMD{smd}, a.Segments...)
		return
	}
	compacted, ok := TryMerge(smd, a.Segments[0], a.Density)
	if !ok {
		a.Segments = append([]core.SegmentMD{smd}, a.Segments...)
	} else {
		a.Segments[0] = compacted
	}
}

func (a *Accumulator) mergeEnd(smd core.SegmentMD) {
	if !a.Merge {
		a.Segments = append(a.Segments, smd)
		return
	}
	compacted, ok := TryMerge(a.Segments[len(a.Segments)-1], smd, a.Density)
	if !ok {
		a.Segments = append(a.Segments, smd)
	} else {
		a.Segments[len(a.Segments)-1] = compacted
	}
}
