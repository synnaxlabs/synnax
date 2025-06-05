// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package reactive

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
)

type constant struct {
	confluence.AbstractUnarySource[spec.Value]
	value spec.Value
}

func newConstant(_ context.Context, cfg factoryConfig) (bool, error) {
	if cfg.node.Type != spec.ConstantType {
		return false, nil
	}
	value := cfg.node.Data["value"]
	c := &constant{
		value: spec.Value{
			DataType: string(cfg.node.Schema.Data["value"].Type),
			Value:    value,
		},
	}
	plumber.SetSource[spec.Value](cfg.pipeline, address.Address(cfg.node.Key), c)
	return true, nil
}

func (n *constant) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	sCtx.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case n.Out.Inlet() <- n.value:
		}
		<-ctx.Done()
		return ctx.Err()
	}, o.Signal...)
}
