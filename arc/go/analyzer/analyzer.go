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
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/analyzer/statement"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

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
		if !applyTypeSubstitutionsToSymbols(ctx, ctx.Scope) {
			return false
		}
		substituteTypeMap(ctx)
	}
	return true
}

func substituteTypeMap(ctx acontext.Context[parser.IProgramContext]) {
	for node, typ := range ctx.TypeMap {
		ctx.TypeMap[node] = ctx.Constraints.ApplySubstitutions(typ)
	}
}

func collectDeclarations(ctx acontext.Context[parser.IProgramContext]) bool {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if fn := item.FunctionDeclaration(); fn != nil {
			name := fn.IDENTIFIER().GetText()
			if _, err := ctx.Scope.Add(ctx, symbol.Symbol{
				Name: name,
				Kind: symbol.KindFunction,
				Type: types.Function(types.FunctionProperties{}),
				AST:  fn,
			}); err != nil {
				ctx.Diagnostics.AddError(err, fn)
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
		} else if flowStmt := item.FlowStatement(); flowStmt != nil {
			if !flow.Analyze(acontext.Child(ctx, flowStmt)) {
				return false
			}
		}
	}
	return true
}

func AnalyzeStatement(ctx acontext.Context[parser.IStatementContext]) bool {
	if !statement.Analyze(ctx) {
		return false
	}
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}
		applyTypeSubstitutionsToSymbols(ctx, ctx.Scope)
	}
	return true
}

func AnalyzeBlock(ctx acontext.Context[parser.IBlockContext]) bool {
	if !statement.AnalyzeBlock(ctx) {
		return false
	}
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}
		applyTypeSubstitutionsToSymbols(ctx, ctx.Scope)
	}
	return true
}

