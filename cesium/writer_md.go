package cesium

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/lock"
	"github.com/synnaxlabs/x/signal"
)

// mdWriter is a writer that writes metadata to the DB.
type mdWriter struct {
	internal core.MDWriter
	keys     []ChannelKey
	lock     lock.Keys[ChannelKey]
	confluence.LinearTransform[[]core.SugaredSegment, WriteResponse]
}

func newMDWriter(writer core.MDWriter, keys []ChannelKey, lock lock.Keys[ChannelKey]) *mdWriter {
	md := &mdWriter{internal: writer, keys: keys, lock: lock}
	md.Transform = md.write
	return md
}

func (m *mdWriter) Flow(ctx signal.Context, opts ...confluence.Option) {
	m.LinearTransform.Flow(
		ctx,
		append(opts, confluence.Defer(func() {
			m.internal.Commit()
			m.lock.Unlock(m.keys...)
		}))...,
	)
}

func (m *mdWriter) write(
	ctx context.Context,
	segments []core.SugaredSegment,
) (WriteResponse, bool, error) {
	mds := make([]core.SegmentMD, len(segments))
	for i, seg := range segments {
		mds[i] = seg.SegmentMD
	}
	err := m.internal.Write(mds)
	return WriteResponse{Err: err}, err != nil, err
}
