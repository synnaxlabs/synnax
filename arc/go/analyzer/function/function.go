// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package function implements semantic analysis for Arc function declarations.
//
// Functions in Arc have three parameter categories:
//   - Config: Compile-time configuration (in curly braces)
//   - Inputs: Runtime parameters (in parentheses)
//   - Outputs: Return values (after the parameter list)
//
// The analyzer validates:
//   - Function names are unique
//   - Parameter names are unique within their category
//   - Input types are valid
//   - Output types are valid
//   - Optional parameters come after required parameters
//   - Functions with return types return on all code paths
//   - Named outputs are assigned in the function body
package function

import (
	"github.com/antlr4-go/antlr/v4"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// CollectDeclarations registers all function declarations in the symbol table,
// including their full signatures (config, inputs, outputs). This is called during
// the first pass of AnalyzeProgram to establish scopes and signatures before
// analyzing function bodies that may reference other functions.
func CollectDeclarations(ctx acontext.Context[parser.IProgramContext]) bool {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if fn := item.FunctionDeclaration(); fn != nil {
			name := fn.IDENTIFIER().GetText()

			// Collect signature (config, inputs, outputs) without adding params to scope
			var config, inputs, outputs types.Params
			if !collectConfig(ctx, fn.ConfigBlock(), &config) {
				return false
			}
			if !collectInputs(acontext.Child(ctx, fn.InputList()), &inputs) {
				return false
			}
			if !collectOutputs(ctx, fn.OutputType(), &outputs) {
				return false
			}

			if _, err := ctx.Scope.Add(ctx, symbol.Symbol{
				Name: name,
				Kind: symbol.KindFunction,
				Type: types.Function(types.FunctionProperties{
					Config:  config,
					Inputs:  inputs,
					Outputs: outputs,
				}),
				AST: fn,
			}); err != nil {
				ctx.Diagnostics.AddError(err, fn)
				return false
			}
		}
	}
	return true
}

// collectConfig extracts config parameter types without adding them to scope.
func collectConfig[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	configBlock parser.IConfigBlockContext,
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
		*config = append(*config, types.Param{Name: configName, Type: configType})
	}
	return true
}

// collectInputs extracts input parameter types without adding them to scope.
func collectInputs(
	ctx acontext.Context[parser.IInputListContext],
	inputs *types.Params,
) bool {
	if ctx.AST == nil {
		return true
	}
	seenOptional := false
	for _, input := range ctx.AST.AllInput() {
		var inputType types.Type
		if typeCtx := input.Type_(); typeCtx != nil {
			inputType, _ = atypes.InferFromTypeContext(typeCtx)
		}
		inputName := input.IDENTIFIER().GetText()

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

		*inputs = append(*inputs, types.Param{
			Name:  inputName,
			Type:  inputType,
			Value: defaultValue,
		})
	}
	return true
}

// collectOutputs extracts output types without adding them to scope.
func collectOutputs[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	outputType parser.IOutputTypeContext,
	outputs *types.Params,
) bool {
	if outputType == nil {
		return true
	}

	// Case 1: Single named output without parens (e.g., "result f64")
	if identifier := outputType.IDENTIFIER(); identifier != nil && outputType.Type_() != nil {
		outputName := identifier.GetText()
		outputTypeVal, _ := atypes.InferFromTypeContext(outputType.Type_())
		*outputs = append(*outputs, types.Param{Name: outputName, Type: outputTypeVal})
		return true
	}

	// Case 2: Unnamed single output (e.g., "f64")
	if typeCtx := outputType.Type_(); typeCtx != nil {
		outputTypeVal, _ := atypes.InferFromTypeContext(typeCtx)
		*outputs = append(*outputs, types.Param{Name: ir.DefaultOutputParam, Type: outputTypeVal})
		return true
	}

	// Case 3: Multiple or parenthesized outputs (e.g., "(result f64)" or "(a f64, b f64)")
	if multiOutputBlock := outputType.MultiOutputBlock(); multiOutputBlock != nil {
		for _, namedOutput := range multiOutputBlock.AllNamedOutput() {
			outputName := namedOutput.IDENTIFIER().GetText()
			var outputTypeVal types.Type
			if typeCtx := namedOutput.Type_(); typeCtx != nil {
				outputTypeVal, _ = atypes.InferFromTypeContext(typeCtx)
			}
			*outputs = append(*outputs, types.Param{Name: outputName, Type: outputTypeVal})
		}
	}
	return true
}

