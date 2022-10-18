package accumulate

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/telem"
)

// TryMerge attempts to merge the given segment with the last segment in the accumulator.
// In order to merge, the segments must be contiguous in position space and on disk.
// If the segments are merged, the merged segment is returned. Otherwise, TryMerge
// returns false.
func TryMerge(sa, sb core.SegmentMD, density telem.Density) (core.SegmentMD, bool) {
	s1, s2 := sa, sb
	if sb.Alignment.Before(sa.Alignment) {
		s1, s2 = sb, sa
	}
	// Check if the segments are contiguous in position and
	posContiguous := s1.End(density) == (s2.Alignment)
	diskContiguous := s1.EndOffset() == s2.Offset && s1.FileKey == s2.FileKey
	if !posContiguous || !diskContiguous {
		return core.SegmentMD{}, false
	}
	return core.SegmentMD{
		ChannelKey: s1.ChannelKey,
		Alignment:  s1.Alignment,
		Offset:     s1.Offset,
		Size:       s1.Size + s2.Size,
		FileKey:    s1.FileKey,
	}, true
}
