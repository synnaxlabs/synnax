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

func binaryOp(name string, outputs types.Params, body doc.Doc) *symbol.Symbol {
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
		Trigger: symbol.TriggerInput(ir.LHSInputParam),
		Doc:     body,
	}
}

func comparison(name string, body doc.Doc) *symbol.Symbol {
	return binaryOp(name, types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}}, body)
}

func logical(name string, body doc.Doc) *symbol.Symbol {
	constraint := types.NumericConstraint()
	return binaryOp(
		name,
		types.Params{{Name: ir.DefaultOutputParam, Type: types.Variable("T", &constraint)}},
		body,
	)
}

func not(name string, body doc.Doc) *symbol.Symbol {
	return &symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: types.U8()}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
		}),
		Trigger: symbol.TriggerInput(ir.DefaultInputParam),
		Doc:     body,
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

var (
	geDoc  = doc.New(doc.Paragraph("Greater-than-or-equal comparison. Returns 1 if `a >= b`, 0 otherwise."), doc.Divider(), doc.Code("arc", "ge(a, b)  // equivalent to: a >= b"))
	gtDoc  = doc.New(doc.Paragraph("Greater-than comparison. Returns 1 if `a > b`, 0 otherwise."), doc.Divider(), doc.Code("arc", "gt(a, b)  // equivalent to: a > b"))
	leDoc  = doc.New(doc.Paragraph("Less-than-or-equal comparison. Returns 1 if `a <= b`, 0 otherwise."), doc.Divider(), doc.Code("arc", "le(a, b)  // equivalent to: a <= b"))
	ltDoc  = doc.New(doc.Paragraph("Less-than comparison. Returns 1 if `a < b`, 0 otherwise."), doc.Divider(), doc.Code("arc", "lt(a, b)  // equivalent to: a < b"))
	eqDoc  = doc.New(doc.Paragraph("Equality comparison. Returns 1 if `a == b`, 0 otherwise."), doc.Divider(), doc.Code("arc", "eq(a, b)  // equivalent to: a == b"))
	neDoc  = doc.New(doc.Paragraph("Inequality comparison. Returns 1 if `a != b`, 0 otherwise."), doc.Divider(), doc.Code("arc", "ne(a, b)  // equivalent to: a != b"))
	andDoc = doc.New(doc.Paragraph("Logical AND. Returns a nonzero value if both inputs are nonzero, 0 otherwise."), doc.Divider(), doc.Code("arc", "and(a, b)  // equivalent to: a && b"))
	orDoc  = doc.New(doc.Paragraph("Logical OR. Returns a nonzero value if either input is nonzero, 0 otherwise."), doc.Divider(), doc.Code("arc", "or(a, b)  // equivalent to: a || b"))
	notDoc = doc.New(doc.Paragraph("Logical NOT. Returns 1 if the input is 0, 0 otherwise."), doc.Divider(), doc.Code("arc", "not(a)  // equivalent to: !a"))
)

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes. Operators are root-level (no module) — they install directly
// at the root scope so lowering passes can emit calls without imports.
func NewSymbols() []*symbol.Symbol {
	return []*symbol.Symbol{
		comparison(geSymbolName, geDoc),
		comparison(gtSymbolName, gtDoc),
		comparison(leSymbolName, leDoc),
		comparison(ltSymbolName, ltDoc),
		comparison(eqSymbolName, eqDoc),
		comparison(neSymbolName, neDoc),
		logical(andSymbolName, andDoc),
		logical(orSymbolName, orDoc),
		not(notSymbolName, notDoc),
	}
}
