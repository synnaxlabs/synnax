// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package confluence

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// DownsampleFunc applies to each series in a frame and returns downsampled series.
type DownSampleFunc func(series telem.Series, factor int) telem.Series

// Downsampler is a segment that reads values from an input Stream, downsamples them
// using a provided downsampling function, and publishes the downsampled values to its
// outlets. // TODO: make generic
type DownSampler struct {
	AbstractLinear[framer.Frame, framer.Frame]                     // TODO what is Abstract linear
	Factors                      map[channel.Key]int // Per-series down sampling factors
	CustomDownSampler            DownSampleFunc
}

func (d *DownSampler) OutTo(inlets ...Inlet[framer.Frame]) {
	if len(inlets) != 1 {
		panic("[confluence.DownSampler] - must have exactly one inlet")
	}
	d.AbstractLinear.OutTo(inlets[0])
}

func (d *DownSampler) FLow(ctx signal.Context, opts ...Option) {
	fo := NewOptions(opts)
	fo.AttachClosables(d.Out)
	d.GoRange(ctx, d.downsample, fo.Signal...)
}

func (d *DownSampler) downsample(ctx context.Context, frame framer.Frame) error {
	downSampledFrame := frame
	for i, key := frame.Keys {
		if factor, ok := d.Factors[key]; ok {
			downSampledFrame.Series[i] = d.CustomDownSampler(frame.Series[i], factor)
		} else {
			downSampledFrame.Series[i] = frame.Series[i]
		}
	}
	return signal.SendUnderContext(ctx, d.Out.Inlet(), downSampledFrame)
}

func downSampleSeries(series telem.Series, factor int) telem.Series {
	if factor <= 1 || len(series.Data) <= factor {
		return series
	}

	downsampled := telem.Series{
		TimeRange: series.TimeRange,
		Data: make([]telem.Point, 0, len(series.Data)/factor),
		TimeStamps: make([]time.Time, 0, len(series.Data)/factor),
	}

	for i := 0; i < len(series.Data); i += factor {
		downsampled.Data = append(downsampled.Data, series.Data[i])
		downsampled.TimeStamps = append(downsampled.TimeStamps, series.TimeStamps[i])
	}
	return downsampled;
}
