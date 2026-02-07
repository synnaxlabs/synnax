// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

// Sender wraps freighter.StreamSenderCloser to provide a confluence compatible
// interface for sending messages over a network freighter.
type Sender[M freighter.Payload] struct {
	Sender freighter.StreamSenderCloser[M]
	confluence.UnarySink[M]
}

// Flow implements Flow.
func (s *Sender[M]) Flow(ctx signal.Context, opts ...confluence.Option) {
	ctx.Go(s.send, confluence.NewOptions(opts).Signal...)
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
		case res, ok := <-s.In.Outlet():
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
type TransformSender[I confluence.Value, M freighter.Payload] struct {
	Sender    freighter.StreamSenderCloser[M]
	Transform confluence.TransformFunc[I, M]
	confluence.UnarySink[I]
}

// Flow implements the Flow interface.
func (s *TransformSender[I, M]) Flow(ctx signal.Context, opts ...confluence.Option) {
	ctx.Go(s.send, confluence.NewOptions(opts).Signal...)
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
		case res, ok := <-s.In.Outlet():
			if !ok {
				break o
			}
			tRes, shouldSend, tErr := s.Transform(ctx, res)
			if tErr != nil {
				err = tErr
				break o
			}
			if !shouldSend {
				continue o
			}
			if sErr := s.Sender.Send(tRes); sErr != nil {
				err = sErr
				break o
			}
		}
	}
	return err
}

type MapTargetedSender[M freighter.Payload] map[address.Address]freighter.StreamSenderCloser[M]

func (s MapTargetedSender[M]) Send(_ context.Context, target address.Address, msg M) error {
	sender, ok := s[target]
	if !ok {
		return address.NewTargetNotFoundError(target)
	}
	return sender.Send(msg)
}

func (s MapTargetedSender[M]) Close() error {
	var err error
	for _, s := range s {
		err = errors.Join(err, s.CloseSend())
	}
	return err
}

// BatchSwitchSender wraps a map of freighter.StreamSenderCloser to provide a confluence
// compatible interface for sending messages over a network freighter. BatchSwitchSender
// receives a batch of values, resolves their target addresses through a BatchSwitchFunc,
// and sends them on their merry way.
type BatchSwitchSender[I, O freighter.Payload] struct {
	Senders TargetedSender[O]
	Switch  confluence.BatchSwitchFunc[I, O]
	confluence.UnarySink[I]
}

type TargetedSender[M freighter.Payload] interface {
	Send(ctx context.Context, target address.Address, msg M) error
	Close() error
}

func (bsw *BatchSwitchSender[I, O]) Flow(
	ctx signal.Context,
	opts ...confluence.Option,
) {
	ctx.Go(bsw.send, confluence.NewOptions(opts).Signal...)
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

// MultiTransformSender wraps a slice of freighter.StreamSender(s) to provide a confluence
// compatible interface for sending transformed messages over a network freighter.
// MultiTransformSender transforms each input message and sends a copy to each sender.
type MultiTransformSender[I confluence.Value, M freighter.Payload] struct {
	confluence.UnarySink[I]
	Transform confluence.TransformFunc[I, M]
	Senders   []freighter.StreamSenderCloser[M]
}

// Flow implements the Flow interface.
func (m *MultiTransformSender[I, M]) Flow(
	ctx signal.Context,
	opts ...confluence.Option,
) {
	ctx.Go(m.send, confluence.NewOptions(opts).Signal...)
}

func (m *MultiTransformSender[I, M]) send(ctx context.Context) error {
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
		case res, ok := <-m.In.Outlet():
			if !ok {
				break o
			}
			// Transform the input message
			tRes, ok, tErr := m.Transform(ctx, res)
			if tErr != nil {
				err = tErr
				break o
			}
			if !ok {
				continue
			}
			// Send the transformed message to all senders
			for _, sender := range m.Senders {
				if sErr := sender.Send(tRes); sErr != nil {
					err = sErr
					break o
				}
			}
		}
	}
	return err
}

func (m *MultiTransformSender[I, M]) closeSenders() error {
	var err error
	for _, s := range m.Senders {
		err = errors.Join(err, s.CloseSend())
	}
	return err
}
