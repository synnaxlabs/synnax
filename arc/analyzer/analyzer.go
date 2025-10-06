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
	context2 "context"

	"github.com/synnaxlabs/arc/analyzer/context"
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

func AnalyzeProgram(ctx context.Context[parser.IProgramContext]) bool {
	// PASS 1: Collect declarations with empty type signatures
	for _, item := range ctx.AST.AllTopLevelItem() {
		if fn := item.FunctionDeclaration(); fn != nil {
			name := fn.IDENTIFIER().GetText()
			_, err := ctx.Scope.Add(ctx, ir.Symbol{
				Name:       name,
				Kind:       ir.KindFunction,
				Type:       ir.Function{Key: name},
				ParserRule: fn,
			})
			if err != nil {
				ctx.Diagnostics.AddError(err, fn)
				return false
			}
		} else if stage := item.StageDeclaration(); stage != nil {
			name := stage.IDENTIFIER().GetText()
			_, err := ctx.Scope.Add(ctx, ir.Symbol{
				Name:       name,
				Kind:       ir.KindStage,
				Type:       ir.Stage{Key: name},
				ParserRule: stage,
			})
			if err != nil {
				ctx.Diagnostics.AddError(err, stage)
				return false
			}
		}
	}

	// PASS 2: Analyze tree and collect constraints
	for _, item := range ctx.AST.AllTopLevelItem() {
		if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
			if !analyzeFunctionDeclaration(context.Child(ctx, funcDecl)) {
				return false
			}
		} else if taskDecl := item.StageDeclaration(); taskDecl != nil {
			if !analyzeStageDeclaration(context.Child(ctx, taskDecl)) {
				return false
			}
		} else if flowStmt := item.FlowStatement(); flowStmt != nil {
			if !flow.Analyze(context.Child(ctx, flowStmt)) {
				return false
			}
		}
	}

	// PASS 3: Unify type variables and resolve all types
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}

		// Apply substitutions to all types in the symbol table
		if !applyTypeSubstitutions(ctx) {
			return false
		}
	}

	return true
}

func AnalyzeStatement(ctx context.Context[parser.IStatementContext]) bool {
	return statement.Analyze(ctx)
}

func AnalyzeBlock(ctx context.Context[parser.IBlockContext]) bool {
	return statement.AnalyzeBlock(ctx)
}

