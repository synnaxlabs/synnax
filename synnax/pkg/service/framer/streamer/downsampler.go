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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/confluence"
)

type downSampler struct {
	confluence.LinearTransform[framer.StreamerResponse, framer.StreamerResponse]
	cfg Config
}

func newDownSampler(cfg Config) confluence.Segment[framer.StreamerResponse, framer.StreamerResponse] {
	return &downSampler{cfg: cfg}
}

func (d *downSampler) transform(
	_ context.Context,
	in framer.StreamerResponse,
) (out framer.StreamerResponse, ok bool, err error) {
	in.Frame = in.Frame.ShallowCopy()
	for i, s := range in.Frame.Series {
		in.Frame.Series[i] = s.DownSample(d.cfg.DownSampleFactor)
	}
	return in, true, nil
}
