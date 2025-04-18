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
	"github.com/synnaxlabs/x/computron"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/telem"
)

type calculator struct {
	c                *computron.Calculator
	ch               channel.Channel
	hwm              telem.AlignmentPair
	requiredValues   map[channel.Key]telem.MultiSeries
	requiredChannels map[channel.Key]channel.Channel
}

func newCalculator(
	c *computron.Calculator,
	ch channel.Channel,
	required map[channel.Key]channel.Channel,
) *calculator {
	requiredValues := make(map[channel.Key]telem.MultiSeries, len(required))
	for k := range required {
		requiredValues[k] = telem.MultiSeries{}
	}
	return &calculator{
		c:                c,
		ch:               ch,
		hwm:              0,
		requiredValues:   requiredValues,
		requiredChannels: required,
	}
}

func (c *calculator) Next(fr framer.Frame) (telem.Series, error) {
	minAlignment := telem.MaxAlignmentPair
	for i, k := range fr.Keys {
		if v, ok := c.requiredValues[k]; !ok {
			v = v.Append(fr.Series[i]).KeepGreaterThan(c.hwm)
			c.requiredValues[k] = v
			if v.AlignmentBounds().Upper < minAlignment {
				minAlignment = v.AlignmentBounds().Upper
			}
		}
	}
	if minAlignment <= c.hwm {
		return telem.Series{}, nil
	}
	var (
		start = c.hwm
		end   = minAlignment
		os    = telem.AllocSeries(c.ch.DataType, int64(end-start))
	)
	c.hwm = minAlignment
	for i := start; i <= end; i++ {
		for k, v := range c.requiredValues {
			ch := c.requiredChannels[k]
			c.c.Set(ch.Name, computron.LValueFromMultiSeriesAlignment(v, i))
		}
		v, err := c.c.Run()
		if err != nil {
			return telem.Series{}, err
		}
		computron.SetLValueOnSeries(v, os, int64(i-start))
	}
	return os, nil
}

type calculationTransform struct {
	confluence.LinearTransform[framer.IteratorResponse, framer.IteratorResponse]
	calculators []*calculator
}

func newCalculationTransform(
	calculators []*calculator,
) ResponseSegment {
	t := &calculationTransform{calculators: calculators}
	t.Transform = t.transform
	return t
}

func (t *calculationTransform) transform(_ context.Context, req framer.IteratorResponse) (framer.IteratorResponse, bool, error) {
	for _, c := range t.calculators {
		s, err := c.Next(req.Frame)
		if err != nil {
			return framer.IteratorResponse{}, false, err
		}
		if s.Len() > 0 {
			req.Frame.Series = append(req.Frame.Series, s)
			req.Frame.Keys = append(req.Frame.Keys, c.ch.Key())
		}
	}
	return req, true, nil
}
