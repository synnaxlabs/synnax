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
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var bCtx = context.Background()

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

func compileWithAnalyzer(exprSource string, resolver symbol.Resolver) ([]byte, types.Type) {
	expr := MustSucceed(parser.ParseExpression(exprSource))
	analyzerCtx := acontext.CreateRoot(bCtx, expr, resolver)
	Expect(aexpression.Analyze(analyzerCtx)).To(BeTrue(), analyzerCtx.Diagnostics.String())
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

func TestExpression(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expression Compiler Suite")
}
