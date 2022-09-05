package cesium

import (
	"context"
	"github.com/arya-analytics/cesium/internal/kv"
	"github.com/arya-analytics/x/confluence"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"sync"
)

// Iterator iterates over a DB's segments in time order. Cesium's iterator is unique
// in that its position is not a single point in time, but rather a time range providing
// a sectioned 'view' over the DB's data. By doing so, the caller can iterate over arbitrary
// time ranges.
//
// An iterator can be used to iterate over one or more channel's data, although behavior
// when iterating over multiple channels is somewhat undefined.
//
// Iterator is not safe for concurrent use, although it is safe to have multiple iterators
// open over the same data operating in different goroutines.
type Iterator interface {
	// Next reads the next segment in the iterator. It returns true if the segment was
	// successfully read, and false otherwise. It sets the iterator's view to the time
	// range occupied by the segment. The view is guaranteed to be within the iterator's
	// bounds
	Next() bool
	// Prev reads the previous segment in the iterator. It returns true if the segment
	// was successfully read, and false otherwise. It sets the iterator's view to the
	// time range occupied by the segment. the view is guaranteed to be within the
	// iterator's bounds
	Prev() bool
	// First seeks to and reads the first segment in the iterator. It returns true if
	// the Segment was successfully read, and false otherwise. It sets the iterator's
	// view to the time range occupied by the segment. the view is guaranteed to be
	// within the iterator's bounds.
	First() bool
	// Last seeks to and reads the first segment in the iterator. It returns true if
	// the segment was successfully read, and false otherwise. It sets the iterator's
	// view to the time range occupied by the segment. The view is guaranteed to be
	// within the iterator's bounds.
	Last() bool
	// NextSpan reads all segments from the end of the current view to the end of the
	// provided span. It returns true if any segments were read. The segments read by
	// NextSpan are guaranteed to represent all data in the view, and are guaranteed to
	// NOT include any data outside the span. The segments read are NOT guaranteed
	// to be contiguous, however, and are also NOT guaranteed to be in time order.
	NextSpan(span telem.TimeSpan) bool
	// PrevSpan reads all segments from the beginning of the current view to the beginning
	// of the provided span. It returns true if any segments were read. The segments read
	// by PrevSpan are guaranteed to represent all data in the view, and are guaranteed
	// to NOT include any data outside the span. The segments read are NOT guaranteed to
	// be contiguous, however, and are also NOT guaranteed to be in any particular order.
	PrevSpan(span telem.TimeSpan) bool
	// ReadView reads all segments in the provided view. It returns true if any segments
	// were read. The segments read by SetRange are guaranteed to represent all data in
	// the view, and are guaranteed to NOT include any data outside the range. The
	// segments read are NOT guaranteed to be contiguous, however, and are also NOT
	// guaranteed to be in any particular order.
	ReadView(tr telem.TimeRange) bool
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
	// SeekLT seeks the Iterator to the beginning of the first segment with a timestamp
	// less than the provided timestamp and sets the iterator's view to a single point
	// in time at the beginning of the segment. The view is guaranteed to be within the
	// iterator's bounds. It returns true if the Iterator is pointing to the beginning of
	// a valid segment. It does NOT read any segments from disk, and the iterator will
	// be considered invalid until the first call to a non-seeking method (e.g. Next,
	// Prev, NextSpan, PrevSpan, SetRange, etc).
	SeekLT(ts telem.TimeStamp) bool
	// SeekGE seeks the Iterator to the end of the first segment with a timestamp greater
	// than the provided timestamp and set's the iterators view to a single point in time
	// at the end of the segment. The view is guaranteed to be within the iterator's
	// bounds. It returns true if the Iterator is pointing to the end of a valid segment.
	// It does NOT read any segments from disk, and the iterator will be considered invalid
	// until the first call to a non-seeking method (e.g. Next, Prev, NextSpan, PrevSpan,
	// SetRange, etc).
	SeekGE(ts telem.TimeStamp) bool
	// View returns the current range of values the Iterator has a 'view' of.  This view
	// represents the range of data range segments currently held in Value.
	View() telem.TimeRange
	// Close closes the iterator, ensuring that all in-progress segment reads have
	// completed. An iterator MUST be closed after use, or else it will leak resources.
	Close() error
	// Valid returns true if the iterator is pointing at any valid segments AND has
	// not accumulated an error.
	Valid() bool
	// Error returns any errors accumulated by the iterator.
	Error() error
	// Value returns all segments currently under the iterator's view. The segments
	// are guaranteed to represent ALL data in the view, and are guaranteed to NOT
	// include any data outside the view. The segments are NOT guaranteed to be contiguous,
	// however, and are also NOT guaranteed to be in time order.
	Value() []Segment
}

