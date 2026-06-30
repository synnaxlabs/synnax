// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package flow implements semantic analysis for Arc flow statements connecting channels
// and functions into reactive data pipelines.
package flow

import (
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// freshenKey derives a unique freshening prefix from a call site's AST node so
// each invocation of a polymorphic function gets its own set of type variables.
// Without this, two calls to math.avg with different concrete types would unify
// against a single shared T and conflict.
func freshenKey(node antlr.ParserRuleContext, name string) string {
	tok := node.GetStart()
	return fmt.Sprintf("%s_%d_%d", name, tok.GetLine(), tok.GetColumn())
}

func AnalyzeSingleFunction(ctx context.Context[parser.IFunctionContext]) {
	funcType, name := resolveFunc(ctx, ctx.AST)
	if funcType == nil {
		return
	}
	if rejectWASMFlowNode(ctx, funcType, name) {
		return
	}
	freshType := types.Freshen(funcType.Type, freshenKey(ctx.AST, name))
	args := inputArguments(ctx, ctx.AST.ConfigValues())
	expression.AnalyzeCall(ctx, name, freshType, args, funcType.AnalyzeArguments, ctx.AST, funcType.Trigger.Target)
}

// Analyze validates a flow statement's node chain and routing tables.
func Analyze(ctx context.Context[parser.IFlowStatementContext]) {
	nodes := ctx.AST.AllFlowNode()
	for i, node := range nodes {
		var prevNode parser.IFlowNodeContext
		if i != 0 {
			prevNode = nodes[i-1]
		}
		isLastNode := i == len(nodes)-1
		analyzeNode(context.Child(ctx, node), prevNode, isLastNode)
	}
	for _, routingTable := range ctx.AST.AllRoutingTable() {
		analyzeRoutingTable(context.Child(ctx, routingTable))
	}
}

func analyzeNode(ctx context.Context[parser.IFlowNodeContext], prevNode parser.IFlowNodeContext, isLastNode bool) {
	if id := ctx.AST.Identifier(); id != nil {
		analyzeIdentifier(context.Child(ctx, id), prevNode, isLastNode)
		return
	}
	if fn := ctx.AST.Function(); fn != nil {
		parseFunction(context.Child(ctx, fn), prevNode)
		return
	}
	if expr := ctx.AST.Expression(); expr != nil {
		AnalyzeSingleExpression(context.Child(ctx, expr))
		return
	}
	// NEXT and inline stage/sequence declarations are resolved during sequence
	// analysis, not here. The grammar guarantees flowNode is one of:
	// identifier | function | expression | stageDeclaration | sequenceDeclaration | NEXT.
}

func parseFunction(ctx context.Context[parser.IFunctionContext], prevNode parser.IFlowNodeContext) {
	funcType, name := resolveFunc(ctx, ctx.AST)
	if funcType == nil {
		return
	}
	if rejectWASMFlowNode(ctx, funcType, name) {
		return
	}

	freshType := types.Freshen(funcType.Type, freshenKey(ctx.AST, name))
	args := inputArguments(ctx, ctx.AST.ConfigValues())

	var externallySatisfied []string
	if prevNode != nil && funcType.Trigger.Target != "" {
		externallySatisfied = append(externallySatisfied, funcType.Trigger.Target)
	}
	if isFedByRoutingTable(ctx) {
		// Inputs are bound by the routing table; defer to the routing/edge checks.
		for _, p := range freshType.Inputs {
			externallySatisfied = append(externallySatisfied, p.Name)
		}
	}
	expression.AnalyzeCall(ctx, name, freshType, args, funcType.AnalyzeArguments, ctx.AST, funcType.Trigger.Target, externallySatisfied...)

	if prevNode == nil {
		return
	}
	// TriggerOnly: the upstream is pure activation, so there is no input to
	// type-check against its value (or absence).
	if funcType.Trigger.Target == "" {
		return
	}
	upstreamType, ok := resolveUpstreamType(ctx, prevNode, name)
	if !ok {
		return
	}
	consultTrigger(ctx, funcType, freshType, name, upstreamType, suppliedNames(args, freshType, funcType.Trigger.Target))
}

// resolveUpstreamType returns the value type flowing from prevNode into the func
// node, and whether the trigger consult should proceed.
func resolveUpstreamType(
	ctx context.Context[parser.IFunctionContext],
	prevNode parser.IFlowNodeContext,
	name string,
) (types.Type, bool) {
	if prevIDNode := prevNode.Identifier(); prevIDNode != nil {
		idName := prevIDNode.IDENTIFIER().GetText()
		idSym, err := ctx.Resolve(idName)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, prevIDNode))
			return types.Type{}, false
		}
		if idSym.Kind != symbol.KindChannel {
			ctx.Diagnostics.Add(diagnostics.Errorf(prevIDNode, "%s is not a channel", idName))
			return types.Type{}, false
		}
		if idSym.Type.Kind != types.KindChan {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "%s is not a valid channel", idName))
			return types.Type{}, false
		}
		return idSym.Type.Unwrap(), true
	}
	if prevExpr := prevNode.Expression(); prevExpr != nil {
		return atypes.InferFromExpression(context.Child(ctx, prevExpr)).Unwrap(), true
	}
	if prevFuncNode := prevNode.Function(); prevFuncNode != nil {
		if hasRoutingTableBetween(ctx) {
			return types.Type{}, false
		}
		prevFuncType, prevFuncName := resolveFunc(ctx, prevFuncNode)
		if prevFuncType == nil {
			return types.Type{}, false
		}
		prevFreshType := types.Freshen(prevFuncType.Type, freshenKey(prevFuncNode, prevFuncName))
		prevOutputType := resolveFuncOutput(ctx, prevFreshType, prevFuncName,
			"'"+name+"' expects an input parameter")
		if !prevOutputType.IsValid() {
			return types.Type{}, false
		}
		return prevOutputType, true
	}
	return types.Type{}, false
}

