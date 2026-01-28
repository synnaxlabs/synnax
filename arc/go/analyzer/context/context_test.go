// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package context_test

import (
	stdcontext "context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	analyzerContext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/testutil"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Context", func() {
	var bCtx stdcontext.Context

	BeforeEach(func() {
		bCtx = stdcontext.Background()
	})

	Describe("CreateRoot", func() {
		It("Should initialize all fields correctly", func() {
			ast := testutil.NewMockAST(1)
			ctx := analyzerContext.CreateRoot(bCtx, ast, nil)
			Expect(ctx.Context).To(Equal(bCtx))
			Expect(ctx.Scope).ToNot(BeNil())
			Expect(ctx.Diagnostics).ToNot(BeNil())
			Expect(*ctx.Diagnostics).To(HaveLen(0))
			Expect(ctx.Constraints).ToNot(BeNil())
			Expect(ctx.TypeMap).ToNot(BeNil())
			Expect(ctx.TypeMap).To(HaveLen(0))
			Expect(ctx.AST).To(Equal(ast))
			Expect(ctx.TypeHint).To(Equal(types.Type{}))
			Expect(ctx.InTypeInferenceMode).To(BeFalse())
		})
	})

	Describe("Child", func() {
		It("Should share all pointers except AST", func() {
			var (
				parentAST = testutil.NewMockAST(1)
				childAST  = testutil.NewMockAST(2)
				parent    = analyzerContext.CreateRoot(bCtx, parentAST, nil)
				child     = analyzerContext.Child(parent, childAST)
			)
			Expect(child.AST).To(Equal(childAST))
			Expect(child.AST).ToNot(Equal(parent.AST))
			Expect(child.Context).To(Equal(parent.Context))
			Expect(child.Scope).To(BeIdenticalTo(parent.Scope))
			Expect(child.Diagnostics).To(BeIdenticalTo(parent.Diagnostics))
			Expect(child.Constraints).To(BeIdenticalTo(parent.Constraints))
			Expect(child.TypeHint).To(Equal(parent.TypeHint))
			Expect(child.InTypeInferenceMode).To(Equal(parent.InTypeInferenceMode))
		})

		It("Should share state mutations", func() {
			var (
				parentAST = testutil.NewMockAST(1)
				childAST  = testutil.NewMockAST(2)
				parent    = analyzerContext.CreateRoot(bCtx, parentAST, nil)
				child     = analyzerContext.Child(parent, childAST)
			)
			child.Diagnostics.Add(diagnostics.Infof(childAST, "test diagnostic"))
			Expect(*parent.Diagnostics).To(HaveLen(1))
			testAST := testutil.NewMockAST(3)
			child.TypeMap[testAST] = types.I32()
			Expect(parent.TypeMap[testAST]).To(Equal(types.I32()))
		})

		It("Should preserve parent's TypeHint and InTypeInferenceMode", func() {
			parentAST := testutil.NewMockAST(1)
			childAST := testutil.NewMockAST(2)
			parent := analyzerContext.CreateRoot(bCtx, parentAST, nil)
			parent.TypeHint = types.F64()
			parent.InTypeInferenceMode = true
			child := analyzerContext.Child(parent, childAST)
			Expect(child.TypeHint).To(Equal(types.F64()))
			Expect(child.InTypeInferenceMode).To(BeTrue())
		})
	})

	Describe("WithScope", func() {
		It("Should return new context with updated scope", func() {
			var (
				ast           = testutil.NewMockAST(1)
				ctx           = analyzerContext.CreateRoot(bCtx, ast, nil)
				originalScope = ctx.Scope
				newScope      = MustSucceed(ctx.Scope.Add(bCtx, symbol.Symbol{
					Name: "test",
					Kind: symbol.KindFunction,
					Type: types.Function(types.FunctionProperties{}),
				}))
				newCtx = ctx.WithScope(newScope)
			)
			Expect(newCtx.Scope).To(Equal(newScope))
			Expect(newCtx.Scope).ToNot(Equal(originalScope))
			Expect(ctx.Scope).To(Equal(originalScope))
			Expect(newCtx.Context).To(Equal(ctx.Context))
			Expect(newCtx.Diagnostics).To(BeIdenticalTo(ctx.Diagnostics))
			Expect(newCtx.Constraints).To(BeIdenticalTo(ctx.Constraints))
			Expect(newCtx.AST).To(Equal(ctx.AST))
			Expect(newCtx.TypeHint).To(Equal(ctx.TypeHint))
			Expect(newCtx.InTypeInferenceMode).To(Equal(ctx.InTypeInferenceMode))
		})
	})

	Describe("WithTypeHint", func() {
		It("Should return new context with updated type hint", func() {
			var (
				ast    = testutil.NewMockAST(1)
				ctx    = analyzerContext.CreateRoot(bCtx, ast, nil)
				newCtx = ctx.WithTypeHint(types.F64())
			)
			Expect(newCtx.TypeHint).To(Equal(types.F64()))
			Expect(ctx.TypeHint).To(Equal(types.Type{}))
			Expect(newCtx.Context).To(Equal(ctx.Context))
			Expect(newCtx.Scope).To(Equal(ctx.Scope))
			Expect(newCtx.Diagnostics).To(BeIdenticalTo(ctx.Diagnostics))
			Expect(newCtx.Constraints).To(BeIdenticalTo(ctx.Constraints))
			Expect(newCtx.AST).To(Equal(ctx.AST))
			Expect(newCtx.InTypeInferenceMode).To(Equal(ctx.InTypeInferenceMode))
		})

		It("Should allow chaining with WithScope", func() {
			var (
				ast      = testutil.NewMockAST(1)
				ctx      = analyzerContext.CreateRoot(bCtx, ast, nil)
				newScope = MustSucceed(ctx.Scope.Add(bCtx, symbol.Symbol{
					Name: "test",
					Kind: symbol.KindFunction,
					Type: types.Function(types.FunctionProperties{}),
				}))
				newCtx = ctx.WithTypeHint(types.I32()).WithScope(newScope)
			)
			Expect(newCtx.TypeHint).To(Equal(types.I32()))
			Expect(newCtx.Scope).To(Equal(newScope))
		})
	})

	Describe("Integration", func() {
		It("Should support realistic workflow with one parsed AST", func() {
			var (
				prog     = MustSucceed(parser.Parse(`func test() {}`))
				rootCtx  = analyzerContext.CreateRoot(bCtx, prog, nil)
				newScope = MustSucceed(rootCtx.Scope.Add(bCtx, symbol.Symbol{
					Name: "x",
					Kind: symbol.KindVariable,
					Type: types.I32(),
				}))
				mockChild = testutil.NewMockAST(99)
				finalCtx  = analyzerContext.Child(rootCtx, mockChild).
						WithScope(newScope).
						WithTypeHint(types.String())
			)
			Expect(finalCtx.AST).To(Equal(mockChild))
			Expect(finalCtx.TypeHint).To(Equal(types.String()))
			Expect(finalCtx.Scope).To(Equal(newScope))
			finalCtx.Diagnostics.Add(diagnostics.Errorf(finalCtx.AST, "test"))
			Expect(*rootCtx.Diagnostics).To(HaveLen(1))
		})
	})
})