// StreamIterator provides a streaming interface for iterating over a DB's segments
// in time order. StreamIterator provides the underlying functionality for Iterator,
// and has almost exactly the same semantics. The streaming interface is exposed
// as a confluence segment that can accept one input stream and one output stream.
//
// To read segments issue an IteratorRequest to the StreamIterator's inlet. The
// StreamIterator will respond by sending one or more IteratorResponse messages to
// the outlet. All responses containing segment data will have a type of
// IteratorResponseTypeData and will contain one or more segments. The last response
// for any request will have a type of IteratorResponseTypeAck and will contain
// the name of the command that was acknowledged, and incremented sequence number,
// and ack boolean indicating whether the command was successfully processed.
//
// To close the StreamIterator, simply close the inlet. The StreamIterator will ensure
// that all in-progress requests have been served before closing the outlet. The
// StreamIterator will return any accumulated error through the signal context
// provided to Flow.
type StreamIterator = confluence.Segment[IteratorRequest, IteratorResponse]

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

// IteratorRequest is issued to an StreamIterator asking it to read data from a DB.
type IteratorRequest struct {
	// Command is the command to execute. See Iterator documentation for the behavior
	// of specific commands.
	Command IteratorCommand
	// The following fields are only used for commands that require them.
	Span  telem.TimeSpan
	Range telem.TimeRange
	Stamp telem.TimeStamp
}

// IteratorCommand is an enumeration of methods that can be called on a StreamIterator.
// They follow the same behavior as Iterator.
//
//go:generate stringer -type=IteratorCommand
type IteratorCommand uint8

// hasOps is a little utility telling us whether the command we've issued has any read
// operations associated with it.
func (i IteratorCommand) hasOps() bool { return i <= IterReadView }

const (
	// IterNext is an enumerated representation of Iterator.Next.
	IterNext IteratorCommand = iota + 1
	// IterPrev is an enumerated representation of Iterator.Prev.
	IterPrev
	// IterFirst is an enumerated representation of Iterator.First.
	IterFirst
	// IterLast is an enumerated representation of Iterator.Last.
	IterLast
	// IterNextSpan is an enumerated representation of Iterator.NextSpan.
	IterNextSpan
	// IterPrevSpan is an enumerated representation of Iterator.PrevSpan.
	IterPrevSpan
	// IterReadView is an enumerated representation of Iterator.ReadView.
	IterReadView
	// IterValid is an enumerated representation of Iterator.Valid.
	IterValid
	// IterError is an enumerated representation of Iterator.Error.
	IterError
	// IterSeekFirst is an enumerated representation of Iterator.SeekFirst.
	IterSeekFirst
	// IterSeekLast is an enumerated representation of Iterator.SeekLast.
	IterSeekLast
	// IterSeekLT is an enumerated representation of Iterator.SeekLT.
	IterSeekLT
	// IterSeekGE is an enumerated representation of Iterator.SeekGE.
	IterSeekGE
)

