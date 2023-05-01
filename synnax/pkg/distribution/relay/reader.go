// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type reader struct {
	addr      address.Address
	requests  confluence.AbstractLinear[ReadRequest, demand]
	responses confluence.AbstractLinear[Data, Data]
	keys      channel.Keys
	confluence.Source[Data]
	confluence.Sink[ReadRequest]
}

func (r *reader) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(r.responses.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req := <-r.requests.In.Outlet():
				r.keys = req.Keys
				r.requests.Out.Inlet() <- demand{Variant: change.Set, Key: r.addr, Value: req}
			case f, ok := <-r.responses.In.Outlet():
				if !ok {
					r.requests.Out.Inlet() <- change.Change[address.Address, ReadRequest]{Variant: change.Delete, Key: r.addr}
					return nil
				}
				r.responses.Out.Inlet() <- Data{
					Error: f.Error,
					Frame: f.Frame.FilterKeys(r.keys),
				}
			}
		}
	}, o.Signal...)
}