// analyzeFunctionDeclaration analyzes a function declaration
func analyzeFunctionDeclaration(ctx context.Context[parser.IFunctionDeclarationContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	fnScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	fnType := fnScope.Type.(ir.Function)
	if !analyzeParams(
		context.Child(ctx, ctx.AST.ParameterList()).WithScope(fnScope),
		&fnType.Params,
	) {
		return false
	}
	// Parse return type (single or multi-output)
	if retType := ctx.AST.ReturnType(); retType != nil {
		if typeCtx := retType.Type_(); typeCtx != nil {
			// Single return type
			fnType.Return, _ = atypes.InferFromTypeContext(typeCtx)
		} else if multiOutputBlock := retType.MultiOutputBlock(); multiOutputBlock != nil {
			// Multiple named outputs
			for _, namedOutput := range multiOutputBlock.AllNamedOutput() {
				outputName := namedOutput.IDENTIFIER().GetText()
				var outputType ir.Type
				if typeCtx := namedOutput.Type_(); typeCtx != nil {
					outputType, _ = atypes.InferFromTypeContext(typeCtx)
				}
				if !fnType.Outputs.Put(outputName, outputType) {
					ctx.Diagnostics.AddError(
						errors.Newf("duplicate output %s", outputName),
						namedOutput,
					)
					return false
				}

				// Also add to scope for use within function body
				if _, err := fnScope.Add(ctx, ir.Symbol{
					Name:       outputName,
					Kind:       ir.KindOutput,
					Type:       outputType,
					ParserRule: namedOutput,
				}); err != nil {
					ctx.Diagnostics.AddError(err, namedOutput)
					return false
				}
			}
		}
	}
	fnScope.Type = fnType
	if block := ctx.AST.Block(); block != nil {
		if !statement.AnalyzeBlock(context.Child(ctx, block).WithScope(fnScope)) {
			return false
		}
		// Check if the function has a return type and if all paths return
		if fnType.Return != nil && !blockAlwaysReturns(block) {
			ctx.Diagnostics.AddError(
				errors.Newf("function '%s' must return a value of type %s on all paths", name, fnType.Return),
				ctx.AST,
			)
			return false
		}
	}
	return true
}

// checkOutputAssignedInBlock checks if an output variable is assigned in the given block
func checkOutputAssignedInBlock(block parser.IBlockContext, outputName string) bool {
	if block == nil {
		return false
	}

	for _, stmt := range block.AllStatement() {
		// Check if it's an assignment to the output
		if assignment := stmt.Assignment(); assignment != nil {
			if assignment.IDENTIFIER().GetText() == outputName {
				return true
			}
		}

		// Check if statements
		if ifStmt := stmt.IfStatement(); ifStmt != nil {
			// Check main if block
			if checkOutputAssignedInBlock(ifStmt.Block(), outputName) {
				return true
			}
			// Check else-if blocks
			for _, elseIf := range ifStmt.AllElseIfClause() {
				if checkOutputAssignedInBlock(elseIf.Block(), outputName) {
					return true
				}
			}
			// Check else block
			if elseClause := ifStmt.ElseClause(); elseClause != nil {
				if checkOutputAssignedInBlock(elseClause.Block(), outputName) {
					return true
				}
			}
		}
	}

	return false
}

func analyzeParams(
	ctx context.Context[parser.IParameterListContext],
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
			ctx.Diagnostics.AddError(
				errors.Newf("duplicate parameter %s", paramName),
				param,
			)
			return false
		}

		// Also add to scope for use within stage body
		if _, err := ctx.Scope.Add(ctx, ir.Symbol{
			Name:       paramName,
			Kind:       ir.KindParam,
			Type:       paramType,
			ParserRule: param,
		}); err != nil {
			ctx.Diagnostics.AddError(err, param)
			return false
		}
	}
	return true
}

// blockAlwaysReturns checks if a block always returns a value on all execution paths
func blockAlwaysReturns(block parser.IBlockContext) bool {
	if block == nil {
		return false
	}
	statements := block.AllStatement()
	if len(statements) == 0 {
		return false
	}
	// Check statements from last to first
	for i := len(statements) - 1; i >= 0; i-- {
		stmt := statements[i]
		// Check if it's a return statement
		if stmt.ReturnStatement() != nil {
			return true
		}
		// Check if it's an if statement that covers all paths
		if ifStmt := stmt.IfStatement(); ifStmt != nil {
			// An if statement guarantees a return only if:
			// 1. It has an else clause
			// 2. All branches (if, else-ifs, else) return
			if ifStmt.ElseClause() != nil {
				// Check if the main if block returns
				if !blockAlwaysReturns(ifStmt.Block()) {
					continue // This if doesn't guarantee return
				}
				// Check all else-if branches
				allElseIfsReturn := true
				for _, elseIf := range ifStmt.AllElseIfClause() {
					if !blockAlwaysReturns(elseIf.Block()) {
						allElseIfsReturn = false
						break
					}
				}
				if !allElseIfsReturn {
					continue
				}
				if blockAlwaysReturns(ifStmt.ElseClause().Block()) {
					return true //
				}
			}
			// No else or not all branches return, continue checking previous statements
		}
		// For other statement types, continue checking previous statements
		// (assignments, expressions, etc. don't affect return paths)
	}
	return false
}

// applyTypeSubstitutions applies resolved type variables throughout the symbol table
func applyTypeSubstitutions(ctx context.Context[parser.IProgramContext]) bool {
	// Walk the entire symbol table and apply substitutions
	return applySubstitutionsToScope(ctx, ctx.Scope)
}