// isFedByRoutingTable reports whether ctx's func node is the first func after a
// routing table, whose entries bind its inputs.
func isFedByRoutingTable(ctx context.Context[parser.IFunctionContext]) bool {
	parent := ctx.AST.GetParent()
	if parent == nil {
		return false
	}
	grandparent := parent.GetParent()
	if grandparent == nil {
		return false
	}
	flowStmt, ok := grandparent.(parser.IFlowStatementContext)
	if !ok {
		return false
	}
	sawTable := false
	for _, child := range flowStmt.GetChildren() {
		if _, ok := child.(parser.IRoutingTableContext); ok {
			sawTable = true
			continue
		}
		flowNode, ok := child.(parser.IFlowNodeContext)
		if !ok {
			continue
		}
		if sawTable {
			if fn := flowNode.Function(); fn != nil {
				return fn == ctx.AST
			}
		}
	}
	return false
}

// hasRoutingTableBetween reports whether the flow statement enclosing the func
// node also contains a routing table.
func hasRoutingTableBetween(ctx context.Context[parser.IFunctionContext]) bool {
	parent := ctx.AST.GetParent()
	if parent == nil {
		return false
	}
	grandparent := parent.GetParent()
	if grandparent == nil {
		return false
	}
	flowStmt, ok := grandparent.(parser.IFlowStatementContext)
	if !ok {
		return false
	}
	return len(flowStmt.AllRoutingTable()) > 0
}

