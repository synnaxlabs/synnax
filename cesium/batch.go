package cesium

import (
	"context"
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/x/confluence"
	"sort"
)

// |||||| RETRIEVE ||||||

// retrieveBatch implements a simple batching algorithm that optimizes disk IO for a
// set of retrieve operations (reads). The algorithm works as follows:
//
//  1. Batch the operations by the file they belong to.
//  2. Sort the operations by their offset in the file.
//
// The intent is to maximize sequential IO for a given set of operations.
type retrieveBatch struct {
	confluence.LinearTransform[[]readOperation, []retrieveOperationSet]
}

func newRetrieveBatch() *retrieveBatch {
	rb := &retrieveBatch{}
	rb.Transform = rb.batch
	return rb
}

func (rb *retrieveBatch) batch(
	ctx context.Context,
	ops []readOperation,
) ([]retrieveOperationSet, bool, error) {
	fileGrouped := make(map[core.FileKey]retrieveOperationSet)
	for _, op := range ops {
		fileGrouped[op.FileKey()] = retrieveOperationSet{Set: append(fileGrouped[op.FileKey()].Set, op)}
	}
	channelSorted := make([]retrieveOperationSet, 0, len(fileGrouped))
	for _, opSet := range fileGrouped {
		sort.Slice(opSet.Set, func(i, j int) bool {
			return opSet.Set[i].offset() < opSet.Set[j].offset()
		})
		channelSorted = append(channelSorted, opSet)
	}
	return channelSorted, true, nil
}

// |||||| CREATE ||||||

// createBatch implements a simple batching algorithm that optimizes disk IO for a
// set of create operations (writes). The algorithm works as follows:
//
//  1. Batch the operations by the file they belong to. The batching algorithm
//     has no influence on the files segments are allocated to. This is handled
//     by allocate.Allocator.
//  2. Sort the operations by their channel key. It's common to retrieve large,
//     contiguous ranges of data from an individual channel. By keeping segments
//     of the same channel together, we can minimize the number of disk seeks.
type createBatch struct {
	confluence.LinearTransform[[]writeOperation, []createOperationSet]
}

func newCreateBatch() *createBatch {
	cb := &createBatch{}
	cb.Transform = cb.batch
	return cb
}

func (cb *createBatch) batch(
	ctx context.Context,
	ops []writeOperation,
) ([]createOperationSet, bool, error) {
	fileGrouped := make(map[core.FileKey]createOperationSet)
	for _, op := range ops {
		fileGrouped[op.FileKey()] = createOperationSet{Set: append(fileGrouped[op.FileKey()].Set, op)}
	}
	channelSorted := make([]createOperationSet, 0, len(fileGrouped))
	for _, fileOps := range fileGrouped {
		sort.Slice(fileOps.Set, func(j, k int) bool {
			return fileOps.Set[j].
				ChannelKey() > fileOps.Set[k].ChannelKey()
		})
		channelSorted = append(channelSorted, fileOps)
	}
	return channelSorted, true, nil
}
