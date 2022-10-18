package cesium

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// Iterator iterates over a DB's segments in time order. Cesium's iterator is unique
// in that its position is not a single point in time, but rather a time range providing
// a sectioned 'view' over the DB's data. By doing so, the caller can iterate over arbitrary
// spans of time.
//
// An iterator can be used to iterate over one or more channel's data, although behavior
// when iterating over multiple Channels is somewhat undefined.
//
// kvPositionIterator is not safe for concurrent use, although it is safe to have multiple iterators
// open over the same data operating in different goroutines.
//
// It's important to not that the segments returned by the iterator may not exactly reflect
// the segments written to disk. THe DB may merge small segments together, or split large
// segments apart. The kvPositionIterator may also return partial segments that fit within the
// current view.
type Iterator interface {
	// SeekFirst seeks the iterator to the first segment in the range and sets the
	// iterator's view to a single point in time at the beginning of the segment. The
	// view is guaranteed to be within the iterator's bounds. It does NOT read any
	// segments from disk, and the iterator will not be considered valid until the first
	// call to a non-seeking method (e.g. Next, Prev, NextSpan, PrevSpan, SetRange, etc).
	SeekFirst() bool
	// SeekLast seeks the iterator to the last segment in the range and sets the iterator's
	// view to a single point in time at the end of the segment. The view is guaranteed
	// to be within the iterator's bounds. It returns true if the iterator is pointing
	// to the end of a valid segment, and false otherwise. It does NOT read any segments
	// from disk, and the iterator will be considered invalid until the first call to a
	// non-seeking method (e.g. Next, Prev, NextSpan, PrevSpan, SetRange, etc).
	SeekLast() bool
	// SeekLE seeks the kvPositionIterator to the beginning of the first segment with a timestamp
	// less than the provided timestamp and sets the iterator's view to a single point
	// in time at the beginning of the segment. The view is guaranteed to be within the
	// iterator's bounds. It returns true if the kvPositionIterator is pointing to the beginning of
	// a valid segment. It does NOT read any segments from disk, and the iterator will
	// be considered invalid until the first call to a non-seeking method (e.g. Next,
	// Prev, NextSpan, PrevSpan, SetRange, etc).
	SeekLE(ts telem.TimeStamp) bool
	// SeekGE seeks the kvPositionIterator to the end of the first segment with a timestamp greater
	// than the provided timestamp and set's the iterators view to a single point in time
	// at the end of the segment. The view is guaranteed to be within the iterator's
	// bounds. It returns true if the kvPositionIterator is pointing to the end of a valid segment.
	// It does NOT read any segments from disk, and the iterator will be considered invalid
	// until the first call to a non-seeking method (e.g. Next, Prev, NextSpan, PrevSpan,
	// SetRange, etc).
	SeekGE(ts telem.TimeStamp) bool
	// Next reads the next segment in the iterator. It returns true if the segment was
	// successfully read, and false otherwise. It sets the iterator's view to the time
	// range occupied by the segment. The view is guaranteed to be within the iterator's
	// bounds.
	Next(span telem.TimeSpan) bool
	// Prev reads the previous segment in the iterator. It returns true if the segment
	// was successfully read, and false otherwise. It sets the iterator's view to the
	// time range occupied by the segment. the view is guaranteed to be within the
	// iterator's bounds
	Prev(span telem.TimeSpan) bool
	// View returns the current range of values the kvPositionIterator has a 'view' of.  This view
	// represents the range of data range segments currently held in Value.
	View() telem.TimeRange
	// Close closes the iterator, ensuring that all in-progress segment reads have
	// completed. An iterator MUST be closed after use, or else it will leak resources.
	Close() error
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
}

// IteratorResponseType is the type of the response an iterator will return.
type IteratorResponseType uint8

const (
	// IteratorResponseTypeAck is a response that indicates that an iteration request
	// has completed successfully.
	IteratorResponseTypeAck IteratorResponseType = iota + 1
	// IteratorResponseTypeData is a response that indicates that an iteration request
	// returned data.
	IteratorResponseTypeData
)

// IteratorCommand is an enumeration of commands that can be sent to an iterator.
type IteratorCommand uint8

const (
	// IterNext represents a call to Iterator.Next.
	IterNext IteratorCommand = iota + 1
	// IterPrev represents a call to Iterator.Prev.
	IterPrev
	// IterSeekFirst represents a call to Iterator.SeekFirst.
	IterSeekFirst
	// IterSeekLast represents a call to Iterator.SeekLast.
	IterSeekLast
	// IterSeekLE represents a call to Iterator.SeekLE.
	IterSeekLE
	// IterSeekGE represents a call to Iterator.SeekGE.
	IterSeekGE
	// IterValid represents a call to Iterator.Valid.
	IterValid
	// IterError represents a call to Iterator.Close.
	IterError
)

// HasOps returns true if the IteratorCommand has any associated on disk operations.
func (i IteratorCommand) HasOps() bool { return i <= IterPrev }

// IteratorRequest is issued to an StreamIterator asking it to read data from a DB.
//
//go:generate stringer -type=IteratorCommand
type IteratorRequest struct {
	// Command is the command to execute. See kvPositionIterator documentation for the behavior
	// of specific commands.
	Command IteratorCommand
	// The following fields are only used for commands that require them.
	Span   telem.TimeSpan
	Target telem.TimeStamp
}

// IteratorResponse is a response containing segments satisfying a RetrieveP Query as
// well as any errors encountered during the retrieval.
type IteratorResponse struct {
	// Variant is the type of response issued.
	Variant IteratorResponseType
	// Counter is  incremented for each request issued to the StreamIterator. The
	// first request will have a counter value of 1.
	Counter int
	// Command is only defined when the response type is IteratorResponseTypeAck.
	// It indicates the command that was acknowledged.
	Command IteratorCommand
	// Ack is only valid when the response type is IteratorResponseTypeAck. It
	// indicates whether the command was successfully processed.
	Ack bool
	// Err is only set an IterError command is issued.
	Err error
	// Segments is only set when the response type is IteratorResponseTypeData. It
	// contains the segments that were read.
	Segments []Segment
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
	return i.exec(IteratorRequest{Command: IterSeekLE, Target: ts})
}

// SeekGE implements Iterator.
func (i *iterator) SeekGE(ts telem.TimeStamp) bool {
	return i.exec(IteratorRequest{Command: IterSeekGE, Target: ts})
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

// Value implements Iterator.
func (i *iterator) Value() []Segment { return i.value }

// View implements Iterator.
func (i *iterator) View() telem.TimeRange { return i.wrapped.mdIter.View() }

// Close implements Iterator.
func (i *iterator) Close() error {
	i.inlet.Close()
	if err := i.wg.Wait(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (i *iterator) exec(req IteratorRequest) bool {
	ok, _ := i.execErr(req)
	return ok
}

func (i *iterator) execErr(req IteratorRequest) (bool, error) {
	i.value = nil
	i.inlet.Inlet() <- req
	for res := range i.outlet.Outlet() {
		if res.Variant == IteratorResponseTypeAck {
			return res.Ack, res.Err
		}
		i.value = append(i.value, res.Segments...)
	}
	panic("unreachable")
}
