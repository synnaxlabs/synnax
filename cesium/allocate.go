package cesium

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/allocate"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/confluence"
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
