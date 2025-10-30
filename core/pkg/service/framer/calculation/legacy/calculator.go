// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/computron"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type cacheEntry struct {
	ch   channel.Channel
	data telem.MultiSeries
}

// Calculator is an extension of the lua-based computron.Calculator to provide
// specific functionality for evaluating calculations on channels using frame data.
type Calculator struct {
	// base is the underlying computron calculator.
	base *computron.Calculator
	// ch is the calculated channel we're operating on.
	ch channel.Channel
	// highWaterMark is the high-water mark of the sample that we've run the calculation on.
	highWaterMark struct {
		initialized bool
		alignment   telem.Alignment
		timestamp   telem.TimeStamp
	}
	// cache is a map of required channels and an accumulated buffer of data. Data
	// is accumulated for each channel until a calculation can be performed, and is
	// then flushed.
	cache map[channel.Key]cacheEntry
}

// OpenCalculator opens a new calculator that evaluates the Expression of the provided
// channel. The requiredChannels provided must include ALL and ONLY the channels
// corresponding to the keys specified in ch.Requires.
//
// The calculator must be closed by calling Close() after use, or memory leaks will occur.
func OpenCalculator(
	ch channel.Channel,
	requiredChannels []channel.Channel,
) (*Calculator, error) {
	base, err := computron.Open(ch.Expression)
	if err != nil {
		return nil, err
	}
	required := make(map[channel.Key]cacheEntry, len(requiredChannels))
	for _, requiredCh := range requiredChannels {
		required[requiredCh.Key()] = cacheEntry{ch: requiredCh}
	}
	c := &Calculator{
		base:  base,
		ch:    ch,
		cache: required,
	}
	c.resetCache()
	return c, nil
}

// Channel returns information about the channel being calculated.
func (c *Calculator) Channel() channel.Channel { return c.ch }

func (c *Calculator) resetCache() {
	c.highWaterMark.timestamp = 0
	c.highWaterMark.alignment = 0
	c.highWaterMark.initialized = false
	for k, e := range c.cache {
		e.data = telem.MultiSeries{}
		c.cache[k] = e
	}
}

func (c *Calculator) emptySeries() telem.Series {
	return telem.Series{DataType: c.ch.DataType}
}

func cacheMisalignmentError(requested telem.Alignment, actualBounds telem.AlignmentBounds) error {
	return errors.Newf(
		`attempted to run calculation on alignment %s, but cache
with alignment bounds %s did not contain data for requested alignment. Fixing issue by resetting cache.`,
		requested, actualBounds,
	)
}

// Next executes the next calculation step. It takes in the given frame and determines
// if enough data is available to perform the next set of calculations. The returned
// telem.Series will have a length equal to the number of new calculations completed.
// If no calculations are completed, the length of the series will be 0, and the caller
// is free to discard the returned value.
//
// Any error encountered during calculations is returned as well.
func (c *Calculator) Next(fr framer.Frame) (telem.Series, error) {
	// Handle calculations with no cache channels (constants only)
	if len(c.cache) == 0 {
		return telem.Series{DataType: c.ch.DataType}, nil
	}

	for rawI, s := range fr.RawSeries() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		k := fr.RawKeyAt(rawI)
		if v, ok := c.cache[k]; ok {
			// The first case means we've never initialized a high-water mark, and second
			// case mean's we've switched domains. In both, reset the high water-mark.
			if !c.highWaterMark.initialized ||
				c.highWaterMark.alignment.DomainIndex() != s.AlignmentBounds().Lower.DomainIndex() {
				c.resetCache()
				c.highWaterMark.initialized = true
				c.highWaterMark.alignment = s.AlignmentBounds().Lower
				c.highWaterMark.timestamp = s.TimeRange.Start
			}
			v.data = v.data.Append(s).FilterGreaterThanOrEqualTo(c.highWaterMark.alignment)
			c.cache[k] = v
		}
	}
	minAlignment := telem.MaxAlignment
	minTimeStamp := telem.TimeStamp(0)
	for _, v := range c.cache {
		if v.data.AlignmentBounds().Upper < minAlignment {
			minAlignment = v.data.AlignmentBounds().Upper
			minTimeStamp = v.data.TimeRange().End
		}
	}
	// Return early if there's no data to process
	if minAlignment == telem.MaxAlignment || minAlignment <= c.highWaterMark.alignment {
		return c.emptySeries(), nil
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
		for _, v := range c.cache {
			if !v.data.AlignmentBounds().Contains(a) {
				c.resetCache()
				return c.emptySeries(), cacheMisalignmentError(a, v.data.AlignmentBounds())
			}
			c.base.Set(v.ch.Name, computron.LValueFromMultiSeriesAlignment(v.data, a))
		}
		v, err := c.base.Run()
		if err != nil {
			return c.emptySeries(), err
		}
		computron.SetLValueOnSeries(v, os, int(a-startAlign))
	}
	return os, nil
}

// Close closes the calculator, releasing internal resources. No other methods can be
// called on the calculator after Close has been called.
func (c *Calculator) Close() { c.base.Close() }
