package iterator

import (
	"context"
	"github.com/arya-analytics/cesium"
	core "github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
)

// emitter translates iterator commands into req and writes them to a stream.
type emitter struct {
	confluence.AbstractUnarySource[Request]
	confluence.EmptyFlow
}

// Next emits a Next request to the stream.
func (e *emitter) next() { e.emit(Request{Command: Next}) }

// Prev emits a Prev request to the stream.
func (e *emitter) Prev() { e.emit(Request{Command: Prev}) }

// First emits a First request to the stream.
func (e *emitter) First() { e.emit(Request{Command: First}) }

// Last emits a Last request to the stream.
func (e *emitter) Last() { e.emit(Request{Command: Last}) }

// NextSpan emits a NextSpan request to the stream.
func (e *emitter) NextSpan(span telem.TimeSpan) {
	e.emit(Request{Command: NextSpan, Span: span})
}

// PrevSpan emits a PrevSpan request to the stream.
func (e *emitter) PrevSpan(span telem.TimeSpan) {
	e.emit(Request{Command: PrevSpan, Span: span})
}

// NextRange emits a NextRange request to the stream.
func (e *emitter) NextRange(rng telem.TimeRange) {
	e.emit(Request{Command: NextRange, Range: rng})
}

// SeekFirst emits a SeekFirst request to the stream.
func (e *emitter) SeekFirst() { e.emit(Request{Command: SeekFirst}) }

// SeekLast emits a SeekLast request to the stream.
func (e *emitter) SeekLast() { e.emit(Request{Command: SeekLast}) }

// SeekLT emits a SeekLT request to the stream.
func (e *emitter) SeekLT(stamp telem.TimeStamp) {
	e.emit(Request{Command: SeekLT, Stamp: stamp})
}

// SeekGE emits a SeekGE request to the stream.
func (e *emitter) SeekGE(stamp telem.TimeStamp) {
	e.emit(Request{Command: SeekGE, Stamp: stamp})
}

// Close emits a Close request to the stream.
func (e *emitter) Close() {
	e.emit(Request{Command: Close})
}

// Valid emits a Valid request to the stream.
func (e *emitter) Valid() { e.emit(Request{Command: Valid}) }

func (e *emitter) Exhaust() { e.emit(Request{Command: Exhaust}) }

// Error emits an Error request to the stream.
func (e *emitter) Error() { e.emit(Request{Command: Error}) }

func (e *emitter) emit(req Request) { e.Out.Inlet() <- req }

func executeRequest(ctx context.Context, host core.NodeID, iter cesium.StreamIterator, req Request) Response {
	switch req.Command {
	case Open:
		ack := newAck(host, req.Command, false)
		ack.Error = errors.New(
			"[segment.iterator.serve] - Open command called multiple times",
		)
		return ack
	case Next:
		return newAck(host, req.Command, iter.Next())
	case Prev:
		return newAck(host, req.Command, iter.Prev())
	case First:
		return newAck(host, req.Command, iter.First())
	case Last:
		return newAck(host, req.Command, iter.Last())
	case NextSpan:
		return newAck(host, req.Command, iter.NextSpan(req.Span))
	case PrevSpan:
		return newAck(host, req.Command, iter.PrevSpan(req.Span))
	case NextRange:
		return newAck(host, req.Command, iter.NextRange(req.Range))
	case SeekFirst:
		return newAck(host, req.Command, iter.SeekFirst())
	case SeekLast:
		return newAck(host, req.Command, iter.SeekLast())
	case SeekLT:
		return newAck(host, req.Command, iter.SeekLT(req.Stamp))
	case SeekGE:
		return newAck(host, req.Command, iter.SeekGE(req.Stamp))
	case Valid:
		return newAck(host, req.Command, iter.Valid())
	case Error:
		err := iter.Error()
		ack := newAck(host, req.Command, err == nil)
		ack.Error = err
		return ack
	case Close:
		err := iter.Close()
		ack := newAck(host, req.Command, err == nil)
		ack.Error = err
		return ack
	case Exhaust:
		for iter.First(); iter.Next(); iter.Valid() {
		}
		return newAck(host, req.Command, true)
	default:
		ack := newAck(host, req.Command, false)
		ack.Error = errors.New("[segment.iterator] - unknown command")
		return ack
	}
}
