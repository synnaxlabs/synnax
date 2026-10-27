// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/codes"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// AnalyzeCall validates args against fnType.Inputs and runs the hook, accounting for
// the trigger in positional binding; externallySatisfied names aren't reported missing.
func AnalyzeCall[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	fnName string,
	fnType types.Type,
	args []symbol.Argument,
	hook symbol.ArgumentsHook,
	site antlr.ParserRuleContext,
	trigger string,
	externallySatisfied ...string,
) {
	signature := ir.FormatFunctionSignature(fnName, fnType)
	supplied := set.New[string]()
	for _, b := range bindArguments(ctx, fnName, fnType, args, signature, trigger) {
		supplied.Add(b.param.Name)
		checkArgType(ctx, fnName, b.param, b.arg, signature)
	}
	satisfied := set.New(externallySatisfied...)
	for _, param := range fnType.Inputs {
		if param.Value == nil && !supplied.Contains(param.Name) &&
			!satisfied.Contains(param.Name) {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				site,
				"missing required argument for parameter '%s' of func '%s'",
				param.Name,
				fnName,
			).
				WithCode(codes.FuncArgCount).WithNote("signature: " + signature))
		}
	}
	if hook != nil {
		hook(ctx.Diagnostics, args)
	}
}

// boundArgument pairs a call argument with the param it binds.
type boundArgument struct {
	arg   symbol.Argument
	param types.Param
}

// bindArguments matches each arg to its param by name or position and reports unknown
// names, out-of-range positions, and duplicates. Type checks are the caller's.
func bindArguments[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	fnName string,
	fnType types.Type,
	args []symbol.Argument,
	signature string,
	trigger string,
) []boundArgument {
	supplied := set.New[string]()
	var bound []boundArgument
	for _, arg := range args {
		param, ok := matchParam(ctx, fnName, fnType, arg, signature, trigger)
		if !ok {
			continue
		}
		if supplied.Contains(param.Name) {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				arg.AST,
				"duplicate argument for parameter '%s' of func '%s'",
				param.Name,
				fnName,
			).
				WithNote("signature: " + signature))
			continue
		}
		supplied.Add(param.Name)
		bound = append(bound, boundArgument{arg: arg, param: param})
	}
	return bound
}

// matchParam resolves arg to its param by name or position, accounting for the
// trigger when binding by position. Reports an unknown name or out-of-range arg.
func matchParam[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	fnName string,
	fnType types.Type,
	arg symbol.Argument,
	signature string,
	trigger string,
) (types.Param, bool) {
	if arg.Name != "" {
		param, ok := fnType.Inputs.Get(arg.Name)
		if !ok {
			ctx.Diagnostics.Add(diagnostics.Errorf(arg.AST,
				"unknown parameter '%s' for func '%s'", arg.Name, fnName).
				WithNote("signature: " + signature))
			return types.Param{}, false
		}
		return param, true
	}
	positional := fnType.Inputs.Positional(trigger)
	if arg.Index >= len(positional) {
		if arg.Index == len(positional) {
			ctx.Diagnostics.Add(diagnostics.Errorf(arg.AST,
				"too many arguments for func '%s': expected at most %d",
				fnName, len(positional)).
				WithCode(codes.FuncArgCount).WithNote("signature: " + signature))
		}
		return types.Param{}, false
	}
	return positional[arg.Index], true
}

// checkArgType checks arg's type against param, dereferencing a channel argument to
// its value type unless param is itself channel-typed.
func checkArgType[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	fnName string,
	param types.Param,
	arg symbol.Argument,
	signature string,
) {
	argType := atypes.InferFromExpression(ctx.Child(arg.Expr))
	if param.Type.Kind != types.KindChan {
		argType = argType.UnwrapChan()
	}
	if err := atypes.Check(ctx.Constraints, param.Type, argType, arg.AST,
		fmt.Sprintf("argument '%s' of '%s'", param.Name, fnName)); err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, arg.AST).
			WithCode(codes.FuncArgType).WithNote("signature: " + signature))
	}
}
