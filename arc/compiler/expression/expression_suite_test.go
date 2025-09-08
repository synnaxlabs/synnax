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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler/core"
	"github.com/synnaxlabs/arc/compiler/expression"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

func expectExpression(expression string, expectedType types.Type, expectedOpcodes ...any) {
	bytecode, exprType := compileExpression(expression)
	expected := WASM(expectedOpcodes...)
	Expect(bytecode).To(Equal(expected))
	Expect(exprType).To(Equal(expectedType))
}

func compileExpression(source string) ([]byte, types.Type) {
	return compileWithCtx(NewContext(), source)
}

func compileWithCtx(ctx *core.Context, source string) ([]byte, types.Type) {
	var (
		expr     = MustSucceedWithOffset[text.IExpressionContext](2)(text.ParseExpression(source))
		exprType = MustSucceedWithOffset[types.Type](2)(expression.Compile(ctx, expr, nil))
	)
	return ctx.Writer.Bytes(), exprType
}

func TestExpression(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expression Compiler Suite")
}
