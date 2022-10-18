package cesium

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

// indexAligner aligns segments with start timestamps to
// positions in the index.
type indexAligner struct {
	indexes map[ChannelKey]index.Searcher
	confluence.LinearTransform[[]core.SugaredSegment, []core.SugaredSegment]
}

func newIndexAligner(indexes map[ChannelKey]index.Searcher) *indexAligner {
	ia := &indexAligner{indexes: indexes}
	ia.Transform = ia.align
	return ia
}

func (ia *indexAligner) Flow(ctx signal.Context, opts ...confluence.Option) {
	ia.LinearTransform.Flow(ctx, append(opts, confluence.Defer(func() {
		for _, idx := range ia.indexes {
			lo.Must0(idx.Release())
		}
	}))...)
}

func (ia *indexAligner) align(
	ctx context.Context,
	segments []core.SugaredSegment,
) ([]core.SugaredSegment, bool, error) {
	for i, seg := range segments {
		idx, ok := ia.indexes[seg.ChannelKey]
		if !ok {
			panic("index not found")
		}
		pos, err := idx.SearchP(seg.Start, position.Uncertain)
		if err != nil {
			return nil, false, err
		}
		seg.Alignment = pos.Value()
		segments[i] = seg
	}
	return segments, true, nil
}
