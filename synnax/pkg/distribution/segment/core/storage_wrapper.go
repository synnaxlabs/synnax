package core

import (
	"github.com/arya-analytics/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
)

// StorageWrapper wraps slices of storage.Segment into slices of Segment by
// adding the appropriate host information.
type StorageWrapper struct {
	Host core.NodeID
}

// Wrap converts a slice of cesium.segment into a slice of Segment.
func (cw *StorageWrapper) Wrap(segments []cesium.Segment) []Segment {
	wrapped := make([]Segment, len(segments))
	for i, seg := range segments {
		wrapped[i] = Segment{
			ChannelKey: channel.NewKey(cw.Host, seg.ChannelKey),
			Segment:    seg,
		}
	}
	return wrapped
}