func applySubstitutionsToScope(ctx context.Context[parser.IProgramContext], scope *ir.Scope) bool {
	// Apply substitutions to the current scope's type
	if scope.Type != nil {
		scope.Type = ctx.Constraints.ApplySubstitutions(scope.Type)
	}

	// Recursively apply to children
	for _, child := range scope.Children {
		if !applySubstitutionsToScope(ctx, child) {
			return false
		}
	}

	return true
}

func analyzeStageDeclaration(ctx context.Context[parser.IStageDeclarationContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	stageScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	stageType := stageScope.Type.(ir.Stage)
	if configBlock := ctx.AST.ConfigBlock(); configBlock != nil {
		for _, param := range configBlock.AllConfigParameter() {
			paramName := param.IDENTIFIER().GetText()
			var configType ir.Type
			if typeCtx := param.Type_(); typeCtx != nil {
				configType, _ = atypes.InferFromTypeContext(typeCtx)
			}
			if !stageType.Config.Put(paramName, configType) {
				ctx.Diagnostics.AddError(
					errors.Newf("duplicate configuration parameter %s", param),
					ctx.AST,
				)
			}
			_, err = stageScope.Add(ctx, ir.Symbol{
				Name:       paramName,
				Kind:       ir.KindConfigParam,
				Type:       configType,
				ParserRule: param,
			})
			if err != nil {
				ctx.Diagnostics.AddError(err, param)
				return false
			}
		}
	}

	if !analyzeParams(
		context.Child(ctx, ctx.AST.ParameterList()).WithScope(stageScope),
		&stageType.Params,
	) {
		return false
	}

	// Parse return type (single or multi-output)
	if retType := ctx.AST.ReturnType(); retType != nil {
		if typeCtx := retType.Type_(); typeCtx != nil {
			// Single return type
			stageType.Return, _ = atypes.InferFromTypeContext(typeCtx)
		} else if multiOutputBlock := retType.MultiOutputBlock(); multiOutputBlock != nil {
			// Multiple named outputs
			for _, namedOutput := range multiOutputBlock.AllNamedOutput() {
				outputName := namedOutput.IDENTIFIER().GetText()
				var outputType ir.Type
				if typeCtx := namedOutput.Type_(); typeCtx != nil {
					outputType, _ = atypes.InferFromTypeContext(typeCtx)
				}
				if !stageType.Outputs.Put(outputName, outputType) {
					ctx.Diagnostics.AddError(
						errors.Newf("duplicate output %s", outputName),
						namedOutput,
					)
					return false
				}

				// Also add to scope for use within stage body
				if _, err := stageScope.Add(ctx, ir.Symbol{
					Name:       outputName,
					Kind:       ir.KindOutput,
					Type:       outputType,
					ParserRule: namedOutput,
				}); err != nil {
					ctx.Diagnostics.AddError(err, namedOutput)
					return false
				}
			}
		}
	}
	stageScope.Type = stageType
	if block := ctx.AST.Block(); block != nil {
		// Track output assignments
		outputAssignments := make(map[string]bool)

		stageScope.OnResolve = func(ctx context2.Context, s *ir.Scope) error {
			if s.Kind == ir.KindChannel {
				stageType.Channels.Read.Add(uint32(s.ID))
			}
			// Track assignments to outputs
			if s.Kind == ir.KindOutput {
				outputAssignments[s.Name] = true
			}
			return nil
		}
		if !statement.AnalyzeBlock(context.Child(ctx, block).WithScope(stageScope)) {
			return false
		}

		// Check if the stage has a return type and if all paths return
		if stageType.Return != nil && !blockAlwaysReturns(block) {
			ctx.Diagnostics.AddError(
				errors.Newf("stage '%s' must return a value of type %s on all paths", name, stageType.Return),
				ctx.AST,
			)
			return false
		}

		// Check for unassigned named outputs
		if stageType.HasNamedOutputs() {
			for outputName := range stageType.Outputs.Iter() {
				if !checkOutputAssignedInBlock(block, outputName) {
					ctx.Diagnostics.AddWarning(
						errors.Newf("output '%s' is never assigned in stage '%s'", outputName, name),
						ctx.AST,
					)
				}
			}
		}
	}
	return true
}
