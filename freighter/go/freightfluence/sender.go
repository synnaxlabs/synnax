package freightfluence

import (
	"context"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/x/address"
	. "github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
)

// Sender wraps freighter.StreamSenderCloser to provide a confluence compatible
// interface for sending messages over a network freighter.
type Sender[M freighter.Payload] struct {
	Sender freighter.StreamSenderCloser[M]
	UnarySink[M]
}

// Flow implements Flow.
func (s *Sender[M]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(s.send, NewOptions(opts).Signal...)
}

func (s *Sender[M]) send(ctx context.Context) error {
	var err error
	defer func() {
		err = errors.CombineErrors(s.Sender.CloseSend(), err)
	}()
o:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break o
		case res, ok := <-s.UnarySink.In.Outlet():
			if !ok {
				break o
			}
			if sErr := s.Sender.Send(res); sErr != nil {
				err = sErr
				break o
			}
		}
	}
	return err
}

// TransformSender wraps freighter.StreamSenderCloser to provide a confluence compatible
// interface for sending messages over a network freighter. TransformSender adds
// a transform function to the Sender. This is particularly useful in cases
// where network message types are different from the message types used by the
// rest of a program.
type TransformSender[I Value, M freighter.Payload] struct {
	Sender freighter.StreamSenderCloser[M]
	TransformFunc[I, M]
	UnarySink[I]
}

// Flow implements the Flow interface.
func (s *TransformSender[I, M]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(s.send, NewOptions(opts).Signal...)
}

func (s *TransformSender[I, M]) send(ctx context.Context) error {
	var err error
	defer func() {
		err = errors.CombineErrors(s.Sender.CloseSend(), err)
	}()
o:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break o
		case res, ok := <-s.UnarySink.In.Outlet():
			if !ok {
				break o
			}
			tRes, ok, tErr := s.TransformFunc.Transform(ctx, res)
			if tErr != nil {
				err = tErr
				break o
			}
			if sErr := s.Sender.Send(tRes); sErr != nil {
				err = sErr
				break o
			}
		}
	}
	return err
}

// MultiSender wraps a slice of freighter.StreamSender(s) to provide a confluence
// compatible interface for sending messages over a network freighter. MultiSender
// sends a copy of each message received from the Outlet.
type MultiSender[M freighter.Payload] struct {
	Senders []freighter.StreamSenderCloser[M]
	UnarySink[M]
}

// Flow implements the Flow interface.
func (m *MultiSender[M]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(m.send, NewOptions(opts).Signal...)
}

func (m *MultiSender[M]) send(ctx context.Context) error {
	var err error
	defer func() {
		err = errors.CombineErrors(m.closeSenders(), err)
	}()
o:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break o
		case res, ok := <-m.UnarySink.In.Outlet():
			if !ok {
				break o
			}
			for _, sender := range m.Senders {
				if sErr := sender.Send(res); sErr != nil {
					err = sErr
					break o
				}
			}
		}
	}
	return err
}

func (m *MultiSender[M]) closeSenders() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, s := range m.Senders {
		c.Exec(s.CloseSend)
	}
	return c.Error()
}

type senderMap[M freighter.Payload] map[address.Address]freighter.StreamSenderCloser[M]

func (s senderMap[M]) send(target address.Address, msg M) error {
	sender, ok := s[target]
	if !ok {
		return address.TargetNotFound(target)
	}
	return sender.Send(msg)
}

func (s senderMap[M]) close() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, s := range s {
		c.Exec(s.CloseSend)
	}
	return c.Error()
}

// SwitchSender wraps a map of freighter.StreamSenderCloser to provide a confluence
// compatible interface for sending messages over a network freighter. SwitchSender
// receives a value, resolves its target address through a SwitchFunc, and sends it
// on its merry way.
type SwitchSender[M freighter.Payload] struct {
	Senders senderMap[M]
	SwitchFunc[M]
	UnarySink[M]
}

func (sw *SwitchSender[M]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(sw.send, NewOptions(opts).Signal...)
}

func (sw *SwitchSender[M]) send(ctx context.Context) error {
	var err error
	defer func() {
		err = errors.CombineErrors(sw.Senders.close(), err)
	}()
o:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break o
		case msg, ok := <-sw.In.Outlet():
			if !ok {
				break o
			}
			target, ok, swErr := sw.ApplySwitch(ctx, msg)
			if !ok || swErr != nil {
				err = swErr
				break o
			}
			if sErr := sw.Senders.send(target, msg); sErr != nil {
				err = sErr
				break o
			}
		}
	}
	return err
}

// BatchSwitchSender wraps a map of freighter.StreamSenderCloser to provide a confluence
// compatible interface for sending messages over a network freighter. BatchSwitchSender
// receives a batch of values, resolves their target addresses through a BatchSwitchFunc,
// and sends them on their merry way.
type BatchSwitchSender[I, O freighter.Payload] struct {
	Senders senderMap[O]
	BatchSwitchFunc[I, O]
	UnarySink[I]
}

func (bsw *BatchSwitchSender[I, O]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(bsw.send, NewOptions(opts).Signal...)
}

func (bsw *BatchSwitchSender[I, O]) send(ctx context.Context) error {
	var (
		err     error
		addrMap = make(map[address.Address]O)
	)
	defer func() {
		err = errors.CombineErrors(bsw.Senders.close(), err)
	}()
o:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break o
		case msg, ok := <-bsw.In.Outlet():
			if !ok {
				break o
			}
			if swErr := bsw.ApplySwitch(ctx, msg, addrMap); swErr != nil {
				err = swErr
				break o
			}
			for target, batch := range addrMap {
				sErr := bsw.Senders.send(target, batch)
				delete(addrMap, target)
				if sErr != nil {
					err = sErr
					break o
				}
			}
		}
	}
	return err
}
