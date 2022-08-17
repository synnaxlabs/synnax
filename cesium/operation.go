package cesium

import (
	"context"
	"github.com/arya-analytics/cesium/internal/allocate"
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/cesium/internal/kv"
	"github.com/arya-analytics/cesium/internal/operation"
	"github.com/arya-analytics/cesium/internal/segment"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/telem"
	"go.uber.org/zap"
	"io"
	"sync"
)

// |||||| RETRIEVE ||||||

type retrieveOperation interface {
	operation.Operation[core.FileKey]
	Offset() telem.Offset
}

// retrieveOperationUnary executes a single segment read on a file.
type retrieveOperationUnary struct {
	ctx       context.Context
	seg       *segment.Sugared
	dataRead  alamos.Duration
	wg        *sync.WaitGroup
	logger    *zap.Logger
	errC      chan<- error
	responses *confluence.AbstractUnarySource[RetrieveResponse]
}

// Context implements retrieveOperation.
func (rou retrieveOperationUnary) Context() context.Context { return rou.ctx }

// FileKey implements retrieveOperation.
func (rou retrieveOperationUnary) FileKey() core.FileKey { return rou.seg.FileKey() }

// WriteError implements retrieveOperation.
func (rou retrieveOperationUnary) WriteError(err error) { rou.errC <- err }

// Offset implements retrieveOperation.
func (rou retrieveOperationUnary) Offset() telem.Offset { return rou.seg.BoundedOffset() }

// Exec implements persist.Operation.
func (rou retrieveOperationUnary) Exec(f core.File) {
	if rou.wg != nil {
		defer rou.wg.Done()
	}
	s := rou.dataRead.Stopwatch()
	s.Start()
	err := rou.seg.ReadDataFrom(f)
	if err == io.EOF {
		panic("[cesium] unexpected EOF encountered while reading segment")
	}
	if err != nil {
		rou.WriteError(err)
	}
	s.Stop()
	rou.logger.Info("retrieved segment")
	rou.responses.Out.Inlet() <- RetrieveResponse{Segments: []segment.Segment{rou.seg.Segment()}}
}

// retrieveOperationSet represents a set of retrieveOperations to execute together.
// The operations in the set are assumed to be ordered by file offset. All operations
// should have the same file key.
type retrieveOperationSet struct {
	operation.Set[core.FileKey, retrieveOperation]
}

// Offset implements retrieveOperation.
func (ros retrieveOperationSet) Offset() telem.Offset { return ros.Set[0].Offset() }

// |||||| CREATE ||||||

type createOperation interface {
	operation.Operation[core.FileKey]
	allocate.Item[channel.Key]
	ChannelKey() channel.Key
	SetFileKey(fk core.FileKey)
}

// createOperationUnary executes a single segment write to a file.
type createOperationUnary struct {
	seg       *segment.Sugared
	ctx       context.Context
	logger    *zap.Logger
	metrics   createMetrics
	wg        *sync.WaitGroup
	kv        *kv.Header
	responses confluence.AbstractUnarySource[CreateResponse]
}

// Context implements createOperation.
func (cou createOperationUnary) Context() context.Context { return cou.ctx }

// FileKey implements createOperation.
func (cou createOperationUnary) FileKey() core.FileKey { return cou.seg.FileKey() }

// ChannelKey implements createOperation.
func (cou createOperationUnary) ChannelKey() channel.Key { return cou.seg.ChannelKey() }

// WriteError implements createOperation.
func (cou createOperationUnary) WriteError(err error) {
	cou.responses.Out.Inlet() <- CreateResponse{Error: err}
}

// BindWaitGroup implements createOperation.
func (cou createOperationUnary) BindWaitGroup(wg *sync.WaitGroup) { cou.wg = wg }

// Size implements createOperation.
func (cou createOperationUnary) Size() telem.Size { return cou.seg.UnboundedSize() }

// Key implements createOperation.
func (cou createOperationUnary) Key() channel.Key { return cou.ChannelKey() }

// SetFileKey implements createOperation.
func (cou createOperationUnary) SetFileKey(fk core.FileKey) { cou.seg.SetFileKey(fk) }

// Exec implements createOperation.
func (cou createOperationUnary) Exec(f core.File) {
	if cou.ctx.Err() != nil {
		return
	}
	if cou.wg != nil {
		defer cou.wg.Done()
	}
	totalFlush := cou.metrics.totalFlush.Stopwatch()
	totalFlush.Start()
	defer totalFlush.Stop()

	if err := cou.seg.WriteDataTo(f); err != nil {
		cou.WriteError(err)
		return
	}
	cou.metrics.dataWrite.Record(totalFlush.Elapsed())

	ks := cou.metrics.headerFlush.Stopwatch()
	ks.Start()
	if err := cou.kv.Set(cou.seg.Header()); err != nil {
		cou.WriteError(err)
		return
	}
	ks.Stop()
}

type createOperationSet struct {
	operation.Set[core.FileKey, createOperation]
}

// Size implements createOperation.
func (cos createOperationSet) Size() (total telem.Size) {
	for _, op := range cos.Set {
		total += op.Size()
	}
	return total
}

// Key implements createOperation.
func (cos createOperationSet) Key() channel.Key { return cos.ChannelKey() }

// ChannelKey implements createOperation.
func (cos createOperationSet) ChannelKey() channel.Key { return cos.Set[0].ChannelKey() }

// SetFileKey implements createOperation.
func (cos createOperationSet) SetFileKey(fk core.FileKey) {
	for _, op := range cos.Set {
		op.SetFileKey(fk)
	}
}
