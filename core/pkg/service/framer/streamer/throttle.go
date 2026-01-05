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
	"time"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type throttle struct {
	rate telem.Rate
	fr   frame.Frame
	confluence.LinearTransform[Response, Response]
}

func newThrottle(cfg Config) responseSegment {
	t := &throttle{
		rate: cfg.ThrottleRate,
		fr:   frame.Frame{},
	}
	return t
}

func (t *throttle) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(t.Out)
	sCtx.Go(func(ctx context.Context) error {
		ticker := time.NewTicker(t.rate.Period().Duration())
		first := true
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				if t.fr.Len() > 0 {
					if err := signal.SendUnderContext(
						sCtx,
						t.Out.Inlet(),
						Response{Frame: t.fr},
					); err != nil {
						return err
					}
					t.fr = frame.Alloc(t.fr.Count())
				}
			case res, ok := <-t.In.Outlet():
				if first && res.Frame.Empty() {
					first = false
					if err := signal.SendUnderContext(
						sCtx,
						t.Out.Inlet(),
						res,
					); err != nil {
						return err
					}
				}
				if !ok {
					return nil
				}
				t.fr = t.fr.Extend(res.Frame)
			}
		}
	}, o.Signal...)
}
