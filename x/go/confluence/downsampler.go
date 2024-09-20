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
	"github.com/synnaxlabs/x/signal"
)

// Downsampler is a segment that reads values from an input Stream, downsamples them
// using a provided downsampling function, and publishes the downsampled values to its
// outlets.
type DownSampler[V Value] struct {
	AbstractLinear[V, V]
	DownSample         DownSampleFunc[V]
	DownSamplingFactor int
}

// DownsampleFunc applies to each series in a frame and returns downsampled series.
type DownSampleFunc[V Value] func(ctx context.Context, v V, factor int) V

func (d *DownSampler[V]) OutTo(inlets ...Inlet[V]) {
	if len(inlets) != 1 {
		panic("[confluence.DownSampler] - must have exactly one inlet")
	}
	d.AbstractLinear.OutTo(inlets[0])
}

// Flow implements the Flow interface?
func (d *DownSampler[V]) Flow(ctx signal.Context, opts ...Option) {
	fo := NewOptions(opts)
	fo.AttachClosables(d.Out)
	d.GoRange(ctx, d.downsample, fo.Signal...)
}

func (d *DownSampler[V]) downsample(ctx context.Context, v V) error {
	downsampled := d.DownSample(ctx, v, d.DownSamplingFactor)
	return signal.SendUnderContext(ctx, d.Out.Inlet(), downsampled)
}
