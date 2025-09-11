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
	"github.com/synnaxlabs/x/maps"
)

var (
	binaryOpIrType = ir.Stage{
		Params: maps.Ordered[string, ir.Type]{
			Keys:   []string{"a", "b"},
			Values: []ir.Type{ir.Number{}, ir.Number{}},
		},
		Return: ir.U8{},
	}
	symbolGE = ir.Symbol{Name: "ge", Kind: ir.KindStage, Type: binaryOpIrType}
	symbolLE = ir.Symbol{Name: "le", Kind: ir.KindStage, Type: binaryOpIrType}
	symbolLT = ir.Symbol{Name: "lt", Kind: ir.KindStage, Type: binaryOpIrType}
	symbolGT = ir.Symbol{Name: "gt", Kind: ir.KindStage, Type: binaryOpIrType}
	symbolEQ = ir.Symbol{Name: "eq", Kind: ir.KindStage, Type: binaryOpIrType}
	symbolNE = ir.Symbol{Name: "ne", Kind: ir.KindStage, Type: binaryOpIrType}
)

type operator struct {
	base
	compare func(a, b stage.Value) bool
	a, b    *stage.Value
}

func (n *operator) Next(ctx context.Context, value stage.Value) {
	if value.Param == "a" {
		n.a = &value
	} else {
		n.b = &value
	}
	if n.a != nil && n.b != nil {
		n.outputHandler(ctx, stage.Value{
			Param: "output",
			Type:  ir.U8{},
			Value: n.compare(*n.a, *n.b),
		})
	}
}

func createBinaryOpFactory(compare func(a, b uint64) uint64) Constructor {
	return func(_ context.Context, cfg Config) (stage.Stage, error) {
		o := &operator{compare: compare}
		o.key = cfg.Node.Key
		return o, nil
	}
}
