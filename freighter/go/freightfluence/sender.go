// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freightfluence

import (
	"context"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

// Sender wraps freighter.StreamSenderCloser to provide a confluence compatible
// interface for sending messages over a network freighter.
type Sender[P freighter.Payload] struct {
	Sender freighter.StreamSenderCloser[P]
	UnarySink[P]
}

var _ Flow = (*Sender[any])(nil)

// Flow implements Flow.
func (s *Sender[P]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(s.send, NewOptions(opts).Signal...)
}

func (s *Sender[P]) send(ctx context.Context) error {
	var err error
	defer func() {
		err = errors.Combine(s.Sender.CloseSend(), err)
	}()
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return err
		case res, ok := <-s.UnarySink.In.Outlet():
			if !ok {
				return nil
			}
			if err = s.Sender.Send(res); err != nil {
				return err
			}
		}
	}
}

// TransformSender wraps freighter.StreamSenderCloser to provide a confluence compatible
// interface for sending messages over a network freighter. TransformSender adds a
// transform function to the Sender. This is particularly useful in cases where network
// message types are different from the message types used by the rest of a program.
type TransformSender[I Value, P freighter.Payload] struct {
	Sender    freighter.StreamSenderCloser[P]
	Transform TransformFunc[I, P]
	UnarySink[I]
}

var _ Flow = (*TransformSender[any, any])(nil)

// Flow implements the Flow interface.
func (ts *TransformSender[I, M]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(ts.send, NewOptions(opts).Signal...)
}

func (ts *TransformSender[I, M]) send(ctx context.Context) error {
	var err error
	defer func() {
		err = errors.Combine(ts.Sender.CloseSend(), err)
	}()
o:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break o
		case res, ok := <-ts.UnarySink.In.Outlet():
			if !ok {
				break o
			}
			tRes, ok, tErr := ts.Transform(ctx, res)
			if tErr != nil {
				err = tErr
				break o
			}
			if sErr := ts.Sender.Send(tRes); sErr != nil {
				err = sErr
				break o
			}
		}
	}
	return err
}

// MultiSender wraps a slice of freighter.StreamSender(s) to provide a confluence
// compatible interface for sending messages over a network freighter. MultiSender sends
// a copy of each message received from the Outlet.
type MultiSender[P freighter.Payload] struct {
	Senders []freighter.StreamSenderCloser[P]
	UnarySink[P]
}

var _ Flow = (*MultiSender[any])(nil)

// Flow implements the Flow interface.
func (ms *MultiSender[M]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(ms.send, NewOptions(opts).Signal...)
}

func (ms *MultiSender[M]) send(ctx context.Context) error {
	var err error
	defer func() {
		err = errors.Combine(ms.closeSenders(), err)
	}()
o:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break o
		case res, ok := <-ms.UnarySink.In.Outlet():
			if !ok {
				break o
			}
			for _, sender := range ms.Senders {
				if sErr := sender.Send(res); sErr != nil {
					err = sErr
					break o
				}
			}
		}
	}
	return err
}

func (ms *MultiSender[M]) closeSenders() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, s := range ms.Senders {
		c.Exec(s.CloseSend)
	}
	return c.Error()
}

var _ TargetedSender[any] = (*MapTargetedSender[any])(nil)

type MapTargetedSender[P freighter.Payload] map[address.Address]freighter.StreamSenderCloser[P]

func (mts MapTargetedSender[M]) Send(_ context.Context, target address.Address, msg M) error {
	sender, ok := mts[target]
	if !ok {
		return address.NewErrTargetNotFound(target)
	}
	return sender.Send(msg)
}

func (mts MapTargetedSender[M]) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, s := range mts {
		c.Exec(s.CloseSend)
	}
	return c.Error()
}

var _ TargetedSender[any] = (*ClientTargetedSender[any, any])(nil)

type ClientTargetedSender[RQ, RS freighter.Payload] struct {
	Transport freighter.StreamClient[RQ, RS]
	MapTargetedSender[RQ]
}

func (cts ClientTargetedSender[RQ, RS]) Send(ctx context.Context, target address.Address, req RQ) error {
	sender, ok := cts.MapTargetedSender[target]
	if !ok {
		if err := cts.open(ctx, target); err != nil {
			return err
		}
		sender = cts.MapTargetedSender[target]
	}
	return sender.Send(req)
}

