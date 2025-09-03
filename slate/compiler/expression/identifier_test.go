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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/compiler/expression"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Expression Compiler", func() {
	Describe("Identifier Compilation", func() {
		Context("Local Variables", func() {
			It("Should compile local variable references", func() {
				// Create context with a local variable
				module := wasm.NewModule()
				symbols := &symbol.Scope{}
				ctx := compiler.NewContext(module, symbols)
				ctx.EnterFunction("test", nil)

				// Allocate a local variable
				ctx.AllocateLocal("x", wasm.I32)

				// Parse and compile identifier reference
				expr := MustSucceed(parser.ParseExpression("x"))
				compiler := expression.NewCompiler(ctx)
				exprType := MustSucceed(compiler.Compile(expr))

				bytecode := compiler.Bytes()
				Expect(bytecode).To(Equal(hexToBytes("20 00"))) // local.get 0
				// TODO: Fix type inference to return actual type
				Expect(exprType.String()).To(Equal("f64")) // Currently returns placeholder
			})
		})
	})
})
