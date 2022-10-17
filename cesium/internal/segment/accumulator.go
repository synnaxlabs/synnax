package segment

import (
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

// Accumulator accumulates segments that satisfy a given range.
// The Segments in an accumulator are always sorted by Alignment position.
type Accumulator struct {
	Density  telem.Density
	Bounds   position.Range
	Compact  bool
	Slice    bool
	Segments []MD
}

func (a *Accumulator) Reset(bounds position.Range) {
	a.Bounds = bounds
	a.Segments = nil
}

func (a *Accumulator) Accumulate(s MD) bool {
	rng := s.Range(a.Density)
	if !rng.OverlapsWith(a.Bounds) {
		return false
	}
	s = a.slice(s)
	if len(a.Segments) == 0 {
		a.Segments = append(a.Segments, s)
	} else if s.Start.Before(a.Segments[0].Start) {
		a.compactStart(s)
	} else {
		a.compactEnd(s)
	}
	return true
}

func (a *Accumulator) Satisfied() bool {
	// get the start of the first segment and the end of the last segment
	// if the range between them is equal to the bounds, then we are satisfied
	if len(a.Segments) == 0 {
		return false
	}
	start := a.Segments[0].Start
	end := a.Segments[len(a.Segments)-1].End(a.Density)
	return a.Bounds.Equals(position.Range{Start: start, End: end})
}

func (a *Accumulator) PartiallySatisfied() bool { return len(a.Segments) > 0 }

func (a *Accumulator) slice(s MD) MD {
	if a.Slice {
		s = Slicer{Density: a.Density, Range: a.Bounds}.Slice(s)
	}
	return s
}

func (a *Accumulator) compactStart(s MD) {
	if !a.Compact {
		a.Segments = append([]MD{s}, a.Segments...)
		return
	}
	compacted, ok := Compactor{Density: a.Density}.Compact(s, a.Segments[0])
	if !ok {
		a.Segments = append([]MD{s}, a.Segments...)
	} else {
		a.Segments[0] = compacted
	}
}

func (a *Accumulator) compactEnd(s MD) {
	if !a.Compact {
		a.Segments = append(a.Segments, s)
		return
	}
	compacted, ok := Compactor{Density: a.Density}.Compact(a.Segments[len(a.Segments)-1], s)
	if !ok {
		a.Segments = append(a.Segments, s)
	} else {
		a.Segments[len(a.Segments)-1] = compacted
	}
}
