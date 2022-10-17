package cesium

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/allocate"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/file"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/confluence"
)

type Allocator struct {
	confluence.LinearTransform[WriteRequest, segment.Segment]
	allocate.Allocator[channel.Key, file.Key]
}

func NewAllocator(alloc allocate.Allocator[channel.Key, file.Key]) *Allocator {
	a := &Allocator{Allocator: alloc}
	a.Transform = a.allocate
	return a
}

func (a *Allocator) allocate(ctx context.Context, w WriteRequest) (segment.Segment, bool, error) {
	items := make([]allocate.Item[channel.Key], len(w.Segments))
	for i, item := range w.Segments {
		items[i] = item.AItem()
	}
	f := a.Allocator.Allocate(items...)
	for i, item := range w.Segments {
		item.MD.FileKey = f[i]
		w.Segments[i] = item
	}
	for _, item := range w.Segments {
		a.Out.Inlet() <- item
	}
	return segment.Segment{}, false, nil
}
