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
	"github.com/synnaxlabs/x/telem"
)

type throttle struct {
	confluence.LinearTransform[Response, Response]
	cfg   Config
	frame framer.Frame
	last  telem.TimeStamp
}

func newThrottle(cfg Config) responseSegment {
	t := &throttle{cfg: cfg}
	t.Transform = t.transform
	return t
}

func (t *throttle) transform(
	_ context.Context,
	in Response,
) (Response, bool, error) {
	t.frame = t.frame.Extend(in.Frame)
	return in, telem.Since(t.last) > t.cfg.ThrottleRate.Period(), nil
}