// consultTrigger type-checks an upstream wire's value against funcType's trigger
// param; a param also bound at the call site is a conflict.
func consultTrigger[T antlr.ParserRuleContext](
	ctx context.Context[T],
	funcType *symbol.Symbol,
	callSig types.Type,
	name string,
	upstreamType types.Type,
	suppliedAtCallSite set.Set[string],
) {
	target := funcType.Trigger.Target
	switch {
	case target == "":
		// TriggerOnly: the wire is pure activation; do not type-check its value.
	case suppliedAtCallSite.Contains(target):
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
			"parameter '%s' of func '%s' is bound by both a call-site argument and an upstream wire",
			target, name))
	default:
		targetParam, ok := callSig.Inputs.Get(target)
		if !ok {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
				"func '%s' declares trigger target '%s' but has no such parameter", name, target))
			return
		}
		if err := atypes.Check(ctx.Constraints, targetParam.Type, upstreamType, ctx.AST,
			"upstream connection to func '"+name+"'"); err != nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
				"upstream value type %s does not match func '%s' trigger parameter '%s' type %s",
				upstreamType, name, target, targetParam.Type))
		}
	}
}

func analyzeIdentifier(
	ctx context.Context[parser.IIdentifierContext],
	prevNode parser.IFlowNodeContext,
	isLastNode bool,
) {
	name := ctx.AST.IDENTIFIER().GetText()
	sym, err := ctx.Resolve(name)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}

	isValueVarSink := (sym.Kind == symbol.KindVariable ||
		sym.Kind == symbol.KindStatefulVariable) && sym.Type.Kind != types.KindChan
	if prevNode != nil {
		validTarget := sym.Kind == symbol.KindChannel || sym.Kind == symbol.KindStage ||
			sym.Kind == symbol.KindSequence || sym.Kind == symbol.KindVariable ||
			sym.Kind == symbol.KindStatefulVariable
		if !validTarget {
			d := diagnostics.Errorf(ctx.AST, "%s is not a channel", name)
			if sym.Kind == symbol.KindFunction {
				d = d.WithNote("use " + name + "{} to instantiate the function")
			}
			ctx.Diagnostics.Add(d)
			return
		}
	}

	if isLastNode && prevNode != nil && (sym.Kind == symbol.KindChannel || isValueVarSink) {
		if prevExpr := prevNode.Expression(); prevExpr != nil {
			exprType := atypes.InferFromExpression(context.Child(ctx, prevExpr))
			chanValueType := sym.Type.Unwrap()
			if err = atypes.Check(
				ctx.Constraints,
				exprType,
				chanValueType,
				ctx.AST,
				"expression to channel sink",
			); err != nil {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"expression type %s does not match channel %s value type %s",
					exprType, name, chanValueType,
				))
				return
			}
		}
	}
}

func resolveFunc[T antlr.ParserRuleContext](
	ctx context.Context[T],
	fn parser.IFunctionContext,
) (*symbol.Symbol, string) {
	head, tail := parser.FunctionNameParts(fn)
	name := head
	if tail != "" {
		name = head + "." + tail
	}
	sym, err := ctx.ResolveQualified(head, tail)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nil, name
	}
	if sym.Kind != symbol.KindFunction {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "%s is not a function", name))
		return nil, name
	}
	return sym, name
}

// rejectWASMFlowNode reports an error and returns true when fn is a WASM-only
// function, which is callable inside a func block but cannot be a flow node.
func rejectWASMFlowNode(
	ctx context.Context[parser.IFunctionContext],
	fn *symbol.Symbol,
	name string,
) bool {
	if fn.Exec != symbol.ExecWASM {
		return false
	}
	ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
		"function '%s' cannot be used as a flow statement. Call it inside a func block instead: %s()",
		name, name))
	return true
}

// resolveFuncOutput returns fn's default output type for chained use, emitting
// a diagnostic when fn has named outputs or no return value. downstreamDesc
// is embedded in the no-return message to describe the unsatisfied consumer.
func resolveFuncOutput[T antlr.ParserRuleContext](
	ctx context.Context[T],
	fnType types.Type,
	fnName string,
	downstreamDesc string,
) types.Type {
	if out, ok := fnType.Outputs.Get(ir.DefaultOutputParam); ok {
		return out.Type
	}
	if len(fnType.Outputs) > 0 {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
			"func '%s' has named outputs and requires a routing table",
			fnName))
	} else {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
			"func '%s' has no return value but %s",
			fnName, downstreamDesc))
	}
	return types.Type{}
}

