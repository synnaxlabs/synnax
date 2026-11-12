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
	"github.com/synnaxlabs/arc/parser"
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
			ctx.Diagnostics.Add(codes.FuncArgMissing.At(site, param.Name, fnName).
				WithNote("signature: " + signature))
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
			ctx.Diagnostics.Add(codes.FuncArgExtra.At(arg.AST, fnName, len(positional)).
				WithNote("signature: " + signature))
		}
		return types.Param{}, false
	}
	return positional[arg.Index], true
}

// parameterType returns the type node of the parameter name of fnName in this module.
func parameterType[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	fnName, name string,
) antlr.ParserRuleContext {
	fn, err := ctx.Scope.Resolve(ctx, fnName)
	if err != nil || fn.Root() != ctx.Scope.Root() {
		return nil
	}
	decl, ok := fn.AST.(parser.IFunctionDeclarationContext)
	if !ok {
		return nil
	}
	if inputs := decl.InputList(); inputs != nil {
		for _, input := range inputs.AllInput() {
			if input.IDENTIFIER().GetText() == name {
				return input.Type_()
			}
		}
	}
	if triggers := decl.TriggerList(); triggers != nil {
		for _, trigger := range triggers.AllTrigger() {
			if trigger.IDENTIFIER().GetText() == name {
				return trigger.Type_()
			}
		}
	}
	return nil
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
	argType := computedArgType(ctx, arg.Expr)
	if !argType.IsValid() {
		argType = atypes.InferFromExpression(ctx.Child(arg.Expr))
	}
	if param.Type.Kind != types.KindChan {
		argType = argType.UnwrapChan()
	}
	paramType, check := param.Type, atypes.Check
	// Lowering warns for a flow variable, which can be a constant.
	if paramType.Unwrap().Kind != types.KindVariable {
		converts, ok := CheckUnits(ctx, Write{
			AST:         arg.AST,
			Expr:        arg.Expr,
			Source:      argType,
			Target:      paramType,
			Subject:     fmt.Sprintf("parameter '%s' of '%s'", param.Name, fnName),
			Declaration: parameterType(ctx, fnName, param.Name),
			Lowered:     readsFlowVariable(ctx, arg.Expr),
		})
		if !ok {
			return
		}
		if converts {
			check = atypes.CheckConverted
		}
	}
	if err := check(ctx.Constraints, paramType, argType, arg.AST,
		fmt.Sprintf("argument '%s' of '%s'", param.Name, fnName)); err != nil {
		d := codes.FuncArgType.Wrap(err, arg.AST)
		if cast, ok := atypes.CastOf(err); ok {
			d = d.WithFix(arg.Expr, cast+"("+parser.GetExpressionText(arg.Expr)+")")
		}
		ctx.Diagnostics.Add(d.WithNote("signature: " + signature))
	}
}

// WithUnit returns t with unit on its value type.
func WithUnit(t types.Type, unit *types.Unit) types.Type {
	if (t.Kind == types.KindChan || t.Kind == types.KindSeries) && t.Elem != nil {
		elem := WithUnit(*t.Elem, unit)
		t.Elem = &elem
		return t
	}
	t.Unit = unit
	return t
}

// readsFlowVariable reports whether expr names a variable outside a function body.
func readsFlowVariable[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	expr parser.IExpressionContext,
) bool {
	if _, err := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction); err == nil {
		return false
	}
	head, tail := parser.ExpressionNameParts(expr)
	if head == "" {
		return false
	}
	sym, err := ctx.Scope.ResolveMember(ctx, head, tail)
	return err == nil && sym.IsValueVariable()
}

// TakeUnit gives the node registered for expr the unit of the target of its value.
func TakeUnit[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	expr parser.IExpressionContext,
	unit *types.Unit,
) {
	node, err := ctx.Scope.Root().GetChildByParserRule(expr)
	if err != nil || len(node.Type.Outputs) == 0 ||
		!types.UnitsConvertible(node.Type.Outputs[0].Type.Unit, unit) {
		return
	}
	node.Type.Outputs[0].Type.Unit = unit
	if node.Kind == symbol.KindConstant {
		node.Type.Inputs[0].Type.Unit = unit
	}
}

// computedArgType returns the output type of the node registered for a computed
// argument, or an invalid type if expr has none. Inferring expr again would replace the
// node body's types.
func computedArgType[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	expr parser.IExpressionContext,
) types.Type {
	synth, err := ctx.Scope.Root().GetChildByParserRule(expr)
	if err != nil {
		return types.Type{}
	}
	out, _ := synth.Type.Outputs.Get(ir.DefaultOutputParam)
	return out.Type
}
