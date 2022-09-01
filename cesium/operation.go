package cesium

import (
	"context"
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/cesium/internal/kv"
	"github.com/arya-analytics/cesium/internal/operation"
	"github.com/arya-analytics/cesium/internal/segment"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/telem"
	"go.uber.org/zap"
	"sync"
)

// retrieveOperationUnary executes a single segment read on a file.
type retrieveOperationUnary struct {
	ctx       context.Context
	seg       *segment.Sugared
	dataRead  alamos.Duration
	wg        *sync.WaitGroup
	logger    *zap.Logger
	errC      chan<- error
	responses *confluence.AbstractUnarySource[IteratorResponse]
}

func (rou retrieveOperationUnary) FileKey() core.FileKey { return rou.seg.FileKey() }

func (rou retrieveOperationUnary) WriteError(err error) { rou.errC <- err }

func (rou retrieveOperationUnary) maybeWriteError(err error) {
	if err != nil {
		rou.errC <- err
	}
}

func (rou retrieveOperationUnary) offset() telem.Offset { return rou.seg.BoundedOffset() }

// Exec implements persist.Operation.
func (rou retrieveOperationUnary) Exec(f core.File) {
	defer rou.wg.Done()
	s := rou.dataRead.Stopwatch()
	s.Start()
	rou.maybeWriteError(rou.seg.ReadDataFrom(f))
	s.Stop()
	rou.responses.Out.Inlet() <- IteratorResponse{
		Variant:  DataResponse,
		Segments: []segment.Segment{rou.seg.Segment()},
	}
}

// retrieveOperationSet represents a set of retrieveOperations to execute together.
// The operations in the set are assumed to be ordered by file offset. All operations
// should have the same file key.
type retrieveOperationSet struct {
	operation.Set[core.FileKey, retrieveOperationUnary]
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

// FileKey implements createOperation.
func (cou createOperationUnary) FileKey() core.FileKey { return cou.seg.FileKey() }

// ChannelKey implements createOperation.
func (cou createOperationUnary) ChannelKey() channel.Key { return cou.seg.ChannelKey() }

// WriteError implements createOperation.
func (cou createOperationUnary) WriteError(err error) {
	cou.responses.Out.Inlet() <- CreateResponse{Error: err}
}

func (cou createOperationUnary) maybeWriteError(err error) {
	if err != nil {
		cou.WriteError(err)
	}
}

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
	defer cou.wg.Done()
	totalFlush := cou.metrics.totalFlush.Stopwatch()
	totalFlush.Start()
	defer totalFlush.Stop()
	cou.maybeWriteError(cou.seg.WriteDataTo(f))
	cou.metrics.dataWrite.Record(totalFlush.Elapsed())
	ks := cou.metrics.headerFlush.Stopwatch()
	ks.Start()
	cou.maybeWriteError(cou.kv.Set(cou.seg.Header()))
	ks.Stop()
}

type createOperationSet struct {
	operation.Set[core.FileKey, createOperationUnary]
}
