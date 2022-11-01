package cesium

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/confluence"
)

// storageWriter writes segment data to the DB.
type storageWriter struct {
	internal storage.Writer
	confluence.LinearTransform[[]core.SugaredSegment, []core.SugaredSegment]
}

func newStorageWriter(internal storage.Writer) *storageWriter {
	s := &storageWriter{internal: internal}
	s.Transform = s.transform
	return s
}

func (s *storageWriter) transform(
	ctx context.Context,
	segments []core.SugaredSegment,
) ([]core.SugaredSegment, bool, error) {
	mds, err := s.internal.Write(segments)
	if err != nil {
		return segments, false, err
	}
	for i, seg := range segments {
		seg.SegmentMD = mds[i]
		segments[i] = seg
	}
	return segments, true, nil
}
