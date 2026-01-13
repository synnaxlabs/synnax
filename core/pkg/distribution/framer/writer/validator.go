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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

type validator struct {
	keys      channel.Keys
	responses struct {
		confluence.AbstractUnarySource[Response]
		confluence.NopFlow
	}
	confluence.AbstractLinear[Request, Request]
	seqNum int
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
			case req, ok := <-v.In.Outlet():
				if !ok {
					return nil
				}
				v.seqNum++
				req.SeqNum = v.seqNum
				if err := v.validate(req); err != nil {
					return err
				} else {
					if err = signal.SendUnderContext(ctx, v.Out.Inlet(), req); err != nil {
						return err
					}
					continue
				}
			}
		}
	}, o.Signal...)
}

func (v *validator) validate(req Request) error {
	if err := validateCommand(req.Command); err != nil {
		return err
	}
	if req.Command == CommandWrite {
		for rawI, k := range req.Frame.RawKeys() {
			if req.Frame.ShouldExcludeRaw(rawI) {
				continue
			}
			if !lo.Contains(v.keys, k) {
				return errors.Wrapf(validate.ErrValidation, "invalid key: %s", k)
			}
		}
	}
	return nil
}
