package cesium

import (
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/kv"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/telem"
	"sync"
)

type StreamIterator interface {
	// Source is the outlet for the StreamIterator values. All segments read from disk
	// are piped to the Source outlet. StreamIterator should be the ONLY entity writing
	// to the Source outlet (StreamIterator.Close will close the Source outlet).
	confluence.Source[RetrieveResponse]
	// Next pipes the next segment in the StreamIterator to the Source outlet.
	// It returns true if the StreamIterator is pointing to a valid segment.
	Next() bool
	// Prev pipes the previous segment in the StreamIterator to the Source outlet.
	// It returns true if the StreamIterator is pointing to a valid segment.
	Prev() bool
	// First seeks to the first segment in the StreamIterator. Returns true
	// if the streamIterator is pointing to a valid segment.
	First() bool
	// Last seeks to the last segment in the StreamIterator. Returns true
	// if the streamIterator is pointing to a valid segment.
	Last() bool
	// NextSpan pipes all segments in the StreamIterator from the current position to
	// the end of the span. It returns true if the streamIterator is pointing to a
	// valid segment. If span is TimeSpanMax, it will exhaust the streamIterator. If
	// span is TimeSpanZero, it won't do anything.
	NextSpan(span TimeSpan) bool
	// PrevSpan pipes all segments in the StreamIterator from the current
	PrevSpan(span TimeSpan) bool
	// NextRange seeks the StreamIterator to the start of the provided range and pipes all segments bound by it
	// to the Source outlet. It returns true if the streamIterator is pointing to a valid segment.
	// If range is TimeRangeMax, exhausts the StreamIterator. If range is TimeRangeZero, it won't do anything.
	NextRange(tr telem.TimeRange) bool
	// SeekFirst seeks the iterator to the first segment in the range.
	SeekFirst() bool
	// SeekLast seeks the iterator to the last segment in the range.
	SeekLast() bool
	// SeekLT seeks the StreamIterator to the first segment with a timestamp less than the provided timestamp.
	// It returns true if the StreamIterator is pointing to a valid segment.
	SeekLT(time TimeStamp) bool
	// SeekGE seeks the StreamIterator to the first segment with a timestamp greater than or equal to the provided timestamp.
	// It returns true if the StreamIterator is pointing to a valid segment.
	SeekGE(time TimeStamp) bool
	// View returns the current range of values the StreamIterator has a 'view' of.  This view represents the range of
	// segments most recently returned to the caller.
	View() TimeRange
	// Close closes the StreamIterator, ensuring that all in-progress segment reads complete before closing the Source outlet.
	Close() error
	// Valid returns true if the StreamIterator is pointing at valid segments.
	Valid() bool
	// Error returns any errors accumulated by the StreamIterator.
	Error() error
}

type streamIterator struct {
	// internal is the iterator that traverses segment metadata in key-value storage. It's essentially the 'brains'
	// behind the operations.
	internal kv.Iterator
	// UnarySource is where values from the iterator will be piped.
	*confluence.AbstractUnarySource[RetrieveResponse]
	confluence.EmptyFlow
	// parser converts segment metadata into executable operations on disk.
	parser *retrieveParser
	// executor is an Output where generated operations are piped for execution.
	executor confluence.Inlet[[]retrieveOperation]
	// wg is used to track the completion status of the latest operations in the iterator.
	wg     *sync.WaitGroup
	opErrC chan error
	_error error
}

func newIteratorFromRetrieve(r Retrieve) StreamIterator {
	responses := &confluence.AbstractUnarySource[RetrieveResponse]{}
	wg := &sync.WaitGroup{}
	internal := kv.NewIterator(r.kve, timeRange(r), channel.GetKeys(r)...)
	errC := make(chan error)
	return &streamIterator{
		internal: internal,
		executor: r.ops,
		parser: &retrieveParser{
			logger:    r.logger,
			responses: responses,
			wg:        wg,
			metrics:   r.metrics,
			errC:      errC,
		},
		wg:                  wg,
		AbstractUnarySource: responses,
		opErrC:              errC,
	}
}

// Next implements StreamIterator.
func (i *streamIterator) Next() bool {
	if !i.internal.Next() {
		return false
	}
	i.pipeOperations()
	return true
}

// Prev implements StreamIterator.
func (i *streamIterator) Prev() bool {
	if !i.internal.Prev() {
		return false
	}
	i.pipeOperations()
	return true
}

// First implements StreamIterator.
func (i *streamIterator) First() bool {
	if !i.internal.First() {
		return false
	}
	i.pipeOperations()
	return true
}

// Last implements StreamIterator.
func (i *streamIterator) Last() bool {
	if !i.internal.Last() {
		return false
	}
	i.pipeOperations()
	return true
}

// NextSpan implements StreamIterator.
func (i *streamIterator) NextSpan(span TimeSpan) bool {
	if !i.internal.NextSpan(span) {
		return false
	}
	i.pipeOperations()
	return true
}

// PrevSpan implements StreamIterator.
func (i *streamIterator) PrevSpan(span TimeSpan) bool {
	if !i.internal.PrevSpan(span) {
		return false
	}
	i.pipeOperations()
	return true
}

// NextRange implements StreamIterator.
func (i *streamIterator) NextRange(tr TimeRange) bool {
	if !i.internal.SetRange(tr) {
		return false
	}
	i.pipeOperations()
	return true
}

// SeekFirst implements StreamIterator.
func (i *streamIterator) SeekFirst() bool { return i.internal.SeekFirst() }

// SeekLast implements StreamIterator.
func (i *streamIterator) SeekLast() bool { return i.internal.SeekLast() }

// SeekLT implements StreamIterator.
func (i *streamIterator) SeekLT(stamp TimeStamp) bool { return i.internal.SeekLT(stamp) }

// SeekGE implements StreamIterator.
func (i *streamIterator) SeekGE(stamp TimeStamp) bool { return i.internal.SeekGE(stamp) }

// Seek implements StreamIterator.
func (i *streamIterator) Seek(stamp TimeStamp) bool { return i.internal.Seek(stamp) }

// View implements StreamIterator.
func (i *streamIterator) View() TimeRange { return i.internal.View() }

// Close implements StreamIterator.
func (i *streamIterator) Close() error {
	err := i.internal.Close()
	go func() {
		for _err := range i.opErrC {
			if _err != nil {
				err = _err
			}
		}
	}()
	i.wg.Wait()
	close(i.opErrC)
	close(i.Out.Inlet())
	return err
}

// Valid implements StreamIterator.
func (i *streamIterator) Valid() bool { return i.internal.Valid() && i.Error() == nil }

func (i *streamIterator) error() error {
	select {
	case err := <-i.opErrC:
		if err != nil {
			i._error = err
		}
	default:
	}
	return i._error
}

func (i *streamIterator) Error() error {
	if i.error() != nil {
		return i.error()
	}
	return i.internal.Error()
}

func (i *streamIterator) pipeOperations() {
	ops := i.parser.parse(i.internal.Ranges())
	if len(ops) == 0 {
		return
	}
	i.wg.Add(len(ops))
	i.executor.Inlet() <- ops
}
