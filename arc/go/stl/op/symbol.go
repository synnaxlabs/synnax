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

func binaryOp(name string, outputs types.Params) *symbol.Symbol {
	constraint := types.NumericConstraint()
	return &symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: ir.LHSInputParam, Type: types.Variable("T", &constraint)},
				{Name: ir.RHSInputParam, Type: types.Variable("T", &constraint)},
			},
			Outputs: outputs,
		}),
	}
}

func comparison(name string) *symbol.Symbol {
	return binaryOp(name, types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}})
}

func logical(name string) *symbol.Symbol {
	constraint := types.NumericConstraint()
	return binaryOp(
		name,
		types.Params{{Name: ir.DefaultOutputParam, Type: types.Variable("T", &constraint)}},
	)
}

func not(name string) *symbol.Symbol {
	return &symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: types.U8()}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
		}),
	}
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
	notSymbolName = "not"
)

// Symbols are the symbols this package contributes to a program's ambient
// prelude. Operators are root-level (no module) — they install directly at
// the root scope so lowering passes can emit calls without imports.
var Symbols = []*symbol.Symbol{
	comparison(geSymbolName),
	comparison(gtSymbolName),
	comparison(leSymbolName),
	comparison(ltSymbolName),
	comparison(eqSymbolName),
	comparison(neSymbolName),
	logical(andSymbolName),
	logical(orSymbolName),
	not(notSymbolName),
}
