// Copyright 2025 Synnax Labs, Inc.
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

type downSampler struct {
	confluence.LinearTransform[Response, Response]
	cfg Config
}

func newDownSampler(cfg Config) responseSegment {
	d := &downSampler{cfg: cfg}
	d.Transform = d.transform
	return d
}

func (d *downSampler) transform(
	_ context.Context,
	in Response,
) (out Response, ok bool, err error) {
	in.Frame = in.Frame.ShallowCopy()
	for i, s := range in.Frame.SeriesI() {
		in.Frame.SetSeriesAt(i, s.DownSample(d.cfg.DownSampleFactor))
	}
	return in, true, nil
}