// IteratorResponse is a response containing segments satisfying a Retrieve Query as
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
	internal streamIterator
	inlet    confluence.Inlet[IteratorRequest]
	outlet   confluence.Outlet[IteratorResponse]
	value    []IteratorResponse
	shutdown context.CancelFunc
	wg       signal.WaitGroup
}

func wrapStreamIterator(stream *streamIterator) *iterator {
	ctx, cancel := signal.Background()
	requests := confluence.NewStream[IteratorRequest]()
	responses := confluence.NewStream[IteratorResponse]()
	stream.InFrom(requests)
	stream.OutTo(responses)
	stream.Flow(ctx)

	return &iterator{
		internal: *stream,
		inlet:    requests,
		outlet:   responses,
		shutdown: cancel,
		wg:       ctx,
	}
}

// Next implements Iterator.
func (i *iterator) Next() bool {
	i.resetValue()
	return i.exec(IteratorRequest{Command: IterNext})
}

// Prev implements Iterator.
func (i *iterator) Prev() bool {
	i.resetValue()
	return i.exec(IteratorRequest{Command: IterPrev})
}

// First implements Iterator.
func (i *iterator) First() bool {
	i.resetValue()
	return i.exec(IteratorRequest{Command: IterFirst})
}

// Last implements Iterator.
func (i *iterator) Last() bool {
	i.resetValue()
	return i.exec(IteratorRequest{Command: IterLast})
}

// NextSpan implements Iterator.
func (i *iterator) NextSpan(span telem.TimeSpan) bool {
	i.resetValue()
	return i.exec(IteratorRequest{Command: IterNextSpan, Span: span})
}

// PrevSpan implements Iterator.
func (i *iterator) PrevSpan(span telem.TimeSpan) bool {
	i.resetValue()
	return i.exec(IteratorRequest{Command: IterPrevSpan, Span: span})
}

// ReadView implements Iterator.
func (i *iterator) ReadView(tr telem.TimeRange) bool {
	i.resetValue()
	return i.exec(IteratorRequest{Command: IterReadView, Range: tr})
}

// SeekFirst implements Iterator.
func (i *iterator) SeekFirst() bool {
	i.resetValue()
	return i.exec(IteratorRequest{Command: IterSeekFirst})
}

// SeekLast implements Iterator.
func (i *iterator) SeekLast() bool {
	i.resetValue()
	return i.exec(IteratorRequest{Command: IterSeekLast})
}

// SeekLT implements Iterator.
func (i *iterator) SeekLT(ts telem.TimeStamp) bool {
	i.resetValue()
	return i.exec(IteratorRequest{Command: IterSeekLT, Stamp: ts})
}

// SeekGE implements Iterator.
func (i *iterator) SeekGE(time telem.TimeStamp) bool {
	i.resetValue()
	return i.exec(IteratorRequest{Command: IterSeekGE, Stamp: time})
}

// View implements Iterator.
func (i *iterator) View() telem.TimeRange { return i.internal.internal.View() }

// Close implements Iterator.
func (i *iterator) Close() error {
	i.resetValue()
	i.inlet.Close()
	return i.wg.Wait()
}

// Error implements Iterator.
func (i *iterator) Error() error {
	_, err := i.execErr(IteratorRequest{Command: IterError})
	return err
}

// Valid implements Iterator.
func (i *iterator) Valid() bool {
	return i.exec(IteratorRequest{Command: IterValid})
}

// Value implements Iterator.
func (i *iterator) Value() []Segment {
	var segments []Segment
	for _, resp := range i.value {
		segments = append(segments, resp.Segments...)
	}
	return segments
}

func (i *iterator) exec(req IteratorRequest) bool {
	ok, _ := i.execErr(req)
	return ok
}

func (i *iterator) execErr(req IteratorRequest) (bool, error) {
	i.inlet.Inlet() <- req
	for res := range i.outlet.Outlet() {
		if res.Variant == IteratorResponseTypeAck {
			return res.Ack, res.Err
		}
		i.value = append(i.value, res)
	}
	return false, nil
}

