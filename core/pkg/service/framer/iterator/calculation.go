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

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
)

// calculationTransform runs calculations on each data response as it arrives, so a
// read's memory stays bounded by the size of one response.
type calculationTransform struct {
	confluence.UnarySink[Response]
	confluence.AbstractUnarySource[Response]
	keepKeys         set.Set[channel.Key]
	calculators      []*calculator.Calculator
	accumulatedError error
}

func newCalculationTransform(
	keepKeys channel.Keys,
	calculators []*calculator.Calculator,
) *calculationTransform {
	return &calculationTransform{
		calculators: calculators,
		keepKeys:    set.New(keepKeys...),
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
	switch {
	case res.Command == CommandError:
		res.Error = errors.Combine(res.Error, t.accumulatedError)
	case res.Variant == ResponseVariantData:
		res.Frame = t.calculate(ctx, res.Frame)
		if res.Frame.Count() == 0 {
			return
		}
	case res.Variant == ResponseVariantAck && t.accumulatedError != nil:
		res.Ack = false
	}
	t.Out.Inlet() <- res
}

func (t *calculationTransform) calculate(
	ctx context.Context,
	fr framer.Frame,
) framer.Frame {
	var err error
	for _, c := range t.calculators {
		if fr, _, err = c.Next(ctx, fr, fr); err != nil {
			t.accumulatedError = err
		}
	}
	return fr.KeepKeys(t.keepKeys)
}
