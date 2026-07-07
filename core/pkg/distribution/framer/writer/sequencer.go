// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"

	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

// sequencer stamps every request entering the writer pipeline with a monotonically
// increasing sequence number, which the synchronizer uses to correlate acknowledgements
// from the writer's peer and gateway branches. Request validation happens downstream,
// as close to storage as possible: frame keys that cannot be routed are rejected by the
// peer switch, and everything else is validated by the storage engine on the node that
// services each channel, against its own authoritative channel record.
type sequencer struct {
	confluence.AbstractLinear[Request, Request]
	responses struct {
		confluence.NopFlow
		confluence.AbstractUnarySource[Response]
	}
	seqNum int
}

// Flow implements the confluence.Flow interface.
func (s *sequencer) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.responses.Out, s.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-s.In.Outlet():
				if !ok {
					return nil
				}
				s.seqNum++
				req.SeqNum = s.seqNum
				if err := signal.SendUnderContext(ctx, s.Out.Inlet(), req); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}