// inputArguments adapts the arguments in a call's brace `{...}` form into the
// unified []symbol.Argument shape, analyzing each value expression as it goes.
func inputArguments[T antlr.ParserRuleContext](
	ctx context.Context[T],
	braceBlock parser.IConfigValuesContext,
) []symbol.Argument {
	if braceBlock == nil {
		return nil
	}
	var args []symbol.Argument
	if namedVals := braceBlock.NamedConfigValues(); namedVals != nil {
		for _, val := range namedVals.AllNamedConfigValue() {
			expr := val.Expression()
			if expr == nil {
				continue
			}
			expression.Analyze(context.Child(ctx, expr))
			args = append(args, symbol.Argument{
				Name: val.IDENTIFIER().GetText(),
				Expr: expr,
				AST:  val,
			})
		}
		return args
	}
	if anonVals := braceBlock.AnonymousConfigValues(); anonVals != nil {
		for i, expr := range anonVals.AllExpression() {
			expression.Analyze(context.Child(ctx, expr))
			args = append(args, symbol.Argument{
				Index: i,
				Expr:  expr,
				AST:   expr,
			})
		}
	}
	return args
}

// suppliedNames returns the set of param names bound by args, resolving positional
// args over the non-trigger params (accounting for the trigger).
func suppliedNames(args []symbol.Argument, fnType types.Type, trigger string) set.Set[string] {
	supplied := set.New[string]()
	positional := fnType.Inputs.Positional(trigger)
	for _, arg := range args {
		if arg.Name != "" {
			supplied.Add(arg.Name)
		} else if arg.Index < len(positional) {
			supplied.Add(positional[arg.Index].Name)
		}
	}
	return supplied
}

func analyzeRoutingTable(ctx context.Context[parser.IRoutingTableContext]) {
	flowStmt, ok := ctx.AST.GetParent().(parser.IFlowStatementContext)
	if !ok {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"routing table must be part of a flow statement",
		))
		return
	}

	tables := flowStmt.AllRoutingTable()

	if len(tables) != 1 || tables[0] != ctx.AST {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"unexpected routing table configuration",
		))
		return
	}

	var nodesBefore, nodesAfter []parser.IFlowNodeContext
	foundRoutingTable := false

	for _, child := range flowStmt.GetChildren() {
		if rt, ok := child.(parser.IRoutingTableContext); ok && rt == ctx.AST {
			foundRoutingTable = true
			continue
		}
		if flowNode, ok := child.(parser.IFlowNodeContext); ok {
			if foundRoutingTable {
				nodesAfter = append(nodesAfter, flowNode)
			} else {
				nodesBefore = append(nodesBefore, flowNode)
			}
		}
	}

	if len(nodesBefore) == 0 && len(nodesAfter) > 0 {
		analyzeInputRoutingTable(ctx, nodesAfter)
	} else if len(nodesBefore) > 0 {
		analyzeOutputRoutingTable(ctx, nodesBefore, nodesAfter)
	} else {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"routing table must have associated flow nodes",
		))
	}
}

