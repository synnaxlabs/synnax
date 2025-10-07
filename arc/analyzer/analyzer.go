// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/diagnostics"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/analyzer/statement"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/maps"
)

type Diagnostics = diagnostics.Diagnostics

func AnalyzeProgram(ctx acontext.Context[parser.IProgramContext]) bool {
	if !collectDeclarations(ctx) {
		return false
	}
	if !analyzeDeclarations(ctx) {
		return false
	}
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}
		if !applyTypeSubstitutions(ctx) {
			return false
		}
	}
	return true
}

func collectDeclarations(ctx acontext.Context[parser.IProgramContext]) bool {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if fn := item.FunctionDeclaration(); fn != nil {
			name := fn.IDENTIFIER().GetText()
			if _, err := ctx.Scope.Add(ctx, ir.Symbol{Name: name, Kind: ir.KindFunction, Type: ir.Function{Key: name}, ParserRule: fn}); err != nil {
				ctx.Diagnostics.AddError(err, fn)
				return false
			}
		} else if stage := item.StageDeclaration(); stage != nil {
			name := stage.IDENTIFIER().GetText()
			if _, err := ctx.Scope.Add(ctx, ir.Symbol{Name: name, Kind: ir.KindStage, Type: ir.Stage{Key: name}, ParserRule: stage}); err != nil {
				ctx.Diagnostics.AddError(err, stage)
				return false
			}
		}
	}
	return true
}

func analyzeDeclarations(ctx acontext.Context[parser.IProgramContext]) bool {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
			if !analyzeFunctionDeclaration(acontext.Child(ctx, funcDecl)) {
				return false
			}
		} else if stageDecl := item.StageDeclaration(); stageDecl != nil {
			if !analyzeStageDeclaration(acontext.Child(ctx, stageDecl)) {
				return false
			}
		} else if flowStmt := item.FlowStatement(); flowStmt != nil {
			if !flow.Analyze(acontext.Child(ctx, flowStmt)) {
				return false
			}
		}
	}
	return true
}

func AnalyzeStatement(ctx acontext.Context[parser.IStatementContext]) bool {
	return statement.Analyze(ctx)
}

func AnalyzeBlock(ctx acontext.Context[parser.IBlockContext]) bool {
	return statement.AnalyzeBlock(ctx)
}

