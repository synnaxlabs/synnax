package cesium

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

// indexMaintainer maintains a set of writable indexes using incoming segmented telemetry.
type indexMaintainer struct {
	Indexes map[core.ChannelKey]index.Writer
	confluence.UnarySink[[]core.SugaredSegment]
}

func newIndexMaintainer(indexes map[core.ChannelKey]index.Writer) *indexMaintainer {
	im := &indexMaintainer{Indexes: indexes}
	im.Sink = im.sink
	return im
}

func (im *indexMaintainer) sink(ctx context.Context, segments []core.SugaredSegment) error {
	for _, seg := range segments {
		idx, ok := im.Indexes[seg.ChannelKey]
		if !ok {
			continue
		}
		if err := im.maintain(idx, seg); err != nil {
			return err
		}
	}
	return nil
}

func (im *indexMaintainer) maintain(idx index.Writer, seg core.SugaredSegment) error {
	alignments, err := index.DecodeAlignments([]core.SugaredSegment{seg})
	if err != nil {
		return err
	}
	return idx.Write(alignments)
}

// indexMaintenanceRouter conditionally routes segments to an index maintainer.
type indexMaintenanceRouter struct {
	Channels core.ChannelReader
	confluence.Filter[[]core.SugaredSegment]
}

func newIndexMaintenanceRouter(channels core.ChannelReader) *indexMaintenanceRouter {
	_if := &indexMaintenanceRouter{Channels: channels}
	_if.Apply = _if.filter
	return _if
}

func (i *indexMaintenanceRouter) filter(
	ctx context.Context,
	segments []core.SugaredSegment,
) (bool, error) {
	var toMaintenance []core.SugaredSegment
	for _, s := range segments {
		ok, err := i.route(s)
		if err != nil {
			return false, err
		}
		if ok {
			toMaintenance = append(toMaintenance, s)
		}
	}
	if len(toMaintenance) > 0 {
		return true, signal.SendUnderContext(ctx, i.Rejects.Inlet(), toMaintenance)
	}
	return true, nil
}

func (i *indexMaintenanceRouter) route(seg core.SugaredSegment) (bool, error) {
	ch, err := i.Channels.GetChannel(seg.ChannelKey)
	if err != nil {
		return false, err
	}
	return ch.IsIndex, nil
}
