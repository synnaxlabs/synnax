package cesium

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// AutoSpan is a TimeSpan value that can be passed to Prev or Next to indicate
// that the iterator should automatically determine the span to use. This is useful
// for efficiently iterating over large spans of time quickly.
const AutoSpan = core.AutoTimeSpan

// Iterator iterates over a DB's telemetry in time order. Cesium's iterator is unique
// in that its position is not a single point in time, but rather a time range providing
// a sectioned 'view' over the DB's data. By doing so, the caller can iterate over arbitrary
// spans of time.
//
// An iterator can be used to iterate over one or more ch's data.
//
// Iterator is not safe for concurrent use, although it is safe to have multiple iterators
// open over the same data operating in different goroutines.
type Iterator interface {
	// SeekFirst seeks the iterator to the first segment in the range and sets the
	// iterator's view to a single point in time at the beginning of the segment.
	// Returns true if the iterator is pointing to a valid segment, and false otherwise.
	// SeekFirst reads no segments from disk, and the iterator will not be considered valid
	// until the first call to Next or Prev.
	SeekFirst() bool
	// SeekLast seeks the iterator to the last segment in the range and sets the iterator's
	// view to a single point in time at the end of the segment. It returns true if the
	// iterator is pointing to the end of a valid segment, and false otherwise. It does NOT
	// read any segments from disk, and the iterator will be considered invalid until the
	// first call to Next or Prev.
	SeekLast() bool
	// SeekLE seeks the iterator to the first sample whose timestamp is less than or equal
	// to the given timestamp, and sets the iterator's view to a single point at the
	// sample's timestamp. It returns true if the Iterator is pointing to the beginning of
	// a valid segment. It does NOT read any segments from disk, and the iterator will
	// be considered invalid until the first call to Next or Prev.
	SeekLE(ts telem.TimeStamp) bool
	// SeekGE seeks the iterator to the end of the first sample with a timestamp greater
	// than the provided timestamp and set's the iterators view to a single point in time
	// at the sample's timestamp. s It returns true if the kvPositionIterator is pointing
	// to a valid segment. It does NOT read any segments from disk, and the iterator will
	// be considered invalid until the first call to Next or Prev.
	SeekGE(ts telem.TimeStamp) bool
	// Next advances the iterator across the given span, reading any telemetry segments
	// it encounters. It returns true if the iterator is pointing to ANY valid segments,
	// and false otherwise. It sets the iterator's view to the time range it advanced across.
	//
	// If the provided span is AutoSpan, the iterator will automatically choose the
	// timespan to advance across. This is useful for quickly iterating over an iterator's
	// entire range.
	Next(span telem.TimeSpan) bool
	// Prev advances the iterator backwards across the given span, reading any telemetry
	// segments it encounters. It returns true if the iterator is pointing to ANY valid
	// segments, and false otherwise. It sets the iterator's view to the time range it
	// advanced across.
	//
	// If the provided span is AutoSpan, the iterator will automatically choose the
	// timespan to advance across. This is useful for quickly iterating over an iterator's
	// entire range.j
	Prev(span telem.TimeSpan) bool
	// View returns the current range of values the iterator has a 'view' of.
	// This view represents the range of telemetry segments currently held in Value.
	// View is guaranteed to be within the iterator's bounds.
	View() telem.TimeRange
	// Valid returns true if the iterator is pointing at any valid segments AND has
	// not accumulated an err.
	Valid() bool
	// Error returns any errors accumulated by the iterator.
	Error() error
	// Value returns all segments currently under the iterator's view. The segments
	// are guaranteed to represent ALL data in the view, and are guaranteed to NOT
	// include any data outside the view. The segments are NOT guaranteed to be contiguous,
	// however, and are also NOT guaranteed to be in time order.
	Value() []Segment
	// Close closes the iterator, ensuring that all in-progress segment reads have
	// completed. An iterator MUST be closed after use, or else it will leak resources.
	Close() error
}

type iterator struct {
	wrapped  *streamIterator
	inlet    confluence.Inlet[IteratorRequest]
	outlet   confluence.Outlet[IteratorResponse]
	value    []Segment
	shutdown context.CancelFunc
	wg       signal.WaitGroup
}

func wrapStreamIterator(wrap *streamIterator) Iterator {
	ctx, cancel := signal.Background()
	requests := confluence.NewStream[IteratorRequest](1)
	responses := confluence.NewStream[IteratorResponse](1)
	wrap.InFrom(requests)
	wrap.OutTo(responses)
	wrap.Flow(ctx)
	return &iterator{
		inlet:    requests,
		outlet:   responses,
		shutdown: cancel,
		wg:       ctx,
	}
}

// Next implements Iterator.
func (i *iterator) Next(span telem.TimeSpan) bool {
	return i.exec(IteratorRequest{Command: IterNext, Span: span})
}

// Prev implements Iterator.
func (i *iterator) Prev(span telem.TimeSpan) bool {
	return i.exec(IteratorRequest{Command: IterPrev, Span: span})
}

// SeekFirst implements Iterator.
func (i *iterator) SeekFirst() bool {
	return i.exec(IteratorRequest{Command: IterSeekFirst})
}

// SeekLast implements Iterator.
func (i *iterator) SeekLast() bool {
	return i.exec(IteratorRequest{Command: IterSeekLast})
}

// SeekLE implements Iterator.
func (i *iterator) SeekLE(ts telem.TimeStamp) bool {
	return i.exec(IteratorRequest{Command: IterSeekLE, Stamp: ts})
}

// SeekGE implements Iterator.
func (i *iterator) SeekGE(ts telem.TimeStamp) bool {
	return i.exec(IteratorRequest{Command: IterSeekGE, Stamp: ts})
}

// Error implements Iterator.
func (i *iterator) Error() error {
	_, err := i.execErr(IteratorRequest{Command: IterError})
	return err
}

// Valid implements Iterator.
func (i *iterator) Valid() bool {
	ok, _ := i.execErr(IteratorRequest{Command: IterValid})
	return ok
}

// SetBounds implements Iterator.
func (i *iterator) SetBounds(bounds telem.TimeRange) {
	i.exec(IteratorRequest{Command: IterSetBounds, Bounds: bounds})
}

// Value implements Iterator.
func (i *iterator) Value() []Segment { return i.value }

// View implements Iterator.
func (i *iterator) View() telem.TimeRange { return i.wrapped.mdIter.View() }

// Close implements Iterator.
func (i *iterator) Close() error {
	i.inlet.Close()
	err := i.wg.Wait()
	i.shutdown()
	return err
}

func (i *iterator) exec(req IteratorRequest) bool {
	ok, _ := i.execErr(req)
	return ok
}

func (i *iterator) execErr(req IteratorRequest) (bool, error) {
	i.value = nil
	i.inlet.Inlet() <- req
	for res := range i.outlet.Outlet() {
		if res.Variant == IteratorAckResponse {
			return res.Ack, res.Err
		}
		i.value = append(i.value, res.Segments...)
	}
	panic("unreachable")
}
