// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package op

import (
	"github.com/synnaxlabs/arc/ir"
)

func createBinaryOpSymbol(name string, outputs ir.NamedTypes) ir.Symbol {
	return ir.Symbol{
		Name: name,
		Kind: ir.KindStage,
		Type: ir.Stage{
			Params: ir.NamedTypes{
				Keys: []string{ir.LHSInputParam, ir.RHSInputParam},
				Values: []ir.Type{
					ir.NewTypeVariable("T", ir.NumericConstraint{}),
					ir.NewTypeVariable("T", ir.NumericConstraint{}),
				},
			},
			Outputs: outputs,
		},
	}
}

func createComparisonSymbol(name string) ir.Symbol {
	return createBinaryOpSymbol(
		name,
		ir.NamedTypes{
			Keys:   []string{ir.DefaultOutputParam},
			Values: []ir.Type{ir.U8{}},
		},
	)
}

func createArithmeticSymbol(name string) ir.Symbol {
	return createBinaryOpSymbol(
		name,
		ir.NamedTypes{
			Keys: []string{ir.DefaultOutputParam},
			Values: []ir.Type{
				ir.NewTypeVariable("T", ir.NumericConstraint{}),
			},
		},
	)
}

var SymbolResolver = ir.MapResolver{
	"ge":  createComparisonSymbol("ge"),
	"le":  createComparisonSymbol("le"),
	"lt":  createComparisonSymbol("lt"),
	"eq":  createComparisonSymbol("eq"),
	"and": createArithmeticSymbol("and"),
	"or":  createArithmeticSymbol("or"),
	"add": createArithmeticSymbol("add"),
	"sub": createArithmeticSymbol("sub"),
	"mul": createArithmeticSymbol("mul"),
	"div": createArithmeticSymbol("div"),
	"mod": createArithmeticSymbol("mod"),
}
