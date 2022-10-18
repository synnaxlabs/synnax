package accumulate

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

// Slice slices the given segment to fit within the given bounds.
func Slice(smd core.SegmentMD, density telem.Density, bounds position.Range) core.SegmentMD {
	rng := smd.Range(density)
	if bounds.Start.After(rng.Start) {
		remove := density.Size(int(bounds.Start.Sub(position.Span(rng.Start))))
		smd.Alignment = bounds.Start
		smd.Offset = smd.Offset + remove
		smd.Size = smd.Size - remove
	}
	if bounds.End.Before(rng.End) {
		smd.Size = smd.Size - density.Size(int(rng.End.Sub(position.Span(bounds.End))))
	}
	return smd
}
