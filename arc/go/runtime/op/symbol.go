// Copyright 2026 Synnax Labs, Inc.
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
			Inputs: types.Params{
				{Name: ir.LHSInputParam, Type: types.Variable("T", &constraint)},
				{Name: ir.RHSInputParam, Type: types.Variable("T", &constraint)},
			},
			Outputs: outputs,
		}),
	}
}

func createComparisonSymbol(name string) symbol.Symbol {
	return createBinaryOpSymbol(
		name,
		types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
	)
}

func createArithmeticSymbol(name string) symbol.Symbol {
	constraint := types.NumericConstraint()
	return createBinaryOpSymbol(
		name,
		types.Params{{Name: ir.DefaultOutputParam, Type: types.Variable("T", &constraint)}},
	)
}

func createUnaryOpSymbol(name string, inputType types.Type, outputs types.Params) symbol.Symbol {
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: inputType}},
			Outputs: outputs,
		}),
	}
}

func createNotSymbol(name string) symbol.Symbol {
	return createUnaryOpSymbol(
		name,
		types.U8(),
		types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
	)
}

func createNegateSymbol(name string) symbol.Symbol {
	constraint := types.NumericConstraint()
	return createUnaryOpSymbol(
		name,
		types.Variable("T", &constraint),
		types.Params{{Name: ir.DefaultOutputParam, Type: types.Variable("T", &constraint)}},
	)
}

const (
	geSymbolName  = "ge"
	gtSymbolName  = "gt"
	leSymbolName  = "le"
	ltSymbolName  = "lt"
	eqSymbolName  = "eq"
	neSymbolName  = "ne"
	andSymbolName = "and"
	orSymbolName  = "or"
	addSymbolName = "add"
	subSymbolName = "subtract"
	mulSymbolName = "multiply"
	divSymbolName = "divide"
	modSymbolName = "mod"
	notSymbolName = "not"
	negSymbolName = "neg"
)

var SymbolResolver = symbol.MapResolver{
	geSymbolName:  createComparisonSymbol(geSymbolName),
	gtSymbolName:  createComparisonSymbol(gtSymbolName),
	leSymbolName:  createComparisonSymbol(leSymbolName),
	ltSymbolName:  createComparisonSymbol(ltSymbolName),
	eqSymbolName:  createComparisonSymbol(eqSymbolName),
	neSymbolName:  createComparisonSymbol(neSymbolName),
	andSymbolName: createArithmeticSymbol(andSymbolName),
	orSymbolName:  createArithmeticSymbol(orSymbolName),
	addSymbolName: createArithmeticSymbol(addSymbolName),
	subSymbolName: createArithmeticSymbol(subSymbolName),
	mulSymbolName: createArithmeticSymbol(mulSymbolName),
	divSymbolName: createArithmeticSymbol(divSymbolName),
	modSymbolName: createArithmeticSymbol(modSymbolName),
	notSymbolName: createNotSymbol(notSymbolName),
	negSymbolName: createNegateSymbol(negSymbolName),
}
