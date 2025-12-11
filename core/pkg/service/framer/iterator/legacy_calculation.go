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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/legacy"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

type legacyCalculationTransform struct {
	confluence.LinearTransform[Response, Response]
	calculators      []*legacy.Calculator
	accumulatedError error
}

func newLegacyCalculationTransform(
	calculators []*legacy.Calculator,
) ResponseSegment {
	t := &legacyCalculationTransform{calculators: calculators}
	t.Transform = t.transform
	return t
}

func (t *legacyCalculationTransform) transform(
	_ context.Context,
	res Response,
) (Response, bool, error) {
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

	for _, c := range t.calculators {
		s, err := c.Next(res.Frame)
		if err != nil {
			t.accumulatedError = err
			continue
		}
		if s.Len() > 0 {
			res.Frame = res.Frame.Append(c.Channel().Key(), s)
		}
	}
	return res, true, nil
}

func (s *Service) newLegacyCalculationTransform(ctx context.Context, cfg *Config) (ResponseSegment, error) {
	var (
		channels   []channel.Channel
		calculated = make(set.Mapped[channel.Key, channel.Channel], len(channels))
		required   = make(set.Mapped[channel.Key, channel.Channel], len(channels))
	)
	if err := s.cfg.Channel.NewRetrieve().
		WhereKeys(cfg.Keys...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	for _, ch := range channels {
		if ch.IsCalculated() && ch.IsLegacyCalculated() {
			calculated[ch.Key()] = ch
			required.Add(ch.Requires...)
		}
	}
	hasCalculated := len(calculated) > 0
	if !hasCalculated {
		return nil, nil
	}
	cfg.Keys = lo.Filter(cfg.Keys, func(item channel.Key, index int) bool {
		return !calculated.Contains(item)
	})
	cfg.Keys = append(cfg.Keys, required.Keys()...)
	var requiredCh []channel.Channel
	err := s.cfg.Channel.NewRetrieve().
		WhereKeys(required.Keys()...).
		Entries(&requiredCh).
		Exec(ctx, nil)
	if err != nil {
		return nil, err
	}
	calculators := make([]*legacy.Calculator, len(calculated))
	for i, v := range calculated.Values() {
		calculators[i], err = legacy.OpenCalculator(v, requiredCh)
		if err != nil {
			return nil, err
		}
	}
	return newLegacyCalculationTransform(calculators), nil
}
