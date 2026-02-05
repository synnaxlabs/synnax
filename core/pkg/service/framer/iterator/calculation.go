// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

type calculationTransform struct {
	confluence.UnarySink[Response]
	confluence.AbstractUnarySource[Response]
	keepKeys         channel.Keys
	calculators      []*calculator.Calculator
	accumulatedError error
	pendingFrames    []framer.Frame
}

func newCalculationTransform(
	keepKeys channel.Keys,
	calculators []*calculator.Calculator,
) *calculationTransform {
	return &calculationTransform{
		calculators:   calculators,
		keepKeys:      keepKeys,
		pendingFrames: make([]framer.Frame, 0, 8),
	}
}

func (t *calculationTransform) close() error {
	var err error
	for _, calc := range t.calculators {
		err = errors.Join(err, calc.Close())
	}
	return err
}

func (t *calculationTransform) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(t.Out)
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case res, ok := <-t.In.Outlet():
				if !ok {
					return nil
				}
				t.processResponse(ctx, res)
			}
		}
	}, o.Signal...)
}

func (t *calculationTransform) processResponse(ctx context.Context, res Response) {
	if res.Command == CommandError {
		res.Error = errors.Combine(res.Error, t.accumulatedError)
		t.Out.Inlet() <- res
		return
	}
	if res.Variant == ResponseVariantData {
		if res.Frame.Count() > 0 {
			t.pendingFrames = append(t.pendingFrames, res.Frame)
		}
		return
	}
	if res.Variant == ResponseVariantAck {
		t.processBufferedFrames(ctx, res)
		return
	}
	t.Out.Inlet() <- res
}

func (t *calculationTransform) processBufferedFrames(ctx context.Context, ackRes Response) {
	defer func() { t.pendingFrames = t.pendingFrames[:0] }()
	if len(t.pendingFrames) == 0 {
		if t.accumulatedError != nil {
			ackRes.Ack = false
		}
		t.Out.Inlet() <- ackRes
		return
	}
	mergedFrame := frame.Merge(t.pendingFrames)
	var err error
	for _, c := range t.calculators {
		mergedFrame, _, err = c.Next(ctx, mergedFrame, mergedFrame)
		if err != nil {
			t.accumulatedError = err
			continue
		}
	}
	mergedFrame = mergedFrame.KeepKeys(t.keepKeys)
	if mergedFrame.Count() > 0 {
		t.Out.Inlet() <- Response{
			Variant: ResponseVariantData,
			Command: ackRes.Command,
			SeqNum:  ackRes.SeqNum,
			Frame:   mergedFrame,
		}
	}
	if t.accumulatedError != nil {
		ackRes.Ack = false
	}
	t.Out.Inlet() <- ackRes
}
