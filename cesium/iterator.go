package cesium

import (
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/kv"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/telem"
	"github.com/samber/lo"
	"sync"
)

type StreamIterator interface {
	// Source is the outlet for the StreamIterator values. All segments read from disk
	// are piped to the Source outlet. StreamIterator should be the ONLY entity writing
	// to the Source outlet (StreamIterator.Close will close the Source outlet).
	confluence.Source[IteratorResponse]
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
	*confluence.AbstractUnarySource[IteratorResponse]
	confluence.EmptyFlow
	// parser converts segment metadata into executable operations on disk.
	parser *retrieveParser
	// executor is an Output where generated operations are piped for execution.
	executor confluence.Inlet[[]retrieveOperationUnary]
	// wg is used to track the completion status of the latest operations in the iterator.
	wg *sync.WaitGroup
	// sync is a flag indicating whether the iterator should wait for all operations to
	// for a particular command to complete before returning.
	sync bool
	// sendAcknowledgements is a flag indicating whether the iterator should send
	// acknowledgements through the response pipe. The caller can use these acknowledgements
	// to determine if they've received all data for a particular method.
	sendAcknowledgements bool
	// opErrC is used for operations to communicate execution errors back to the caller.
	opErrC chan error
	// _error is the error accumulated by the iterator.
	_error error
	// methodCounter is an internal counter that tracks the number of methods that have
	// been executed on the iterator. This counter is used to communicate method completion
	// acknowledgements through the response pipe.
	methodCounter int
}

func newIteratorFromRetrieve(r Retrieve) StreamIterator {
	responses := &confluence.AbstractUnarySource[IteratorResponse]{}
	wg := &sync.WaitGroup{}
	tr, err := telem.GetTimeRange(r)
	if err != nil {
		tr = TimeRangeMax
	}
	internal := kv.NewIterator(r.kve, tr, channel.GetKeys(r)...)
	errC := make(chan error)
	return &streamIterator{
		AbstractUnarySource: responses,
		internal:            internal,
		executor:            r.ops,
		parser: &retrieveParser{
			logger:    r.logger,
			responses: responses,
			wg:        wg,
			metrics:   r.metrics,
			errC:      errC,
		},
		wg:                   wg,
		opErrC:               errC,
		sync:                 getSync(r),
		sendAcknowledgements: getSendAcks(r),
	}
}

// Next implements StreamIterator.
func (i *streamIterator) Next() bool { return i.exec(i.internal.Next) }

// Prev implements StreamIterator.
func (i *streamIterator) Prev() bool { return i.exec(i.internal.Prev) }

// First implements StreamIterator.
func (i *streamIterator) First() bool { return i.exec(i.internal.First) }

// Last implements StreamIterator.
func (i *streamIterator) Last() bool { return i.exec(i.internal.Last) }

// NextSpan implements StreamIterator.
func (i *streamIterator) NextSpan(span TimeSpan) bool {
	return i.exec(func() bool { return i.internal.NextSpan(span) })
}

// PrevSpan implements StreamIterator.
func (i *streamIterator) PrevSpan(span TimeSpan) bool {
	return i.exec(func() bool { return i.internal.PrevSpan(span) })
}

// NextRange implements StreamIterator.
func (i *streamIterator) NextRange(tr TimeRange) bool {
	return i.exec(func() bool { return i.internal.SetRange(tr) })
}

// SeekFirst implements StreamIterator.
func (i *streamIterator) SeekFirst() bool {
	return i.exec(i.internal.SeekFirst)
}

// SeekLast implements StreamIterator.
func (i *streamIterator) SeekLast() bool {
	return i.exec(i.internal.SeekLast)
}

// SeekLT implements StreamIterator.
func (i *streamIterator) SeekLT(stamp TimeStamp) bool {
	return i.exec(func() bool { return i.internal.SeekLT(stamp) })
}

// SeekGE implements StreamIterator.
func (i *streamIterator) SeekGE(stamp TimeStamp) bool {
	return i.exec(func() bool { return i.internal.SeekGE(stamp) })
}

// Seek implements StreamIterator.
func (i *streamIterator) Seek(stamp TimeStamp) bool {
	return i.exec(func() bool { return i.internal.Seek(stamp) })
}

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
	i.maybeSendAck(err == nil, err)
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
	err := lo.Ternary(i.internal.Error() == nil, i.internal.Error(), i.error())
	i.maybeSendAck(err == nil, err)
	return err
}

func (i *streamIterator) exec(f func() bool) bool {
	// Check if this was a seek operation (i.e. we don't need to execute any operations).
	// Wrap this in a closure as we must evaluate after the operation executes.
	seek := func() bool {
		return i.internal.View().Span().IsZero()
	}
	if ok := f(); !ok || seek() {
		i.maybeSendAck(ok, nil)
		return ok
	}
	ops := i.parser.parse(i.internal.Ranges())
	i.wg.Add(len(ops))
	i.executor.Inlet() <- ops
	if i.sync {
		i.wg.Wait()
	}
	i.maybeSendAck(true, nil)
	return true
}

func (i *streamIterator) maybeSendAck(ack bool, err error) {
	i.methodCounter += 1
	if i.sendAcknowledgements {
		i.Out.Inlet() <- IteratorResponse{
			Variant: AckResponse,
			Ack:     ack,
			Counter: i.methodCounter,
			Err:     err,
		}
	}
}
