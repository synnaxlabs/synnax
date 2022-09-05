package cesium

import (
	"context"
	"github.com/arya-analytics/cesium/internal/allocate"
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/x/confluence"
)

type allocator struct {
	confluence.LinearTransform[[]writeOperation, []writeOperation]
	allocate.Allocator[channel.Key, core.FileKey, writeOperation]
}

func newAllocator(counter *fileCounter, cfg allocate.Config) *allocator {
	a := &allocator{
		Allocator: allocate.New[channel.Key, core.FileKey, writeOperation](counter, cfg),
	}
	a.Transform = a.allocate
	return a
}

func (a *allocator) allocate(
	ctx context.Context,
	ops []writeOperation,
) ([]writeOperation, bool, error) {
	for i, fk := range a.Allocate(ops...) {
		ops[i].SetFileKey(fk)
	}
	return ops, true, nil
}
