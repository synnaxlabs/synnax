package cesium

import (
	"context"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type Iterator interface {
	SeekFirst() bool
	SeekLast() bool
	SeekGE(ts telem.TimeStamp) bool
	SeekLE(ts telem.TimeStamp) bool
	Next(span telem.TimeSpan) bool
	Prev(span telem.TimeSpan) bool
	Value() Frame
	SetBounds(tr telem.TimeRange)
	Valid() bool
	Close() error
}

type iterator struct {
	internal *streamIterator
	inlet    confluence.Inlet[IteratorRequest]
	outlet   confluence.Outlet[IteratorResponse]
	frame    Frame
	shutdown context.CancelFunc
	wg       signal.WaitGroup
	logger   *zap.Logger
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
func (i *iterator) Value() Frame { return i.frame }

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
	i.frame = Frame{}
	i.inlet.Inlet() <- req
	for res := range i.outlet.Outlet() {
		if res.Variant == IteratorAckResponse {
			return res.Ack, res.Err
		}
		i.frame = i.frame.AppendFrame(res.Frame)
	}
	i.logger.DPanic(unexpectedSteamClosure)
	return false, nil
}