func analyzeOutputRoutingTable(
	ctx context.Context[parser.IRoutingTableContext],
	nodesBefore []parser.IFlowNodeContext,
	nodesAfter []parser.IFlowNodeContext,
) {
	var PrevFunc parser.IFunctionContext
	for _, n := range slices.Backward(nodesBefore) {
		if fn := n.Function(); fn != nil {
			PrevFunc = fn
			break
		}
	}

	if PrevFunc == nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"output routing table must follow a func invocation",
		))
		return
	}

	fnType, fnName := resolveFunc(ctx, PrevFunc)
	if fnType == nil {
		return
	}

	_, hasDefaultOutput := fnType.Type.Outputs.Get(ir.DefaultOutputParam)
	hasNamedOutputs := len(fnType.Type.Outputs) > 1 || (len(fnType.Type.Outputs) == 1 && !hasDefaultOutput)
	if !hasNamedOutputs {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"func '%s' does not have named outputs, cannot use routing table",
			fnName,
		))
		return
	}

	var (
		nextFunc     parser.IFunctionContext
		nextFuncType types.Type
	)
	for _, node := range nodesAfter {
		if fn := node.Function(); fn != nil {
			nextFunc = fn
			head, tail := parser.FunctionNameParts(nextFunc)
			nextFuncScope, err := ctx.Scope.Resolve(ctx, head)
			if err == nil && tail != "" {
				nextFuncScope, err = nextFuncScope.Resolve(ctx, tail)
			}
			if err == nil && nextFuncScope.Kind == symbol.KindFunction {
				nextFuncType = nextFuncScope.Type
			}
			break
		}
	}

	// Analyze each routing entry
	for _, entry := range ctx.AST.AllRoutingEntry() {
		outputName := entry.IDENTIFIER(0).GetText()

		outputType, exists := fnType.Type.Outputs.Get(outputName)
		if !exists {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				entry,
				"func '%s' does not have output '%s'",
				fnName,
				outputName,
			))
			continue
		}

		var targetParamName string
		if len(entry.AllIDENTIFIER()) > 1 {
			targetParamName = entry.IDENTIFIER(1).GetText()

			if nextFunc == nil {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					entry,
					"parameter mapping requires a func after the routing table",
				))
				continue
			}

			if _, exists := nextFuncType.Inputs.Get(targetParamName); !exists {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					entry,
					"func '%s' does not have parameter '%s'",
					parser.FunctionName(nextFunc),
					targetParamName,
				))
				continue
			}
		}

		// First node's source is the select-output type; subsequent nodes
		// chain from the previous node's output.
		flowNodes := entry.AllFlowNode()
		nodeSourceType := outputType.Type
		for i, flowNode := range flowNodes {
			isLastNode := i == len(flowNodes)-1
			var targetParam *string
			if isLastNode && targetParamName != "" {
				targetParam = &targetParamName
			}
			analyzeRoutingTargetWithParam(
				context.Child(ctx, flowNode),
				nodeSourceType,
				nextFuncType,
				targetParam,
			)
			if !isLastNode {
				nodeSourceType = inferFlowNodeOutputType(context.Child(ctx, flowNode))
			}
		}
	}
}

func analyzeInputRoutingTable(
	ctx context.Context[parser.IRoutingTableContext],
	nodes []parser.IFlowNodeContext,
) {
	var nextFunc parser.IFunctionContext
	for i := range nodes {
		if fn := nodes[i].Function(); fn != nil {
			nextFunc = fn
			break
		}
	}

	if nextFunc == nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "input routing table must precede a func invocation"))
		return
	}

	fnType, fnName := resolveFunc(ctx, nextFunc)
	if fnType == nil {
		return
	}

	for _, entry := range ctx.AST.AllRoutingEntry() {
		flowNodes := entry.AllFlowNode()
		if len(flowNodes) == 0 {
			ctx.Diagnostics.Add(diagnostics.Errorf(entry, "routing entry must have at least one target"))
			continue
		}

		lastNode := flowNodes[len(flowNodes)-1]
		if lastNode.Identifier() == nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				lastNode,
				"last element in input routing entry must be a parameter name (identifier)",
			))
			continue
		}

		paramName := lastNode.Identifier().IDENTIFIER().GetText()

		paramType, exists := fnType.Type.Inputs.Get(paramName)
		if !exists {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				lastNode,
				"func '%s' does not have parameter '%s'",
				fnName,
				paramName,
			))
			continue
		}

		// Analyze the flow chain: source (entry.IDENTIFIER) -> processing nodes -> parameter
		// For type checking, we need to verify the output type of the chain matches paramType
		// TODO: Implement full type checking for the flow chain
		// See https://linear.app/synnax/issue/SY-3176/implement-full-type-checking-for-arc-flow-statements
		_ = paramType

		for i := 0; i < len(flowNodes)-1; i++ {
			analyzeNode(context.Child(ctx, flowNodes[i]), nil, false)
		}
	}
}