func (cts ClientTargetedSender[RQ, RS]) Close() error {
	return cts.MapTargetedSender.Close()
}

func (cts ClientTargetedSender[RQ, RS]) open(ctx context.Context, target address.Address) error {
	stream, err := cts.Transport.Stream(ctx, target)
	if err != nil {
		return err
	}
	cts.MapTargetedSender[target] = stream
	return nil
}

// SwitchSender wraps a map of freighter.StreamSenderCloser to provide a confluence
// compatible interface for sending messages over a network freighter. SwitchSender
// receives a value, resolves its target address through a SwitchFunc, and sends it on
// its merry way.
type SwitchSender[P freighter.Payload] struct {
	Sender TargetedSender[P]
	Switch SwitchFunc[P]
	UnarySink[P]
}

var _ Flow = (*SwitchSender[any])(nil)

func (ss *SwitchSender[M]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(ss.send, NewOptions(opts).Signal...)
}

func (ss *SwitchSender[M]) send(ctx context.Context) error {
	var err error
	defer func() {
		err = errors.Combine(ss.Sender.Close(), err)
	}()
o:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break o
		case msg, ok := <-ss.In.Outlet():
			if !ok {
				break o
			}
			target, ok, swErr := ss.Switch(ctx, msg)
			if !ok || swErr != nil {
				err = swErr
				break o
			}
			if sErr := ss.Sender.Send(ctx, target, msg); sErr != nil {
				err = sErr
				break o
			}
		}
	}
	return err
}

// BatchSwitchSender wraps a map of freighter.StreamSenderCloser to provide a confluence
// compatible interface for sending messages over a network freighter. BatchSwitchSender
// receives a batch of values, resolves their target addresses through a
// BatchSwitchFunc, and sends them on their merry way.
type BatchSwitchSender[I, O freighter.Payload] struct {
	Senders TargetedSender[O]
	Switch  BatchSwitchFunc[I, O]
	UnarySink[I]
}

var _ Flow = (*BatchSwitchSender[any, any])(nil)

type TargetedSender[P freighter.Payload] interface {
	Send(context.Context, address.Address, P) error
	Close() error
}

func (bss *BatchSwitchSender[I, O]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(bss.send, NewOptions(opts).Signal...)
}

func (bss *BatchSwitchSender[I, O]) send(ctx context.Context) error {
	var (
		err     error
		addrMap = make(map[address.Address]O)
	)
	defer func() {
		err = errors.Combine(bss.Senders.Close(), err)
	}()
o:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break o
		case msg, ok := <-bss.In.Outlet():
			if !ok {
				break o
			}
			if err = bss.Switch(ctx, msg, addrMap); err != nil {
				return err
			}
			for target, batch := range addrMap {
				sErr := bss.Senders.Send(ctx, target, batch)
				if sErr != nil {
					return sErr
				}
			}
		}
	}
	return err
}

// MultiTransformSender wraps a slice of freighter.StreamSender(s) to provide a
// confluence compatible interface for sending transformed messages over a network
// freighter. MultiTransformSender transforms each input message and sends a copy to
// each sender.
type MultiTransformSender[I Value, P freighter.Payload] struct {
	Senders   []freighter.StreamSenderCloser[P]
	Transform TransformFunc[I, P]
	UnarySink[I]
}

var _ Flow = (*MultiTransformSender[any, any])(nil)

// Flow implements the Flow interface.
func (mts *MultiTransformSender[I, M]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(mts.send, NewOptions(opts).Signal...)
}

func (mts *MultiTransformSender[I, M]) send(ctx context.Context) error {
	var err error
	defer func() {
		err = errors.Combine(mts.closeSenders(), err)
	}()
o:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break o
		case res, ok := <-mts.UnarySink.In.Outlet():
			if !ok {
				break o
			}
			// Transform the input message
			tRes, ok, tErr := mts.Transform(ctx, res)
			if tErr != nil {
				err = tErr
				break o
			}
			if !ok {
				continue
			}
			// Send the transformed message to all senders
			for _, sender := range mts.Senders {
				if sErr := sender.Send(tRes); sErr != nil {
					err = sErr
					break o
				}
			}
		}
	}
	return err
}

func (mts *MultiTransformSender[I, M]) closeSenders() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, s := range mts.Senders {
		c.Exec(s.CloseSend)
	}
	return c.Error()
}
