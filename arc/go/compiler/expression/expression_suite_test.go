// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	"context"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	aexpression "github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/compiler/bindings"
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var bCtx context.Context

var _ = BeforeEach(func() {
	bCtx = context.Background()
})

func expectExpression(expression string, expectedType types.Type, expectedOpcodes ...any) {
	bytecode, exprType := compileExpression(expression)
	Expect(bytecode).To(MatchOpcodes(expectedOpcodes...))
	Expect(exprType).To(Equal(expectedType))
}

func compileExpression(source string) ([]byte, types.Type) {
	return compileWithCtx(NewContext(bCtx), source)
}

func compileWithCtx(ctx ccontext.Context[antlr.ParserRuleContext], source string) ([]byte, types.Type) {
	var (
		expr     = MustSucceedWithOffset[parser.IExpressionContext](2)(parser.ParseExpression(source))
		exprType = MustSucceedWithOffset[types.Type](2)(expression.Compile(ccontext.Child(ctx, expr)))
	)
	return ctx.Writer.Bytes(), exprType
}

func compileWithCtxAndHint(ctx ccontext.Context[antlr.ParserRuleContext], source string, hint types.Type) ([]byte, types.Type) {
	expr := MustSucceedWithOffset[parser.IExpressionContext](2)(parser.ParseExpression(source))
	if hint.Kind == types.KindSeries {
		if lit := parser.GetLiteral(expr); lit != nil {
			ctx.TypeMap[lit] = hint
		}
	}
	exprType := MustSucceedWithOffset[types.Type](2)(expression.Compile(ccontext.Child(ctx, expr)))
	return ctx.Writer.Bytes(), exprType
}

func compileWithAnalyzer(exprSource string, resolver symbol.Resolver) ([]byte, types.Type) {
	expr := MustSucceed(parser.ParseExpression(exprSource))
	analyzerCtx := acontext.CreateRoot(bCtx, expr, resolver)
	aexpression.Analyze(analyzerCtx)
	Expect(analyzerCtx.Diagnostics.Ok()).To(BeTrue(), analyzerCtx.Diagnostics.String())
	if analyzerCtx.Constraints.HasTypeVariables() {
		Expect(analyzerCtx.Constraints.Unify()).To(Succeed())
		for node, typ := range analyzerCtx.TypeMap {
			analyzerCtx.TypeMap[node] = analyzerCtx.Constraints.ApplySubstitutions(typ)
		}
	}
	compilerCtx := ccontext.CreateRoot(bCtx, analyzerCtx.Scope, analyzerCtx.TypeMap, false)
	exprType := MustSucceed(expression.Compile(ccontext.Child(compilerCtx, expr)))
	return compilerCtx.Writer.Bytes(), exprType
}

// testImports provides function indices for test assertions
var testImports *bindings.ImportIndex

func init() {
	m := wasm.NewModule()
	testImports = bindings.SetupImports(m)
}

// expectSeriesExpression is a test helper for series operations that require symbol resolution
func expectSeriesExpression(
	expr string,
	resolver symbol.MapResolver,
	expectedType types.Type,
	expectedOpcodes ...any,
) {
	bytecode, exprType := compileWithAnalyzer(expr, resolver)
	Expect(exprType).To(Equal(expectedType))
	Expect(bytecode).To(MatchOpcodes(expectedOpcodes...))
}

// seriesSymbol creates a symbol for a series variable
func seriesSymbol(name string, elemType types.Type, id int) symbol.Symbol {
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindVariable,
		Type: types.Series(elemType),
		ID:   id,
	}
}

// seriesArithmeticIdx returns the function index for series arithmetic operations
func seriesArithmeticIdx(op string, elemType types.Type, isScalar bool) uint32 {
	return MustSucceed(testImports.GetSeriesArithmetic(op, elemType, isScalar))
}

// seriesReverseArithmeticIdx returns the function index for reverse arithmetic operations (scalar op series)
func seriesReverseArithmeticIdx(op string, elemType types.Type) uint32 {
	return MustSucceed(testImports.GetSeriesReverseArithmetic(op, elemType))
}

// seriesComparisonIdx returns the function index for series-series comparison operations
func seriesComparisonIdx(op string, elemType types.Type) uint32 {
	return MustSucceed(testImports.GetSeriesComparison(op, elemType))
}

func seriesCreateEmptyIdx(elemType types.Type) uint32 {
	return MustSucceed(testImports.GetSeriesCreateEmpty(elemType))
}

func seriesSetElementIdx(elemType types.Type) uint32 {
	return MustSucceed(testImports.GetSeriesSetElement(elemType))
}

func scalarSymbol(name string, t types.Type, id int) symbol.Symbol {
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindVariable,
		Type: t,
		ID:   id,
	}
}

func expectSeriesWithFunctions(
	expr string,
	funcIndices map[string]uint32,
	funcSymbols []symbol.Symbol,
	hint types.Type,
	expectedOpcodes ...any,
) {
	ctx := NewContext(bCtx)
	ctx.FunctionIndices = funcIndices
	for _, sym := range funcSymbols {
		MustSucceed(ctx.Scope.Add(ctx, sym))
	}
	bytecode, exprType := compileWithCtxAndHint(ctx, expr, hint)
	Expect(bytecode).To(MatchOpcodes(expectedOpcodes...))
	Expect(exprType).To(Equal(hint))
}

func expectSeriesLiteralWithHint(
	expr string,
	resolver symbol.Resolver,
	hint types.Type,
	expectedOpcodes ...any,
) {
	parsedExpr := MustSucceed(parser.ParseExpression(expr))
	analyzerCtx := acontext.CreateRoot(bCtx, parsedExpr, resolver)
	aexpression.Analyze(analyzerCtx)
	Expect(analyzerCtx.Diagnostics.Ok()).To(BeTrue(), analyzerCtx.Diagnostics.String())

	if analyzerCtx.Constraints.HasTypeVariables() {
		Expect(analyzerCtx.Constraints.Unify()).To(Succeed())
		for node, typ := range analyzerCtx.TypeMap {
			analyzerCtx.TypeMap[node] = analyzerCtx.Constraints.ApplySubstitutions(typ)
		}
	}

	if hint.IsValid() && hint.Kind == types.KindSeries {
		if lit := parser.GetLiteral(parsedExpr); lit != nil {
			analyzerCtx.TypeMap[lit] = hint
		}
	}

	compilerCtx := ccontext.CreateRoot(bCtx, analyzerCtx.Scope, analyzerCtx.TypeMap, false)
	exprType := MustSucceed(expression.Compile(ccontext.Child(compilerCtx, parsedExpr)))
	Expect(compilerCtx.Writer.Bytes()).To(MatchOpcodes(expectedOpcodes...))
	Expect(exprType).To(Equal(hint))
}

func TestExpression(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expression Compiler Suite")
}
