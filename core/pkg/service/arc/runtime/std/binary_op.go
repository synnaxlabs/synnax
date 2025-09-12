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
	compare func(a, b value.Value) bool
	a, b    *value.Value
}

func (n *operator) Next(ctx context.Context, val value.Value) {
	if val.Param == "a" {
		n.a = &val
	} else {
		n.b = &val
	}
	if n.a != nil && n.b != nil {
		var result uint8
		if n.compare(*n.a, *n.b) {
			result = 1
		} else {
			result = 0
		}
		n.outputHandler(ctx, value.Value{
			Address: val.Address,
			Param:   "",
			Type:    ir.U8{},
		}.PutUint8(result))
	}
}

func createBinaryOpFactory(compare func(a, b value.Value) bool) Constructor {
	return func(_ context.Context, cfg Config) (stage.Stage, error) {
		o := &operator{compare: compare}
		o.key = cfg.Node.Key
		return o, nil
	}
}

// Comparison operator factories
var (
	GEFactory = createBinaryOpFactory(func(a, b value.Value) bool { return a.Ge(b) })
	LEFactory = createBinaryOpFactory(func(a, b value.Value) bool { return a.Le(b) })
	LTFactory = createBinaryOpFactory(func(a, b value.Value) bool { return a.Lt(b) })
	GTFactory = createBinaryOpFactory(func(a, b value.Value) bool { return a.Gt(b) })
	EQFactory = createBinaryOpFactory(func(a, b value.Value) bool { return a.Eq(b) })
	NEFactory = createBinaryOpFactory(func(a, b value.Value) bool { return !a.Eq(b) })
)

// Arithmetic operator types and symbols
type arithmeticOperator struct {
	base
	operate func(a, b value.Value) value.Value
	a, b    *value.Value
}

func (n *arithmeticOperator) Next(ctx context.Context, val value.Value) {
	if val.Param == "a" {
		n.a = &val
	} else {
		n.b = &val
	}
	if n.a != nil && n.b != nil {
		result := n.operate(*n.a, *n.b)
		result.Param = "output"
		n.outputHandler(ctx, result)
		n.a = nil
		n.b = nil
	}
}

func createArithmeticOpFactory(operate func(a, b value.Value) value.Value) Constructor {
	return func(_ context.Context, cfg Config) (stage.Stage, error) {
		o := &arithmeticOperator{operate: operate}
		o.key = cfg.Node.Key
		return o, nil
	}
}

// Arithmetic operator factories
var (
	AddFactory = createArithmeticOpFactory(func(a, b value.Value) value.Value { return a.Add(b) })
	SubFactory = createArithmeticOpFactory(func(a, b value.Value) value.Value { return a.Sub(b) })
	MulFactory = createArithmeticOpFactory(func(a, b value.Value) value.Value { return a.Mul(b) })
	DivFactory = createArithmeticOpFactory(func(a, b value.Value) value.Value { return a.Div(b) })
	ModFactory = createArithmeticOpFactory(func(a, b value.Value) value.Value { return a.Mod(b) })
)
