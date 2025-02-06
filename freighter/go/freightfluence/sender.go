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
	errors2 "github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
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
// interface for sending messages over a network freighter. TransformSender adds
// a transform function to the Sender. This is particularly useful in cases
// where network message types are different from the message types used by the
// rest of a program.
type TransformSender[I Value, M freighter.Payload] struct {
	Sender    freighter.StreamSenderCloser[M]
	Transform TransformFunc[I, M]
	UnarySink[I]
}

// Flow implements the Flow interface.
func (s *TransformSender[I, M]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(s.send, NewOptions(opts).Signal...)
}

func (s *TransformSender[I, M]) send(ctx context.Context) error {
	var err error
	defer func() {
		err = errors.Combine(s.Sender.CloseSend(), err)
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
			tRes, ok, tErr := s.Transform(ctx, res)
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
		err = errors.Combine(m.closeSenders(), err)
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
	c := errors2.NewCatcher(errors2.WithAggregation())
	for _, s := range m.Senders {
		c.Exec(s.CloseSend)
	}
	return c.Error()
}

type MapTargetedSender[M freighter.Payload] map[address.Address]freighter.StreamSenderCloser[M]

func (s MapTargetedSender[M]) Send(_ context.Context, target address.Address, msg M) error {
	sender, ok := s[target]
	if !ok {
		return address.TargetNotFound(target)
	}
	return sender.Send(msg)
}

func (s MapTargetedSender[M]) Close() error {
	c := errors2.NewCatcher(errors2.WithAggregation())
	for _, s := range s {
		c.Exec(s.CloseSend)
	}
	return c.Error()
}

type ClientTargetedSender[RQ, RS freighter.Payload] struct {
	Transport freighter.StreamClient[RQ, RS]
	MapTargetedSender[RQ]
}

func (c ClientTargetedSender[RQ, RS]) Send(ctx context.Context, target address.Address, req RQ) error {
	sender, ok := c.MapTargetedSender[target]
	if !ok {
		if err := c.open(ctx, target); err != nil {
			return err
		}
		sender = c.MapTargetedSender[target]
	}
	return sender.Send(req)
}

func (c ClientTargetedSender[RQ, RS]) Close() error {
	return c.MapTargetedSender.Close()
}

func (c ClientTargetedSender[RQ, RS]) open(ctx context.Context, target address.Address) error {
	stream, err := c.Transport.Stream(ctx, target)
	if err != nil {
		return err
	}
	c.MapTargetedSender[target] = stream
	return nil
}

// SwitchSender wraps a map of freighter.StreamSenderCloser to provide a confluence
// compatible interface for sending messages over a network freighter. SwitchSender
// receives a value, resolves its target address through a SwitchFunc, and sends it
// on its merry way.
type SwitchSender[M freighter.Payload] struct {
	Sender TargetedSender[M]
	Switch SwitchFunc[M]
	UnarySink[M]
}

func (sw *SwitchSender[M]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(sw.send, NewOptions(opts).Signal...)
}

func (sw *SwitchSender[M]) send(ctx context.Context) error {
	var err error
	defer func() {
		err = errors.Combine(sw.Sender.Close(), err)
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
			target, ok, swErr := sw.Switch(ctx, msg)
			if !ok || swErr != nil {
				err = swErr
				break o
			}
			if sErr := sw.Sender.Send(ctx, target, msg); sErr != nil {
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
	Senders TargetedSender[O]
	Switch  BatchSwitchFunc[I, O]
	UnarySink[I]
}

type TargetedSender[M freighter.Payload] interface {
	Send(ctx context.Context, target address.Address, msg M) error
	Close() error
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
		err = errors.Combine(bsw.Senders.Close(), err)
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
			if err = bsw.Switch(ctx, msg, addrMap); err != nil {
				return err
			}
			for target, batch := range addrMap {
				sErr := bsw.Senders.Send(ctx, target, batch)
				if sErr != nil {
					return sErr
				}
			}
		}
	}
	return err
}