// Analyze performs semantic analysis on a function declaration.
// This is called during the second pass after all declarations have been collected.
// The function signature (config, inputs, outputs) is already populated by CollectDeclarations;
// this function adds the parameters to the function's scope and analyzes the body.
func Analyze(ctx acontext.Context[parser.IFunctionDeclarationContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	fn, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	// Add config, inputs, and outputs to the function's scope
	// (types are already populated by CollectDeclarations)
	if !addConfigToScope(ctx, ctx.AST.ConfigBlock(), fn) {
		return false
	}
	if !addInputsToScope(acontext.Child(ctx, ctx.AST.InputList()).WithScope(fn)) {
		return false
	}
	if !addOutputsToScope(ctx, ctx.AST.OutputType(), fn) {
		return false
	}

	if block := ctx.AST.Block(); block != nil {
		fn.AccumulateReadChannels()
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

// addOutputsToScope adds named output parameters to the function's scope.
// The output types are already collected in fn.Type.Outputs by CollectDeclarations.
func addOutputsToScope[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	outputType parser.IOutputTypeContext,
	scope *symbol.Scope,
) bool {
	if outputType == nil {
		return true
	}

	// Case 1: Single named output without parens (e.g., "result f64")
	if identifier := outputType.IDENTIFIER(); identifier != nil && outputType.Type_() != nil {
		outputName := identifier.GetText()
		outputTypeVal, _ := atypes.InferFromTypeContext(outputType.Type_())
		if _, err := scope.Add(ctx, symbol.Symbol{Name: outputName, Kind: symbol.KindOutput, Type: outputTypeVal, AST: outputType}); err != nil {
			ctx.Diagnostics.AddError(err, outputType)
			return false
		}
		return true
	}

	// Case 2: Unnamed single output (e.g., "f64") - nothing to add to scope
	if typeCtx := outputType.Type_(); typeCtx != nil {
		return true
	}

	// Case 3: Multiple or parenthesized outputs (e.g., "(result f64)" or "(a f64, b f64)")
	if multiOutputBlock := outputType.MultiOutputBlock(); multiOutputBlock != nil {
		for _, namedOutput := range multiOutputBlock.AllNamedOutput() {
			outputName := namedOutput.IDENTIFIER().GetText()
			var outputTypeVal types.Type
			if typeCtx := namedOutput.Type_(); typeCtx != nil {
				outputTypeVal, _ = atypes.InferFromTypeContext(typeCtx)
			}
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

// addInputsToScope adds input parameters to the function's scope.
// The input types are already collected in fn.Type.Inputs by CollectDeclarations.
func addInputsToScope(
	ctx acontext.Context[parser.IInputListContext],
) bool {
	if ctx.AST == nil {
		return true
	}

	for _, input := range ctx.AST.AllInput() {
		var inputType types.Type
		if typeCtx := input.Type_(); typeCtx != nil {
			inputType, _ = atypes.InferFromTypeContext(typeCtx)
		}
		inputName := input.IDENTIFIER().GetText()

		var defaultValue any
		if lit := input.Literal(); lit != nil {
			value, err := literal.Parse(acontext.Child(ctx, lit).AST, inputType)
			if err != nil {
				// This error was already reported in collectInputs
				return false
			}
			defaultValue = value.Value
		}

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

// addConfigToScope adds config parameters to the function's scope.
// The config types are already collected in fn.Type.Config by CollectDeclarations.
func addConfigToScope[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	configBlock parser.IConfigBlockContext,
	scope *symbol.Scope,
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
