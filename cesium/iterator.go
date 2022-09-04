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

type Iterator interface {
	// Next pipes the next segment in the Iterator to the Source responses.
	// It returns true if the Iterator is pointing to a valid segment.
	Next() bool
	// Prev pipes the previous segment in the Iterator to the Source responses.
	// It returns true if the Iterator is pointing to a valid segment.
	Prev() bool
	// First seeks to the first segment in the Iterator. Returns true
	// if the iterator is pointing to a valid segment.
	First() bool
	// Last seeks to the last segment in the Iterator. Returns true
	// if the iterator is pointing to a valid segment.
	Last() bool
	// NextSpan pipes all segments in the Iterator from the current position to
	// the end of the span. It returns true if the iterator is pointing to a
	// valid segment. If span is TimeSpanMax, it will exhaust the iterator. If
	// span is TimeSpanZero, it won't do anything.
	NextSpan(span telem.TimeSpan) bool
	// PrevSpan pipes all segments in the Iterator from the current
	PrevSpan(span telem.TimeSpan) bool
	// SetRange seeks the Iterator to the start of the provided range and pipes all segments bound by it
	// to the Source responses. It returns true if the iterator is pointing to a valid segment.
	// If range is TimeRangeMax, exhausts the Iterator. If range is TimeRangeZero, it won't do anything.
	SetRange(tr telem.TimeRange) bool
	// SeekFirst seeks the iterator to the first segment in the range.
	SeekFirst() bool
	// SeekLast seeks the iterator to the last segment in the range.
	SeekLast() bool
	// SeekLT seeks the Iterator to the first segment with a timestamp less than the provided timestamp.
	// It returns true if the Iterator is pointing to a valid segment.
	SeekLT(ts telem.TimeStamp) bool
	// SeekGE seeks the Iterator to the first segment with a timestamp greater than or equal to the provided timestamp.
	// It returns true if the Iterator is pointing to a valid segment.
	SeekGE(ts telem.TimeStamp) bool
	// View returns the current range of values the Iterator has a 'view' of.  This view represents the range of
	// segments most recently returned to the caller.
	View() telem.TimeRange
	// Close closes the Iterator, ensuring that all in-progress segment reads complete before closing the Source responses.
	Close() error
	// Valid returns true if the Iterator is pointing at valid segments.
	Valid() bool
	// Error returns any errors accumulated by the Iterator.
	Error() error
	Value() []Segment
}

type iterator struct {
	internal streamIterator
	inlet    confluence.Inlet[IterateRequest]
	outlet   confluence.Outlet[IterateResponse]
	value    []IterateResponse
	shutdown context.CancelFunc
	wg       signal.WaitGroup
}

func wrapStreamIterator(stream *streamIterator) *iterator {
	ctx, cancel := signal.Background()
	requests := confluence.NewStream[IterateRequest]()
	responses := confluence.NewStream[IterateResponse]()
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

func (i *iterator) Next() bool {
	i.value = nil
	return i.exec(IterateRequest{Command: IterNext})
}

func (i *iterator) Prev() bool {
	i.value = nil
	return i.exec(IterateRequest{Command: IterPrev})
}

func (i *iterator) First() bool {
	i.value = nil
	return i.exec(IterateRequest{Command: IterFirst})
}

func (i *iterator) Last() bool {
	i.value = nil
	return i.exec(IterateRequest{Command: IterLast})
}

func (i *iterator) NextSpan(span telem.TimeSpan) bool {
	i.value = nil
	return i.exec(IterateRequest{Command: IterNextSpan, Span: span})
}

func (i *iterator) PrevSpan(span telem.TimeSpan) bool {
	i.value = nil
	return i.exec(IterateRequest{Command: IterPrevSpan, Span: span})
}

func (i *iterator) SetRange(tr telem.TimeRange) bool {
	i.value = nil
	return i.exec(IterateRequest{Command: IterRange, Range: tr})
}

func (i *iterator) SeekFirst() bool {
	i.value = nil
	return i.exec(IterateRequest{Command: IterSeekFirst})
}

func (i *iterator) SeekLast() bool {
	i.value = nil
	return i.exec(IterateRequest{Command: IterSeekLast})
}

func (i *iterator) SeekLT(ts telem.TimeStamp) bool {
	i.value = nil
	return i.exec(IterateRequest{Command: IterSeekLT, Stamp: ts})
}

func (i *iterator) SeekGE(time telem.TimeStamp) bool {
	i.value = nil
	return i.exec(IterateRequest{Command: IterSeekGE, Stamp: time})
}

func (i *iterator) View() telem.TimeRange { return i.internal.internal.View() }

func (i *iterator) Close() error {
	i.value = nil
	i.inlet.Close()
	return i.wg.Wait()
}

func (i *iterator) Error() error {
	_, err := i.execErr(IterateRequest{Command: IterError})
	return err
}

func (i *iterator) Valid() bool {
	return i.exec(IterateRequest{Command: IterValid})
}

func (i *iterator) Value() []Segment {
	var segments []Segment
	for _, resp := range i.value {
		segments = append(segments, resp.Segments...)
	}
	return segments
}

func (i *iterator) exec(req IterateRequest) bool {
	ok, _ := i.execErr(req)
	return ok
}

func (i *iterator) execErr(req IterateRequest) (bool, error) {
	i.inlet.Inlet() <- req
	for res := range i.outlet.Outlet() {
		if res.Variant == AckResponse {
			return res.Ack, res.Err
		}
		i.value = append(i.value, res)
	}
	return false, nil
}

type StreamIterator = confluence.Segment[IterateRequest, IterateResponse]

type streamIterator struct {
	*confluence.AbstractUnarySource[IterateResponse]
	confluence.UnarySink[IterateRequest]
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
	responses := &confluence.AbstractUnarySource[IterateResponse]{}
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

func (s *streamIterator) exec(req IterateRequest) {
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

func (s *streamIterator) sendAck(req IterateRequest, ack bool, err error) {
	s.commandCounter++
	s.Out.Inlet() <- IterateResponse{
		Variant: AckResponse,
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

func (s *streamIterator) runCmd(req IterateRequest) (bool, error) {
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
	case IterRange:
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
