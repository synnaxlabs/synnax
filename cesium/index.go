package cesium

import (
	"bytes"
	"context"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/telem"
)

type IndexingEngine struct {
	channelSvc core.ChannelService
	memIndexes map[channel.Key]index.Index
	storage    *storage.Storage
}

func (i *IndexingEngine) AcquireSearcher(key channel.Key) (index.Searcher, error) {
	var idx index.CompoundSearcher
	ch, err := i.channelSvc.Get(key)
	if err != nil {
		return nil, err
	}

	memIdx, ok := i.memIndexes[key]
	if !ok {
		idx = append(idx, memIdx)
	}

	if ch.Index != 0 {
		// TODO: open a reader index
	} else {
		idx = append(idx, index.RateSearcher(ch.Rate))
	}
	return idx, nil
}

func (i *IndexingEngine) AcquireWriter(key channel.Key) (index.Writer, error) {
	var idx index.CompoundWriter
	memIdx, ok := i.memIndexes[key]
	if !ok {
		idx = append(idx, memIdx)
	}
	return idx, nil
}

type IndexMaintainer struct {
	confluence.LinearTransform[segment.Segment, segment.Segment]
	Indexes map[channel.Key]index.Writer
}

func NewIndexMaintainer(indexes map[channel.Key]index.Writer) *IndexMaintainer {
	im := &IndexMaintainer{Indexes: indexes}
	im.Transform = im.transform
	return im
}

func (im *IndexMaintainer) transform(ctx context.Context, seg segment.Segment) (segment.Segment, bool, error) {
	var (
		r                            = bytes.NewReader(seg.Data)
		alignments                   = make([]index.Alignment, len(seg.Data)/int(telem.TimeStampDensity))
		b                            = make([]byte, telem.TimeStampDensity)
		i          position.Position = 0
		idx                          = im.Indexes[seg.ChannelKey]
	)
	for {
		var a index.Alignment
		if _, err := r.Read(b); err != nil {
			break
		}
		a.Stamp = index.DecodeTimeStamp(b)
		a.Pos = seg.MD.Alignment + i
		alignments = append(alignments)
	}
	err := idx.Write(alignments)
	return seg, err == nil, err
}

type IndexFilter struct {
	Channels map[ChannelKey]Channel
	confluence.Filter[segment.Segment]
}

func NewIndexFilter(channels map[ChannelKey]Channel) *IndexFilter {
	_if := &IndexFilter{
		Channels: channels,
	}
	_if.Apply = _if.filter
	return _if
}

func (i *IndexFilter) filter(ctx context.Context, seg segment.Segment) (bool, error) {
	ch, ok := i.Channels[seg.ChannelKey]
	if !ok {
		return false, NotFound
	}
	return !ch.IsIndex, nil
}

type IndexAligner struct {
	confluence.LinearTransform[segment.Segment, segment.Segment]
	Indexes map[channel.Key]index.Searcher
}

func NewIndexAligner(indexes map[channel.Key]index.Searcher) *IndexAligner {
	ia := &IndexAligner{Indexes: indexes}
	ia.Transform = ia.transform
	return ia
}

func (ia *IndexAligner) transform(ctx context.Context, seg segment.Segment) (segment.Segment, bool, error) {
	idx, ok := ia.Indexes[seg.ChannelKey]
	if !ok {
		return seg, false, nil
	}
	approx, err := idx.SearchP(seg.Start, position.Uncertain)
	if err != nil {
		return seg, false, err
	}
	seg.MD.Alignment = approx.Value()
	return seg, true, nil
}
