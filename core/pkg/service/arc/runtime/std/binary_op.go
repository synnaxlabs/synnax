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

func createComparisonSymbol(name string) ir.Symbol {
	params := maps.Ordered[string, ir.Type]{}
	params.Put("a", ir.NewTypeVariable("T", ir.NumericConstraint{}))
	params.Put("b", ir.NewTypeVariable("T", ir.NumericConstraint{}))
	return ir.Symbol{
		Name: name,
		Kind: ir.KindStage,
		Type: ir.Stage{Params: params, Return: ir.U8{}},
	}
}

var (
	symbolGE = createComparisonSymbol("ge")
	symbolLE = createComparisonSymbol("le")
	symbolLT = createComparisonSymbol("lt")
	symbolGT = createComparisonSymbol("gt")
	symbolEQ = createComparisonSymbol("eq")
	symbolNE = createComparisonSymbol("ne")
)

type operator struct {
	base
	compare func(a, b value.Value) bool
	a, b    *value.Value
}

func (n *operator) Next(ctx context.Context, param string, val value.Value) {
	if param == "a" {
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
		n.outputHandler(ctx, "output", value.Value{Type: ir.U8{}}.PutUint8(result))
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

type arithmeticOperator struct {
	base
	operate func(a, b value.Value) value.Value
	a, b    *value.Value
}

func (n *arithmeticOperator) Next(ctx context.Context, param string, val value.Value) {
	if param == "a" {
		n.a = &val
	} else {
		n.b = &val
	}
	if n.a != nil && n.b != nil {
		result := n.operate(*n.a, *n.b)
		n.outputHandler(ctx, "output", result)
	}
}

func createArithmeticOpFactory(operate func(a, b value.Value) value.Value) Constructor {
	return func(_ context.Context, cfg Config) (stage.Stage, error) {
		o := &arithmeticOperator{operate: operate}
		o.key = cfg.Node.Key
		return o, nil
	}
}

// createArithmeticSymbol creates a polymorphic arithmetic operator symbol
func createArithmeticSymbol(name string) ir.Symbol {
	return ir.Symbol{
		Name: name,
		Kind: ir.KindStage,
		Type: ir.Stage{
			Params: maps.Ordered[string, ir.Type]{
				Keys: []string{"a", "b"},
				Values: []ir.Type{
					ir.NewTypeVariable("T", ir.NumericConstraint{}),
					ir.NewTypeVariable("T", ir.NumericConstraint{}),
				},
			},
			Return: ir.NewTypeVariable("T", ir.NumericConstraint{}),
		},
	}
}

var (
	symbolAdd = createArithmeticSymbol("add")
	symbolSub = createArithmeticSymbol("sub")
	symbolMul = createArithmeticSymbol("mul")
	symbolDiv = createArithmeticSymbol("div")
	symbolMod = createArithmeticSymbol("mod")
)

var (
	AddFactory = createArithmeticOpFactory(func(a, b value.Value) value.Value { return a.Add(b) })
	SubFactory = createArithmeticOpFactory(func(a, b value.Value) value.Value { return a.Sub(b) })
	MulFactory = createArithmeticOpFactory(func(a, b value.Value) value.Value { return a.Mul(b) })
	DivFactory = createArithmeticOpFactory(func(a, b value.Value) value.Value { return a.Div(b) })
	ModFactory = createArithmeticOpFactory(func(a, b value.Value) value.Value { return a.Mod(b) })
)
