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
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/analyzer/statement"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/maps"
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
	if !analyzeConfig(ctx, ctx.AST.ConfigBlock(), fn, fn.Type.Config) {
		return false
	}
	if !analyzeInputs(
		acontext.Child(ctx, ctx.AST.InputList()).WithScope(fn),
		fn.Type.Inputs,
	) {
		return false
	}
	if !analyzeOutputs(ctx, ctx.AST.OutputType(), fn, fn.Type.Outputs) {
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
		outputType, hasOutput := fn.Type.Outputs.Get(ir.DefaultOutputParam)
		if hasOutput && !blockAlwaysReturns(block) {
			ctx.Diagnostics.AddError(errors.Newf("function '%s' must return a value of type %s on all paths", name, outputType), ctx.AST)
			return false
		}
		for outputName := range fn.Type.Outputs.Iter() {
			if outputName != ir.DefaultOutputParam && !checkOutputAssignedInBlock(block, outputName) {
				ctx.Diagnostics.AddWarning(errors.Newf("output '%s' is never assigned in function '%s'", outputName, name), ctx.AST)
			}
		}
	}
	return true
}

func analyzeOutputs[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	outputType parser.IOutputTypeContext,
	scope *symbol.Scope,
	outputs *maps.Ordered[string, types.Type],
) bool {
	if outputType == nil {
		return true
	}
	if typeCtx := outputType.Type_(); typeCtx != nil {
		outputTypeVal, _ := atypes.InferFromTypeContext(typeCtx)
		if !outputs.Put(ir.DefaultOutputParam, outputTypeVal) {
			ctx.Diagnostics.AddError(errors.New("failed to add output"), outputType)
			return false
		}
		return true
	}
	if multiOutputBlock := outputType.MultiOutputBlock(); multiOutputBlock != nil {
		for _, namedOutput := range multiOutputBlock.AllNamedOutput() {
			outputName := namedOutput.IDENTIFIER().GetText()
			var outputTypeVal types.Type
			if typeCtx := namedOutput.Type_(); typeCtx != nil {
				outputTypeVal, _ = atypes.InferFromTypeContext(typeCtx)
			}
			if !outputs.Put(outputName, outputTypeVal) {
				ctx.Diagnostics.AddError(errors.Newf("duplicate output %s", outputName), namedOutput)
				return false
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

// parseLiteralValue parses a literal AST node and returns its value.
// It supports numeric literals (integer, float) and validates type compatibility.
// Returns an error if the literal cannot be parsed or is incompatible with expectedType.
func parseLiteralValue(
	ctx acontext.Context[parser.ILiteralContext],
	expectedType types.Type,
) (any, error) {
	if num := ctx.AST.NumericLiteral(); num != nil {
		return parseNumericLiteral(num, expectedType)
	}
	if temp := ctx.AST.TemporalLiteral(); temp != nil {
		// TODO: Parse temporal literals when needed
		return nil, errors.New("temporal literals not yet supported for default values")
	}
	if str := ctx.AST.STR_LITERAL(); str != nil {
		// TODO: Parse string literals when needed
		return nil, errors.New("string literals not yet supported for default values")
	}
	if series := ctx.AST.SeriesLiteral(); series != nil {
		return nil, errors.New("series literals not supported for default values")
	}
	return nil, errors.New("unknown literal type")
}

// parseNumericLiteral parses a numeric literal (integer or float) and returns its value.
func parseNumericLiteral(
	numLit parser.INumericLiteralContext,
	expectedType types.Type,
) (any, error) {
	if intLit := numLit.INTEGER_LITERAL(); intLit != nil {
		text := intLit.GetText()
		value, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid integer literal: %s", text)
		}

		// Convert to appropriate type based on expectedType
		switch expectedType.Kind {
		case types.KindI8:
			return int8(value), nil
		case types.KindI16:
			return int16(value), nil
		case types.KindI32:
			return int32(value), nil
		case types.KindI64:
			return value, nil
		case types.KindU8:
			return uint8(value), nil
		case types.KindU16:
			return uint16(value), nil
		case types.KindU32:
			return uint32(value), nil
		case types.KindU64:
			return uint64(value), nil
		case types.KindF32:
			return float32(value), nil
		case types.KindF64:
			return float64(value), nil
		default:
			// Default to i64 if type not specified or unknown
			return value, nil
		}
	}

	if floatLit := numLit.FLOAT_LITERAL(); floatLit != nil {
		text := floatLit.GetText()
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid float literal: %s", text)
		}

		// Convert to appropriate type based on expectedType
		switch expectedType.Kind {
		case types.KindF32:
			return float32(value), nil
		case types.KindF64:
			return value, nil
		case types.KindI8:
			return int8(value), nil
		case types.KindI16:
			return int16(value), nil
		case types.KindI32:
			return int32(value), nil
		case types.KindI64:
			return int64(value), nil
		case types.KindU8:
			return uint8(value), nil
		case types.KindU16:
			return uint16(value), nil
		case types.KindU32:
			return uint32(value), nil
		case types.KindU64:
			return uint64(value), nil
		default:
			// Default to f64 if type not specified or unknown
			return value, nil
		}
	}

	return nil, errors.New("unknown numeric literal")
}

func analyzeInputs(
	ctx acontext.Context[parser.IInputListContext],
	inputTypes *maps.Ordered[string, types.Type],
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
		if literal := input.Literal(); literal != nil {
			value, err := parseLiteralValue(acontext.Child(ctx, literal), inputType)
			if err != nil {
				ctx.Diagnostics.AddError(errors.Wrapf(err, "invalid default value for parameter %s", inputName), literal)
				return false
			}
			defaultValue = value
			seenOptional = true
		} else if seenOptional {
			// Required parameter after optional parameter (trailing-only violation)
			ctx.Diagnostics.AddError(
				errors.Newf("required parameter %s cannot follow optional parameters", inputName),
				input,
			)
			return false
		}

		if !inputTypes.Put(inputName, inputType) {
			ctx.Diagnostics.AddError(errors.Newf("duplicate input %s", inputName), input)
			return false
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

		// Store default value in function type if present
		if defaultValue != nil {
			if ctx.Scope.Type.InputDefaults == nil {
				ctx.Scope.Type.InputDefaults = make(map[string]any)
			}
			ctx.Scope.Type.InputDefaults[inputName] = defaultValue
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
	config *maps.Ordered[string, types.Type],
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
		if !config.Put(configName, configType) {
			ctx.Diagnostics.AddError(errors.Newf("duplicate config %s", configName), cfg)
			return false
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
