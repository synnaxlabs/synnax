// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/computron"
	"github.com/synnaxlabs/x/telem"
)

type Calculator struct {
	base             *computron.Calculator
	ch               channel.Channel
	hwm              telem.Alignment
	requiredValues   map[channel.Key]telem.MultiSeries
	requiredChannels map[channel.Key]channel.Channel
}

func NewCalculator(
	base *computron.Calculator,
	ch channel.Channel,
	required map[channel.Key]channel.Channel,
) *Calculator {
	requiredValues := make(map[channel.Key]telem.MultiSeries, len(required))
	for k := range required {
		requiredValues[k] = telem.MultiSeries{}
	}
	return &Calculator{
		base:             base,
		ch:               ch,
		hwm:              0,
		requiredValues:   requiredValues,
		requiredChannels: required,
	}
}

func (c *Calculator) Channel() channel.Channel {
	return c.ch
}

func (c *Calculator) Next(fr framer.Frame) (telem.Series, error) {
	for rawI, s := range fr.RawSeries() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		k := fr.RawKeyAt(rawI)
		if v, ok := c.requiredValues[k]; ok {
			v = v.Append(s).FilterLessThan(c.hwm)
			c.requiredValues[k] = v
			if c.hwm == 0 {
				c.hwm = v.AlignmentBounds().Lower
			}
		}
	}
	minAlignment := telem.MaxAlignmentPair
	for _, v := range c.requiredValues {
		if v.AlignmentBounds().Upper < minAlignment {
			minAlignment = v.AlignmentBounds().Upper
		}
	}
	if minAlignment <= c.hwm {
		return telem.Series{DataType: c.ch.DataType}, nil
	}
	var (
		start = c.hwm
		end   = minAlignment
		os    = telem.AllocSeries(c.ch.DataType, int64(end-start))
	)
	c.hwm = minAlignment
	for i := start; i < end; i++ {
		for k, v := range c.requiredValues {
			ch := c.requiredChannels[k]
			c.base.Set(ch.Name, computron.LValueFromMultiSeriesAlignment(v, i))
		}
		v, err := c.base.Run()
		if err != nil {
			return telem.Series{}, err
		}
		computron.SetLValueOnSeries(v, os, int64(i-start))
	}
	return os, nil
}

func (c *Calculator) Close() {
	c.base.Close()
}
