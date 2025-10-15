// Copyright 2025 Synnax Labs, Inc.
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
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var bCtx = context.Background()

func expectExpression(expression string, expectedType types.Type, expectedOpcodes ...any) {
	bytecode, exprType := compileExpression(expression)
	expected := WASM(expectedOpcodes...)
	Expect(bytecode).To(Equal(expected))
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

func TestExpression(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expression Compiler Suite")
}
