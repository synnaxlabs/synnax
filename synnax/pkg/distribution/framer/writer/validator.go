// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

type validator struct {
	signal    chan bool
	closed    bool
	keys      channel.Keys
	responses struct {
		confluence.AbstractUnarySource[Response]
		confluence.NopFlow
	}
	confluence.AbstractLinear[Request, Request]
	accumulatedError error
	seqNum           int
}

// Flow implements the confluence.Flow interface.
func (v *validator) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(v.responses.Out, v.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case v.closed = <-v.signal:
			case req, ok := <-v.In.Outlet():
				if !ok {
					return nil
				}
				v.seqNum++
				res := Response{Command: req.Command, SeqNum: v.seqNum, Ack: true}
				block := v.closed && (req.Command == Data || req.Command == Commit)
				if v.accumulatedError != nil || block {
					res.Error = v.accumulatedError
					res.Ack = false
				} else if v.accumulatedError = v.validate(req); v.accumulatedError != nil {
					res.Ack = false
				} else {
					if err := signal.SendUnderContext(ctx, v.Out.Inlet(), req); err != nil {
						return err
					}
					continue
				}
				if err := signal.SendUnderContext(ctx, v.responses.Out.Inlet(), res); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}

func (v *validator) validate(req Request) error {
	if req.Command < Data || req.Command > SetAuthority {
		return errors.Wrapf(validate.Error, "invalid writer command: %d", req.Command)
	}
	if req.Command == Data {
		for _, k := range req.Frame.Keys {
			if !lo.Contains(v.keys, k) {
				return errors.Wrapf(validate.Error, "invalid key: %s", k)
			}
		}

	}
	return nil
}