func analyzeFunctionDeclaration(ctx acontext.Context[parser.IFunctionDeclarationContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	fnScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	fnType := fnScope.Type.(ir.Function)
	if !analyzeParams(acontext.Child(ctx, ctx.AST.ParameterList()).WithScope(fnScope), &fnType.Params) {
		return false
	}
	if !analyzeReturnTypeFn(ctx, ctx.AST.ReturnType(), fnScope, &fnType.Outputs) {
		return false
	}
	if block := ctx.AST.Block(); block != nil {
		fnType.Body = ir.Body{AST: block}
	}
	fnScope.Type = fnType
	if block := ctx.AST.Block(); block != nil {
		if !statement.AnalyzeBlock(acontext.Child(ctx, block).WithScope(fnScope)) {
			return false
		}
		if outputType, hasOutput := fnType.Outputs.Get("output"); hasOutput && outputType != nil && !blockAlwaysReturns(block) {
			ctx.Diagnostics.AddError(errors.Newf("function '%s' must return a value of type %s on all paths", name, outputType), ctx.AST)
			return false
		}
	}
	return true
}

func analyzeReturnTypeFn(
	ctx acontext.Context[parser.IFunctionDeclarationContext],
	retType parser.IReturnTypeContext,
	scope *ir.Scope,
	outputs *maps.Ordered[string, ir.Type],
) bool {
	return parseReturnType(ctx, retType, scope, outputs)
}

func analyzeReturnTypeStage(
	ctx acontext.Context[parser.IStageDeclarationContext],
	retType parser.IReturnTypeContext,
	scope *ir.Scope,
	outputs *maps.Ordered[string, ir.Type],
) bool {
	return parseReturnType(ctx, retType, scope, outputs)
}

func parseReturnType[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	retType parser.IReturnTypeContext,
	scope *ir.Scope,
	outputs *maps.Ordered[string, ir.Type],
) bool {
	if retType == nil {
		return true
	}
	if typeCtx := retType.Type_(); typeCtx != nil {
		outputType, _ := atypes.InferFromTypeContext(typeCtx)
		if !outputs.Put("output", outputType) {
			ctx.Diagnostics.AddError(errors.New("failed to add output"), retType)
			return false
		}
		return true
	}
	if multiOutputBlock := retType.MultiOutputBlock(); multiOutputBlock != nil {
		for _, namedOutput := range multiOutputBlock.AllNamedOutput() {
			outputName := namedOutput.IDENTIFIER().GetText()
			var outputType ir.Type
			if typeCtx := namedOutput.Type_(); typeCtx != nil {
				outputType, _ = atypes.InferFromTypeContext(typeCtx)
			}
			if !outputs.Put(outputName, outputType) {
				ctx.Diagnostics.AddError(errors.Newf("duplicate output %s", outputName), namedOutput)
				return false
			}
			if _, err := scope.Add(ctx, ir.Symbol{Name: outputName, Kind: ir.KindOutput, Type: outputType, ParserRule: namedOutput}); err != nil {
				ctx.Diagnostics.AddError(err, namedOutput)
				return false
			}
		}
	}
	return true
}

func checkOutputAssignedInBlock(block parser.IBlockContext, outputName string) bool {
	if block == nil {
		return false
	}
	for _, stmt := range block.AllStatement() {
		if assignment := stmt.Assignment(); assignment != nil && assignment.IDENTIFIER().GetText() == outputName {
			return true
		}
		if ifStmt := stmt.IfStatement(); ifStmt != nil && checkOutputAssignedInIfStmt(ifStmt, outputName) {
			return true
		}
	}
	return false
}

func checkOutputAssignedInIfStmt(ifStmt parser.IIfStatementContext, outputName string) bool {
	if checkOutputAssignedInBlock(ifStmt.Block(), outputName) {
		return true
	}
	for _, elseIf := range ifStmt.AllElseIfClause() {
		if checkOutputAssignedInBlock(elseIf.Block(), outputName) {
			return true
		}
	}
	if elseClause := ifStmt.ElseClause(); elseClause != nil {
		return checkOutputAssignedInBlock(elseClause.Block(), outputName)
	}
	return false
}

func analyzeParams(
	ctx acontext.Context[parser.IParameterListContext],
	paramTypes *maps.Ordered[string, ir.Type],
) bool {
	if ctx.AST == nil {
		return true
	}
	for _, param := range ctx.AST.AllParameter() {
		var paramType ir.Type
		if typeCtx := param.Type_(); typeCtx != nil {
			paramType, _ = atypes.InferFromTypeContext(typeCtx)
		}
		paramName := param.IDENTIFIER().GetText()
		if !paramTypes.Put(paramName, paramType) {
			ctx.Diagnostics.AddError(errors.Newf("duplicate parameter %s", paramName), param)
			return false
		}
		if _, err := ctx.Scope.Add(ctx, ir.Symbol{Name: paramName, Kind: ir.KindParam, Type: paramType, ParserRule: param}); err != nil {
			ctx.Diagnostics.AddError(err, param)
			return false
		}
	}
	return true
}

func blockAlwaysReturns(block parser.IBlockContext) bool {
	if block == nil {
		return false
	}
	statements := block.AllStatement()
	for i := len(statements) - 1; i >= 0; i-- {
		if statements[i].ReturnStatement() != nil {
			return true
		}
		if ifStmt := statements[i].IfStatement(); ifStmt != nil && ifStmtAlwaysReturns(ifStmt) {
			return true
		}
	}
	return false
}

func ifStmtAlwaysReturns(ifStmt parser.IIfStatementContext) bool {
	if ifStmt.ElseClause() == nil || !blockAlwaysReturns(ifStmt.Block()) {
		return false
	}
	for _, elseIf := range ifStmt.AllElseIfClause() {
		if !blockAlwaysReturns(elseIf.Block()) {
			return false
		}
	}
	return blockAlwaysReturns(ifStmt.ElseClause().Block())
}

func applyTypeSubstitutions(ctx acontext.Context[parser.IProgramContext]) bool {
	return applySubstitutionsToScope(ctx, ctx.Scope)
}

func applySubstitutionsToScope(ctx acontext.Context[parser.IProgramContext], scope *ir.Scope) bool {
	if scope.Type != nil {
		scope.Type = ctx.Constraints.ApplySubstitutions(scope.Type)
	}
	for _, child := range scope.Children {
		if !applySubstitutionsToScope(ctx, child) {
			return false
		}
	}
	return true
}

func analyzeStageDeclaration(ctx acontext.Context[parser.IStageDeclarationContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	stageScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	stageType := stageScope.Type.(ir.Stage)
	if !analyzeConfigBlock(ctx, ctx.AST.ConfigBlock(), stageScope, &stageType.Config) {
		return false
	}
	if !analyzeParams(acontext.Child(ctx, ctx.AST.ParameterList()).WithScope(stageScope), &stageType.Params) {
		return false
	}
	if !analyzeReturnTypeStage(ctx, ctx.AST.ReturnType(), stageScope, &stageType.Outputs) {
		return false
	}
	if block := ctx.AST.Block(); block != nil {
		stageType.Body = ir.Body{AST: block}
	}
	stageScope.Type = stageType
	if block := ctx.AST.Block(); block != nil {
		stageScope.OnResolve = func(ctx context.Context, s *ir.Scope) error {
			if s.Kind == ir.KindChannel {
				stageType.Channels.Read.Add(uint32(s.ID))
			}
			return nil
		}
		if !statement.AnalyzeBlock(acontext.Child(ctx, block).WithScope(stageScope)) {
			return false
		}
		if outputType, hasOutput := stageType.Outputs.Get("output"); hasOutput && outputType != nil && !blockAlwaysReturns(block) {
			ctx.Diagnostics.AddError(errors.Newf("stage '%s' must return a value of type %s on all paths", name, outputType), ctx.AST)
			return false
		}
		for outputName := range stageType.Outputs.Iter() {
			if outputName != "output" && !checkOutputAssignedInBlock(block, outputName) {
				ctx.Diagnostics.AddWarning(errors.Newf("output '%s' is never assigned in stage '%s'", outputName, name), ctx.AST)
			}
		}
	}
	return true
}

func analyzeConfigBlock(
	ctx acontext.Context[parser.IStageDeclarationContext],
	configBlock parser.IConfigBlockContext,
	scope *ir.Scope,
	config *maps.Ordered[string, ir.Type],
) bool {
	if configBlock == nil {
		return true
	}
	for _, param := range configBlock.AllConfigParameter() {
		paramName := param.IDENTIFIER().GetText()
		var configType ir.Type
		if typeCtx := param.Type_(); typeCtx != nil {
			configType, _ = atypes.InferFromTypeContext(typeCtx)
		}
		if !config.Put(paramName, configType) {
			ctx.Diagnostics.AddError(errors.Newf("duplicate configuration parameter %s", paramName), param)
			return false
		}
		if _, err := scope.Add(ctx, ir.Symbol{Name: paramName, Kind: ir.KindConfigParam, Type: configType, ParserRule: param}); err != nil {
			ctx.Diagnostics.AddError(err, param)
			return false
		}
	}
	return true
}
