// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
)

func controlUpdateFrame(ctx context.Context, db *ts.DB) frame.Frame {
	return frame.NewFromStorage(db.ControlUpdateToFrame(ctx, db.ControlStates()))
}

type controlStateSender struct {
	confluence.LinearTransform[StreamerRequest, StreamerResponse]
	db                                 *ts.DB
	controlStateKey                    channel.Key
	previouslyContainedControlStateKey bool
}

func newControlStateSender(
	ts *ts.DB,
	controlStateKey channel.Key,
	keys channel.Keys,
) *controlStateSender {
	c := &controlStateSender{
		db:                                 ts,
		controlStateKey:                    controlStateKey,
		previouslyContainedControlStateKey: lo.Contains(keys, controlStateKey),
	}
	c.Transform = c.transform
	return c
}

func (c *controlStateSender) transform(
	ctx context.Context,
	req StreamerRequest,
) (res StreamerResponse, send bool, err error) {
	containsControlStateKey := lo.Contains(req.Keys, c.controlStateKey)
	previouslyContainedControlStateKey := c.previouslyContainedControlStateKey
	c.previouslyContainedControlStateKey = containsControlStateKey
	if containsControlStateKey && !previouslyContainedControlStateKey {
		send = true
		res.Frame = controlUpdateFrame(ctx, c.db)
	}
	return res, send, err
}

// initialStateSequencer forwards relay responses and injects the initial
// control-state snapshot for streamers whose keys include the control-state
// channel. When the relay sends an open ack, the snapshot must follow it:
// clients consume the first response as the ack and discard its frame, so a
// snapshot emitted at flow start could be lost to that read. Without an ack
// there is nothing to sequence behind and the snapshot is sent at flow start.
type initialStateSequencer struct {
	confluence.AbstractLinear[StreamerResponse, StreamerResponse]
	db               *ts.DB
	snapshotAfterAck bool
	snapshotOnStart  bool
}

func (s *initialStateSequencer) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)
	ctx.Go(func(ctx context.Context) error {
		if s.snapshotOnStart {
			if err := signal.SendUnderContext(
				ctx,
				s.Out.Inlet(),
				StreamerResponse{Frame: controlUpdateFrame(ctx, s.db)},
			); err != nil {
				return err
			}
		}
		pending := s.snapshotAfterAck
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case res, ok := <-s.In.Outlet():
				if !ok {
					return nil
				}
				if err := signal.SendUnderContext(ctx, s.Out.Inlet(), res); err != nil {
					return err
				}
				if pending {
					pending = false
					if err := signal.SendUnderContext(
						ctx,
						s.Out.Inlet(),
						StreamerResponse{Frame: controlUpdateFrame(ctx, s.db)},
					); err != nil {
						return err
					}
				}
			}
		}
	}, o.Signal...)
}

const (
	relayReaderAddr        address.Address = "relay_reader"
	controlStateSenderAddr address.Address = "control_state_sender"
	sequencerAddr          address.Address = "initial_state_sequencer"
	requestMultiplierAddr  address.Address = "request_multiplier"
)

func (s *Service) NewStreamer(cfg StreamerConfig) (Streamer, error) {
	rel, err := s.relay.NewStreamer(cfg)
	if err != nil {
		return nil, err
	}
	controlStateSender := newControlStateSender(s.cfg.TS, s.controlStateKey, cfg.Keys)
	sendAck := cfg.SendOpenAck != nil && *cfg.SendOpenAck
	containsControlKey := lo.Contains(cfg.Keys, s.controlStateKey)
	sequencer := &initialStateSequencer{
		db:               s.cfg.TS,
		snapshotAfterAck: sendAck && containsControlKey,
		snapshotOnStart:  !sendAck && containsControlKey,
	}
	p := plumber.New()
	plumber.SetSegment(p, relayReaderAddr, rel)
	plumber.SetSegment(p, controlStateSenderAddr, controlStateSender)
	plumber.SetSegment(p, sequencerAddr, sequencer)
	plumber.SetSegment(
		p,
		requestMultiplierAddr,
		&confluence.DeltaMultiplier[StreamerRequest]{},
	)
	plumber.MultiRouter[StreamerRequest]{
		Capacity:      5,
		SourceTargets: []address.Address{requestMultiplierAddr},
		SinkTargets:   []address.Address{controlStateSenderAddr, relayReaderAddr},
		Stitch:        plumber.StitchWeave,
	}.MustRoute(p)
	plumber.MustConnect[StreamerResponse](p, relayReaderAddr, sequencerAddr, 10)
	seg := &plumber.Segment[StreamerRequest, StreamerResponse]{
		Pipeline:         p,
		RouteInletsTo:    []address.Address{requestMultiplierAddr},
		RouteOutletsFrom: []address.Address{controlStateSenderAddr, sequencerAddr},
	}
	return seg, nil
}
