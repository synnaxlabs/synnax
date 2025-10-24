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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

func createBinaryOpSymbol(name string, outputs types.Params) symbol.Symbol {
	constraint := types.NumericConstraint()
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: &types.Params{
				Keys: []string{ir.LHSInputParam, ir.RHSInputParam},
				Values: []types.Type{
					types.NewTypeVariable("T", &constraint),
					types.NewTypeVariable("T", &constraint),
				},
			},
			Outputs: &outputs,
		}),
	}
}

func createComparisonSymbol(name string) symbol.Symbol {
	return createBinaryOpSymbol(
		name,
		types.Params{
			Keys:   []string{ir.DefaultOutputParam},
			Values: []types.Type{types.U8()},
		},
	)
}

func createArithmeticSymbol(name string) symbol.Symbol {
	constraint := types.NumericConstraint()
	return createBinaryOpSymbol(
		name,
		types.Params{
			Keys: []string{ir.DefaultOutputParam},
			Values: []types.Type{
				types.NewTypeVariable("T", &constraint),
			},
		},
	)
}

var SymbolResolver = symbol.MapResolver{
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
