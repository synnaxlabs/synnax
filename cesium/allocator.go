package cesium

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/allocate"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/telem"
)

type allocator struct {
	alloc allocate.Allocator[ChannelKey, core.FileKey]
	confluence.LinearTransform[WriteRequest, []core.SugaredSegment]
}

func newAllocator(alloc allocate.Allocator[ChannelKey, core.FileKey]) *allocator {
	a := &allocator{alloc: alloc}
	a.Transform = a.allocate
	return a
}

func (a *allocator) allocate(
	ctx context.Context,
	req WriteRequest,
) ([]core.SugaredSegment, bool, error) {
	var (
		segments   = make([]core.SugaredSegment, len(req.Segments))
		toAllocate = make([]allocate.Item[ChannelKey], len(req.Segments))
	)
	for i, req := range req.Segments {
		toAllocate[i] = allocate.Item[ChannelKey]{
			Key:  req.ChannelKey,
			Size: telem.Size(len(req.Data)),
		}
	}
	fileKeys, err := a.alloc.Allocate(toAllocate...)
	if err != nil {
		return nil, false, err
	}
	for i, reqSeg := range req.Segments {
		segments[i] = core.SugaredSegment{
			Data: reqSeg.Data,
			SegmentMD: core.SegmentMD{
				Start:      reqSeg.Start,
				ChannelKey: reqSeg.ChannelKey,
				FileKey:    fileKeys[i],
				Size:       telem.Size(len(reqSeg.Data)),
			},
		}
	}
	return segments, true, nil

}
