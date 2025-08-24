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

type requiredInfo struct {
	ch   channel.Channel
	data telem.MultiSeries
}

// Calculator is an extension of the lua-based computron.Calculator to provide specific
// functionality for evaluating calculations on channels using frame data.
type Calculator struct {
	// base is the underlying computron calculator.
	base *computron.Calculator
	// ch is the calculated channel we're operating on.
	ch channel.Channel
	// highWaterMark is the high-water mark of the sample that we've run the calculation
	// on.
	highWaterMark struct {
		alignment telem.Alignment
		timestamp telem.TimeStamp
	}
	// required is a map of required channels and an accumulated buffer of data. Data is
	// accumulated for each channel until a calculation can be performed, and is then
	// flushed.
	required map[channel.Key]requiredInfo
}

// OpenCalculator opens a new calculator that evaluates the Expression of the provided
// channel. The requiredChannels provided must include ALL and ONLY the channels
// corresponding to the keys specified in ch.Requires.
//
// The calculator must be closed by calling Close() after use, or memory leaks will
// occur.
func OpenCalculator(
	ch channel.Channel,
	requiredChannels []channel.Channel,
) (*Calculator, error) {
	base, err := computron.Open(ch.Expression)
	if err != nil {
		return nil, err
	}
	required := make(map[channel.Key]requiredInfo, len(requiredChannels))
	for _, requiredCh := range requiredChannels {
		required[requiredCh.Key()] = requiredInfo{ch: requiredCh}
	}
	return &Calculator{
		base:     base,
		ch:       ch,
		required: required,
	}, nil
}

// Channel returns information about the channel being calculated.
func (c *Calculator) Channel() channel.Channel { return c.ch }

// Next executes the next calculation step. It takes in the given frame and determines
// if enough data is available to perform the next set of calculations. The returned
// telem.Series will have a length equal to the number of new calculations completed. If
// no calculations are completed, the length of the series will be 0, and the caller is
// free to discard the returned value.
//
// Any error encountered during calculations is returned as well.
func (c *Calculator) Next(fr framer.Frame) (telem.Series, error) {
	for rawI, s := range fr.RawSeries() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		k := fr.RawKeyAt(rawI)
		if v, ok := c.required[k]; ok {
			v.data = v.data.Append(s).FilterGreaterThanOrEqualTo(c.highWaterMark.alignment)
			c.required[k] = v
			if c.highWaterMark.alignment == 0 {
				c.highWaterMark.alignment = v.data.AlignmentBounds().Lower
				c.highWaterMark.timestamp = v.data.TimeRange().Start
			}
		}
	}
	minAlignment := telem.MaxAlignment
	minTimeStamp := telem.TimeStamp(0)
	for _, v := range c.required {
		if v.data.AlignmentBounds().Upper < minAlignment {
			minAlignment = v.data.AlignmentBounds().Upper
			minTimeStamp = v.data.TimeRange().End
		}
	}
	if minAlignment <= c.highWaterMark.alignment {
		return telem.Series{DataType: c.ch.DataType}, nil
	}
	var (
		startAlign = c.highWaterMark.alignment
		startTS    = c.highWaterMark.timestamp
		endAlign   = minAlignment
		os         = telem.MakeSeries(c.ch.DataType, int(endAlign-startAlign))
	)
	c.highWaterMark.alignment = minAlignment
	c.highWaterMark.timestamp = minTimeStamp
	os.Alignment = startAlign
	os.TimeRange = telem.TimeRange{Start: startTS, End: minTimeStamp}
	for a := startAlign; a < endAlign; a++ {
		for _, v := range c.required {
			c.base.Set(v.ch.Name, computron.LValueFromMultiSeriesAlignment(v.data, a))
		}
		v, err := c.base.Run()
		if err != nil {
			return telem.Series{DataType: c.ch.DataType}, err
		}
		computron.SetLValueOnSeries(v, os, int(a-startAlign))
	}
	return os, nil
}

// Close closes the calculator, releasing internal resources. No other methods can be
// called on the calculator after Close has been called.
func (c *Calculator) Close() { c.base.Close() }