func analyzeRoutingTargetWithParam(
	ctx context.Context[parser.IFlowNodeContext],
	sourceType types.Type,
	nextFuncType types.Type,
	targetParam *string,
) {
	if fn := ctx.AST.Function(); fn != nil {
		fnType, fnName := resolveFunc(ctx, fn)
		if fnType == nil {
			return
		}

		args := inputArguments(ctx, fn.ConfigValues())
		var externallySatisfied []string
		if fnType.Trigger.Target != "" {
			externallySatisfied = append(externallySatisfied, fnType.Trigger.Target)
		}
		expression.AnalyzeCall(ctx, fnName, fnType.Type, args, fnType.AnalyzeArguments, fn, fnType.Trigger.Target, externallySatisfied...)

		if targetParam != nil {
			var outputType types.Type
			if outType, ok := fnType.Type.Outputs.Get(ir.DefaultOutputParam); ok {
				outputType = outType.Type
			} else if len(fnType.Type.Outputs) > 0 {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"func '%s' has named outputs and requires explicit output selection",
					fnName,
				))
				return
			}

			if param, exists := nextFuncType.Inputs.Get(*targetParam); exists {
				if err := atypes.Check(ctx.Constraints, outputType, param.Type, ctx.AST,
					"routing table parameter mapping"); err != nil {
					ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
						"type mismatch: func %s output type %s does not match target parameter %s type %s",
						fnName,
						outputType,
						*targetParam,
						param,
					))
					return
				}
			}
		} else {
			consultTrigger(ctx, fnType, fnType.Type, fnName, sourceType, suppliedNames(args, fnType.Type, fnType.Trigger.Target))
		}
	} else if idNode := ctx.AST.Identifier(); idNode != nil {
		idName := idNode.IDENTIFIER().GetText()
		idSym, err := ctx.Resolve(idName)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return
		}

		// Allow channels, sequences, and stages as routing targets
		if idSym.Kind != symbol.KindChannel && idSym.Kind != symbol.KindSequence && idSym.Kind != symbol.KindStage {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "%s is not a channel, sequence, or stage", idName))
			return
		}

		// Only do type checking for channels (sequences/stages accept any input for activation)
		if idSym.Kind == symbol.KindChannel {
			valueType := idSym.Type.Unwrap()
			if err = atypes.Check(
				ctx.Constraints,
				sourceType,
				valueType,
				ctx.AST,
				"routing table output to channel",
			); err != nil {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"type mismatch: output type %s does not match channel %s value type %s",
					sourceType,
					idName,
					valueType,
				))
				return
			}
		}
	} else if expr := ctx.AST.Expression(); expr != nil {
		AnalyzeSingleExpression(context.Child(ctx, expr))
	}
}

// inferFlowNodeOutputType returns the type a flow node emits to the next node
// in a routing-entry chain.
func inferFlowNodeOutputType(ctx context.Context[parser.IFlowNodeContext]) types.Type {
	if expr := ctx.AST.Expression(); expr != nil {
		return atypes.InferFromExpression(context.Child(ctx, expr))
	}
	if fn := ctx.AST.Function(); fn != nil {
		fnName := parser.FunctionName(fn)
		sym, err := ctx.Resolve(fnName)
		if err != nil || sym.Kind != symbol.KindFunction {
			return types.Type{}
		}
		return resolveFuncOutput(ctx, sym.Type, fnName,
			"the next node in the chain expects an input")
	}
	if idNode := ctx.AST.Identifier(); idNode != nil {
		sym, err := ctx.Resolve(idNode.IDENTIFIER().GetText())
		if err != nil || sym.Kind != symbol.KindChannel {
			return types.Type{}
		}
		return sym.Type.Unwrap()
	}
	return types.Type{}
}
