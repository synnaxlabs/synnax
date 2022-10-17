package segment

import (
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

type Slicer struct {
	Density telem.Density
	Range   position.Range
}

func (slc Slicer) Slice(s MD) MD {
	rng := s.Range(slc.Density)
	if slc.Range.Start.After(rng.Start) {
		remove := slc.Density.Size(int(slc.Range.Start.Sub(position.Span(rng.Start))))
		s.Alignment = slc.Range.Start
		s.Offset = s.Offset + remove
		s.Size = s.Size - remove
	}
	if slc.Range.End.Before(rng.End) {
		s.Size = s.Size - slc.Density.Size(int(rng.End.Sub(position.Span(slc.Range.End))))
	}
	return s
}
