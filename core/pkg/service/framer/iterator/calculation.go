// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
)

type calculationTransform struct {
	confluence.LinearTransform[framer.IteratorResponse, framer.IteratorResponse]
	excludeKeys      channel.Keys
	calculators      []*calculator.Calculator
	accumulatedError error
}

func newCalculationTransform(
	excludeKeys channel.Keys,
	calculators []*calculator.Calculator,
) *calculationTransform {
	t := &calculationTransform{calculators: calculators, excludeKeys: excludeKeys}
	t.Transform = t.transform
	return t
}

func (t *calculationTransform) close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, calc := range t.calculators {
		c.Exec(calc.Close)
	}
	return c.Error()
}

func (t *calculationTransform) transform(
	ctx context.Context,
	res framer.IteratorResponse,
) (framer.IteratorResponse, bool, error) {
	if res.Command == Error {
		res.Error = errors.Combine(res.Error, t.accumulatedError)
		return res, true, nil
	}
	if res.Variant == AckResponse {
		if t.accumulatedError != nil {
			res.Ack = false
		}
		return res, true, nil
	}

	var err error
	for _, c := range t.calculators {
		res.Frame, _, err = c.Next(ctx, res.Frame, res.Frame)
		if err != nil {
			t.accumulatedError = err
			continue
		}
	}
	res.Frame = res.Frame.ExcludeKeys(t.excludeKeys)
	if res.Frame.Count() == 0 {
		return framer.IteratorResponse{}, false, nil
	}
	return res, true, nil
}
