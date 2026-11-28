// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package streamer

import (
	"context"

	"github.com/synnaxlabs/x/confluence"
)

type downsampler struct {
	confluence.LinearTransform[Response, Response]
	cfg Config
	// applied is the Applied of the responses that transform dropped, which the next
	// response it sends carries.
	applied int
}

func newDownsampler(cfg Config) responseSegment {
	d := &downsampler{cfg: cfg}
	d.Transform = d.transform
	return d
}

func (d *downsampler) transform(
	_ context.Context,
	in Response,
) (out Response, ok bool, err error) {
	hadData := in.Frame.HasData()
	in.Frame = in.Frame.ShallowCopy()
	for i, s := range in.Frame.SeriesI() {
		in.Frame.SetSeriesAt(i, s.Downsample(d.cfg.DownsampleFactor))
	}
	// A frame whose samples all lie off the downsampling grid carries nothing to send.
	if hadData && !in.Frame.HasData() {
		d.applied += in.Applied
		return in, false, nil
	}
	in.Applied += d.applied
	d.applied = 0
	return in, true, nil
}