func analyzeFunctionDeclaration(ctx acontext.Context[parser.IFunctionDeclarationContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	fn, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	if !analyzeConfig(ctx, ctx.AST.ConfigBlock(), fn, &fn.Type.Config) {
		return false
	}
	if !analyzeInputs(
		acontext.Child(ctx, ctx.AST.InputList()).WithScope(fn),
		&fn.Type.Inputs,
	) {
		return false
	}
	if !analyzeOutputs(ctx, ctx.AST.OutputType(), fn, &fn.Type.Outputs) {
		return false
	}
	if block := ctx.AST.Block(); block != nil {
		fn.Channels = symbol.NewChannels()
		fn.OnResolve = func(ctx context.Context, s *symbol.Scope) error {
			if s.Kind == symbol.KindChannel || s.Type.Kind == types.KindChan {
				fn.Channels.Read[uint32(s.ID)] = s.Name
			}
			return nil
		}
		if !statement.AnalyzeBlock(acontext.Child(ctx, block).WithScope(fn)) {
			return false
		}
		oParam, hasOutput := fn.Type.Outputs.Get(ir.DefaultOutputParam)
		if hasOutput && !blockAlwaysReturns(block) {
			ctx.Diagnostics.AddError(errors.Newf(
				"function '%s' must return a value of type %s on all paths",
				name,
				oParam.Type,
			), ctx.AST)
			return false
		}
		for _, output := range fn.Type.Outputs {
			if output.Name != ir.DefaultOutputParam && !checkOutputAssignedInBlock(block, output.Name) {
				ctx.Diagnostics.AddWarning(errors.Newf(
					"output '%s' is never assigned in function '%s'",
					output.Name,
					name,
				), ctx.AST)
			}
		}
	}
	return true
}

func analyzeOutputs[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	outputType parser.IOutputTypeContext,
	scope *symbol.Scope,
	outputs *types.Params,
) bool {
	if outputType == nil {
		return true
	}
	if typeCtx := outputType.Type_(); typeCtx != nil {
		outputTypeVal, _ := atypes.InferFromTypeContext(typeCtx)
		*outputs = append(*outputs, types.Param{Name: ir.DefaultOutputParam, Type: outputTypeVal})
		return true
	}
	if multiOutputBlock := outputType.MultiOutputBlock(); multiOutputBlock != nil {
		for _, namedOutput := range multiOutputBlock.AllNamedOutput() {
			outputName := namedOutput.IDENTIFIER().GetText()
			var outputTypeVal types.Type
			if typeCtx := namedOutput.Type_(); typeCtx != nil {
				outputTypeVal, _ = atypes.InferFromTypeContext(typeCtx)
			}
			if outputs.Has(outputName) {
				ctx.Diagnostics.AddError(errors.Newf("duplicate output %s", outputName), namedOutput)
				return false
			}
			*outputs = append(*outputs, types.Param{Name: outputName, Type: outputTypeVal})
			if _, err := scope.Add(ctx, symbol.Symbol{Name: outputName, Kind: symbol.KindOutput, Type: outputTypeVal, AST: namedOutput}); err != nil {
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

func analyzeInputs(
	ctx acontext.Context[parser.IInputListContext],
	inputTypes *types.Params,
) bool {
	if ctx.AST == nil {
		return true
	}

	// Track whether we've seen an optional parameter for trailing-only validation
	seenOptional := false
	for _, input := range ctx.AST.AllInput() {
		var inputType types.Type
		if typeCtx := input.Type_(); typeCtx != nil {
			inputType, _ = atypes.InferFromTypeContext(typeCtx)
		}
		inputName := input.IDENTIFIER().GetText()

		// Parse default value if present
		var defaultValue any
		if lit := input.Literal(); lit != nil {
			value, err := literal.Parse(acontext.Child(ctx, lit).AST, inputType)
			if err != nil {
				ctx.Diagnostics.AddError(errors.Wrapf(
					err,
					"invalid default value for parameter %s",
					inputName,
				), lit)
				return false
			}
			defaultValue = value.Value
			seenOptional = true
		} else if seenOptional {
			ctx.Diagnostics.AddError(
				errors.Newf(
					"required parameter %s cannot follow optional parameters",
					inputName,
				),
				input,
			)
			return false
		}

		if inputTypes.Has(inputName) {
			ctx.Diagnostics.AddError(errors.Newf("duplicate input %s", inputName), input)
			return false
		}
		*inputTypes = append(*inputTypes, types.Param{
			Name:  inputName,
			Type:  inputType,
			Value: defaultValue,
		})
		if _, err := ctx.Scope.Add(ctx, symbol.Symbol{
			Name:         inputName,
			Kind:         symbol.KindInput,
			Type:         inputType,
			AST:          input,
			DefaultValue: defaultValue,
		}); err != nil {
			ctx.Diagnostics.AddError(err, input)
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

func applyTypeSubstitutionsToSymbols[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	scope *symbol.Scope,
) bool {
	if scope.Type.IsValid() {
		scope.Type = ctx.Constraints.ApplySubstitutions(scope.Type)
	}
	for _, child := range scope.Children {
		if !applyTypeSubstitutionsToSymbols[T](ctx, child) {
			return false
		}
	}
	return true
}

func analyzeConfig[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	configBlock parser.IConfigBlockContext,
	scope *symbol.Scope,
	config *types.Params,
) bool {
	if configBlock == nil {
		return true
	}
	for _, cfg := range configBlock.AllConfig() {
		configName := cfg.IDENTIFIER().GetText()
		var configType types.Type
		if typeCtx := cfg.Type_(); typeCtx != nil {
			configType, _ = atypes.InferFromTypeContext(typeCtx)
		}
		if config.Has(configName) {
			ctx.Diagnostics.AddError(errors.Newf("duplicate config %s", configName), cfg)
			return false
		}
		*config = append(*config, types.Param{Name: configName, Type: configType})
		if _, err := scope.Add(ctx, symbol.Symbol{
			Name: configName,
			Kind: symbol.KindConfig,
			Type: configType,
			AST:  cfg,
		}); err != nil {
			ctx.Diagnostics.AddError(err, cfg)
			return false
		}
	}
	return true
}
