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

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator/calculation"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
)

type calculationTransform struct {
	confluence.LinearTransform[framer.IteratorResponse, framer.IteratorResponse]
	calculators      []*calculation.Calculator
	accumulatedError error
}

func newCalculationTransform(
	calculators []*calculation.Calculator,
) ResponseSegment {
	t := &calculationTransform{calculators: calculators}
	t.Transform = t.transform
	return t
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
		res.Frame, err = c.Next(ctx, res.Frame)
		if err != nil {
			t.accumulatedError = err
			continue
		}
	}
	return res, true, nil
}