func (i *iterator) resetValue() { i.value = nil }

type streamIterator struct {
	*confluence.AbstractUnarySource[IteratorResponse]
	confluence.UnarySink[IteratorRequest]
	internal        kv.Iterator
	parser          *retrieveParser
	operations      confluence.Inlet[[]retrieveOperationUnary]
	wg              *sync.WaitGroup
	operationErrors chan error
	commandCounter  int
	_error          error
}

func newStreamIterator(
	tr telem.TimeRange,
	keys []ChannelKey,
	operations confluence.Inlet[[]retrieveOperationUnary],
	logger *zap.Logger,
	metrics retrieveMetrics,
	kve kvx.DB,
) (*streamIterator, error) {
	responses := &confluence.AbstractUnarySource[IteratorResponse]{}
	wg := &sync.WaitGroup{}
	internal, err := kv.NewIterator(kve, tr, keys...)
	if err != nil {
		return nil, err
	}
	errC := make(chan error)
	s := &streamIterator{
		AbstractUnarySource: responses,
		internal:            internal,
		operations:          operations,
		parser: &retrieveParser{
			logger:    logger,
			responses: responses,
			wg:        wg,
			metrics:   metrics,
			errC:      errC,
		},
		wg:              wg,
		operationErrors: errC,
	}
	return s, nil
}

// Flow implements confluence.Flow.
func (s *streamIterator) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-s.In.Outlet():
				if !ok {
					return s.close()
				}
				s.exec(req)
			}
		}
	}, o.Signal...)
}

func (s *streamIterator) exec(req IteratorRequest) {
	ok, err := s.runCmd(req)
	if !ok || !req.Command.hasOps() {
		s.sendAck(req, ok, err)
		return
	}
	ops := s.parser.parse(s.internal.Ranges())
	s.wg.Add(len(ops))
	s.operations.Inlet() <- ops
	s.wg.Wait()
	s.sendAck(req, true, nil)
}

func (s *streamIterator) sendAck(req IteratorRequest, ack bool, err error) {
	s.commandCounter++
	s.Out.Inlet() <- IteratorResponse{
		Variant: IteratorResponseTypeAck,
		Ack:     ack,
		Counter: s.commandCounter,
		Command: req.Command,
		Err:     err,
	}
}

func (s *streamIterator) error() error {
	select {
	case err := <-s.operationErrors:
		if err != nil {
			s._error = err
		}
	default:
	}
	return lo.Ternary(s._error != nil, s._error, s.internal.Error())
}

func (s *streamIterator) close() error {
	err := s.internal.Close()
	go func() {
		for _err := range s.operationErrors {
			if _err != nil {
				err = _err
			}
		}
	}()
	s.wg.Wait()
	close(s.operationErrors)
	return err
}

func (s *streamIterator) runCmd(req IteratorRequest) (bool, error) {
	switch req.Command {
	case IterNext:
		return s.internal.Next(), nil
	case IterPrev:
		return s.internal.Prev(), nil
	case IterFirst:
		return s.internal.First(), nil
	case IterLast:
		return s.internal.Last(), nil
	case IterNextSpan:
		return s.internal.NextSpan(req.Span), nil
	case IterPrevSpan:
		return s.internal.PrevSpan(req.Span), nil
	case IterReadView:
		return s.internal.SetRange(req.Range), nil
	case IterSeekFirst:
		return s.internal.SeekFirst(), nil
	case IterSeekLast:
		return s.internal.SeekLast(), nil
	case IterSeekLT:
		return s.internal.SeekLT(req.Stamp), nil
	case IterSeekGE:
		return s.internal.SeekGE(req.Stamp), nil
	case IterValid:
		return s.internal.Valid(), nil
	case IterError:
		return false, s.error()
	default:
		return false, nil
	}
}
