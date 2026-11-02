// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/diagnostics"
)

// Argument is one call argument: a value expression bound to a param by Name,
// or by position (Index) when Name is empty.
type Argument struct {
	// Index is the argument's position in the call's argument list.
	Index int
	// Name is the param this argument binds to, or empty if positional.
	Name string
	// Expr is the argument's value expression.
	Expr parser.IExpressionContext
	// AST is the argument's parser node, for diagnostic source locations.
	AST antlr.ParserRuleContext
}

// ArgumentsHook runs optional symbol-specific argument validation, reporting any
// findings to the analyzer's diagnostics sink.
type ArgumentsHook func(diags *diagnostics.Diagnostics, args []Argument)

// CallArguments returns the positional arguments of a call in a func body.
func CallArguments(call parser.IFunctionCallSuffixContext) []Argument {
	argList := call.ArgumentList()
	if argList == nil {
		return nil
	}
	return positional(argList.AllExpression())
}

// FunctionArguments returns the named or positional arguments of a flow function node.
func FunctionArguments(fn parser.IFunctionContext) []Argument {
	values := fn.InputValues()
	if values == nil {
		return nil
	}
	if named := values.NamedInputValues(); named != nil {
		var args []Argument
		for _, val := range named.AllNamedInputValue() {
			expr := val.Expression()
			if expr == nil {
				continue
			}
			args = append(args, Argument{
				Name: val.IDENTIFIER().GetText(),
				Expr: expr,
				AST:  val,
			})
		}
		return args
	}
	if anon := values.AnonymousInputValues(); anon != nil {
		return positional(anon.AllExpression())
	}
	return nil
}

func positional(exprs []parser.IExpressionContext) []Argument {
	args := make([]Argument, len(exprs))
	for i, expr := range exprs {
		args[i] = Argument{Index: i, Expr: expr, AST: expr}
	}
	return args
}
