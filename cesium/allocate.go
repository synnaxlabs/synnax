package cesium

import (
	"context"
	"github.com/arya-analytics/cesium/internal/allocate"
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/x/confluence"
)

type allocator struct {
	confluence.LinearTransform[[]createOperationUnary, []createOperationUnary]
	allocate.Allocator[channel.Key, core.FileKey, createOperationUnary]
}

func newAllocator(counter *fileCounter, cfg allocate.Config) *allocator {
	a := &allocator{
		Allocator: allocate.New[channel.Key, core.FileKey, createOperationUnary](counter, cfg),
	}
	a.Transform = a.allocate
	return a
}

func (a *allocator) allocate(
	ctx context.Context,
	ops []createOperationUnary,
) ([]createOperationUnary, bool, error) {
	for i, fk := range a.Allocate(ops...) {
		ops[i].SetFileKey(fk)
	}
	return ops, true, nil
}
