// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constant

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/x/maps"
	"github.com/synnaxlabs/x/query"
)

var (
	symbolName = "constant"
	symbol     = ir.Symbol{
		Name: symbolName,
		Kind: ir.KindStage,
		Type: ir.Stage{
			Config: maps.Ordered[string, ir.Type]{
				Keys:   []string{"value"},
				Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{})},
			},
			Outputs: maps.Ordered[string, ir.Type]{
				Keys:   []string{ir.DefaultOutput},
				Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{})},
			},
		},
	}
	Resolver = ir.MapResolver{symbolName: symbol}
)

type constant struct{}

func (c constant) Init(_ context.Context, changed func(output string)) {
	changed(ir.DefaultOutput)
}

func (c constant) Next(context.Context, func(output string)) {}

type constantFactory struct{}

func (c *constantFactory) Create(cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.NotFound
	}
	return constant{}, nil
}
