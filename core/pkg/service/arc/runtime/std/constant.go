// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package std

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/x/maps"
	"github.com/synnaxlabs/x/signal"
)

var symbolConstant = ir.Symbol{
	Name: "constant",
	Kind: ir.KindStage,
	Type: ir.Stage{
		Config: maps.Ordered[string, ir.Type]{
			Keys:   []string{"value"},
			Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{})},
		},
		Return: ir.NewTypeVariable("T", ir.NumericConstraint{}),
	},
}

type constant struct {
	base
	value value.Value
}

func (c *constant) Flow(ctx signal.Context) {
	c.outputHandler(ctx, "output", c.value)
}

func newConstant(_ context.Context, cfg Config) (stage.Stage, error) {
	// The actual type will be determined by the graph analysis
	// For runtime, we just store the value
	c := &constant{
		base:  base{key: cfg.Node.Key},
		value: value.Value{}.Put(cfg.Node.Config["value"]),
	}
	return c, nil
}
