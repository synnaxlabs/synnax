// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/telem"
)

// phase is where a channel's stride resumes in the series that continues its last one.
type phase struct {
	// upper is the alignment just past the last series the downsampler saw.
	upper telem.Alignment
	// start is the index of the first sample to keep in a series that starts at upper.
	start int64
}

// downsampler keeps every factor-th sample of each channel. The stride continues across
// responses, so splitting a read into pieces keeps the same samples.
type downsampler struct {
	confluence.LinearTransform[Response, Response]
	factor uint32
	// phases holds where the stride of each channel resumes.
	phases map[channel.Key]phase
}

func newDownsampler(factor uint32) responseSegment {
	d := &downsampler{factor: factor, phases: make(map[channel.Key]phase)}
	d.Transform = d.transform
	return d
}

func (d *downsampler) transform(
	_ context.Context,
	in Response,
) (out Response, ok bool, err error) {
	in.Frame = in.Frame.ShallowCopy()
	f := int64(d.factor)
	for i := range in.Frame.Count() {
		key, s := in.Frame.At(i)
		var start int64
		if p, ok := d.phases[key]; ok && p.upper == s.Alignment {
			start = p.start
		}
		d.phases[key] = phase{
			upper: s.AlignmentBounds().Upper,
			start: ((start-s.Len())%f + f) % f,
		}
		in.Frame.SetSeriesAt(i, s.DownsampleFrom(start, d.factor))
	}
	return in, true, nil
}
