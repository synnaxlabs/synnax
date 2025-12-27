// Copyright 2025 Synnax Labs, Inc.
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

type controlStateSender struct {
	db                                 *ts.DB
	controlStateKey                    channel.Key
	previouslyContainedControlStateKey bool
	confluence.LinearTransform[StreamerRequest, StreamerResponse]
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

func (c *controlStateSender) getControlUpdateFrame(ctx context.Context) frame.Frame {
	u := c.db.ControlUpdateToFrame(ctx, c.db.ControlStates())
	return frame.NewFromStorage(u)
}

func (c *controlStateSender) Flow(ctx signal.Context, opts ...confluence.Option) {
	if c.previouslyContainedControlStateKey {
		_ = signal.SendUnderContext(ctx, c.Out.Inlet(), StreamerResponse{Frame: c.getControlUpdateFrame(ctx)})
	}
	c.LinearTransform.Flow(ctx, opts...)
}

func (c *controlStateSender) transform(ctx context.Context, req StreamerRequest) (res StreamerResponse, send bool, err error) {
	containsControlStateKey := lo.Contains(req.Keys, c.controlStateKey)
	previouslyContainedControlStateKey := c.previouslyContainedControlStateKey
	c.previouslyContainedControlStateKey = containsControlStateKey
	if containsControlStateKey && !previouslyContainedControlStateKey {
		send = true
		res.Frame = c.getControlUpdateFrame(ctx)
	}
	return
}

const (
	relayReaderAddr        address.Address = "relay_reader"
	controlStateSenderAddr address.Address = "control_state_sender"
	requestMultiplierAddr  address.Address = "request_multiplier"
)

func (s *Service) NewStreamer(ctx context.Context, cfg StreamerConfig) (Streamer, error) {
	rel, err := s.Relay.NewStreamer(ctx, cfg)
	if err != nil {
		return nil, err
	}
	controlStateSender := newControlStateSender(s.cfg.TS, s.controlStateKey, cfg.Keys)
	p := plumber.New()
	plumber.SetSegment(p, relayReaderAddr, rel)
	plumber.SetSegment(p, controlStateSenderAddr, controlStateSender)
	plumber.SetSegment(p, requestMultiplierAddr, &confluence.DeltaMultiplier[StreamerRequest]{})
	plumber.MultiRouter[StreamerRequest]{
		Capacity:      5,
		SourceTargets: []address.Address{requestMultiplierAddr},
		SinkTargets:   []address.Address{controlStateSenderAddr, relayReaderAddr},
		Stitch:        plumber.StitchWeave,
	}.MustRoute(p)
	seg := &plumber.Segment[StreamerRequest, StreamerResponse]{
		Pipeline:         p,
		RouteInletsTo:    []address.Address{requestMultiplierAddr},
		RouteOutletsFrom: []address.Address{controlStateSenderAddr, relayReaderAddr},
	}
	return seg, nil
}
