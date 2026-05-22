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
	"github.com/synnaxlabs/x/lsp/doc"
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
	comparison(geSymbolName).Document(
		doc.Paragraph("Greater-than-or-equal comparison. Returns 1 if `a >= b`, 0 otherwise."),
		doc.Divider(),
		symbol.Arc("ge(a, b)  // equivalent to: a >= b"),
	),
	comparison(gtSymbolName).Document(
		doc.Paragraph("Greater-than comparison. Returns 1 if `a > b`, 0 otherwise."),
		doc.Divider(),
		symbol.Arc("gt(a, b)  // equivalent to: a > b"),
	),
	comparison(leSymbolName).Document(
		doc.Paragraph("Less-than-or-equal comparison. Returns 1 if `a <= b`, 0 otherwise."),
		doc.Divider(),
		symbol.Arc("le(a, b)  // equivalent to: a <= b"),
	),
	comparison(ltSymbolName).Document(
		doc.Paragraph("Less-than comparison. Returns 1 if `a < b`, 0 otherwise."),
		doc.Divider(),
		symbol.Arc("lt(a, b)  // equivalent to: a < b"),
	),
	comparison(eqSymbolName).Document(
		doc.Paragraph("Equality comparison. Returns 1 if `a == b`, 0 otherwise."),
		doc.Divider(),
		symbol.Arc("eq(a, b)  // equivalent to: a == b"),
	),
	comparison(neSymbolName).Document(
		doc.Paragraph("Inequality comparison. Returns 1 if `a != b`, 0 otherwise."),
		doc.Divider(),
		symbol.Arc("ne(a, b)  // equivalent to: a != b"),
	),
	logical(andSymbolName).Document(
		doc.Paragraph("Logical AND. Returns a nonzero value if both inputs are nonzero, 0 otherwise."),
		doc.Divider(),
		symbol.Arc("and(a, b)  // equivalent to: a && b"),
	),
	logical(orSymbolName).Document(
		doc.Paragraph("Logical OR. Returns a nonzero value if either input is nonzero, 0 otherwise."),
		doc.Divider(),
		symbol.Arc("or(a, b)  // equivalent to: a || b"),
	),
	not(notSymbolName).Document(
		doc.Paragraph("Logical NOT. Returns 1 if the input is 0, 0 otherwise."),
		doc.Divider(),
		symbol.Arc("not(a)  // equivalent to: !a"),
	),
}
