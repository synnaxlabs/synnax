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
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/x/confluence"
)

type calculationTransform struct {
	confluence.LinearTransform[framer.IteratorResponse, framer.IteratorResponse]
	calculators []*calculation.Calculator
}

func newCalculationTransform(
	calculators []*calculation.Calculator,
) ResponseSegment {
	t := &calculationTransform{calculators: calculators}
	t.Transform = t.transform
	return t
}

func (t *calculationTransform) transform(
	_ context.Context,
	res framer.IteratorResponse,
) (framer.IteratorResponse, bool, error) {
	for _, c := range t.calculators {
		s, err := c.Next(res.Frame)
		if err != nil {
			return framer.IteratorResponse{}, false, err
		}
		if s.Len() > 0 {
			res.Frame = res.Frame.Append(c.Channel().Key(), s)
		}
	}
	return res, true, nil
}
