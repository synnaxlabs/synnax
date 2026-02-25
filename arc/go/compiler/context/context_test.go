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
	"context"

	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/resolve"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Context", func() {
	var (
		ctx     context.Context
		scope   *symbol.Scope
		typeMap map[antlr.ParserRuleContext]types.Type
	)

	BeforeEach(func() {
		ctx = context.Background()
		scope = symbol.CreateRootScope(nil)
		typeMap = make(map[antlr.ParserRuleContext]types.Type)
	})

	Describe("CreateRoot", func() {
		It("Should create a root context with initialized fields", func() {
			root := ccontext.CreateRoot(ctx, scope, typeMap, nil)
			Expect(root.Scope).To(Equal(scope))
			Expect(root.TypeMap).To(Equal(typeMap))
			Expect(root.Module).ToNot(BeNil())
			Expect(root.Writer).ToNot(BeNil())
			Expect(root.Resolver).To(BeNil())
		})

		It("Should track the writer when a resolver is provided", func() {
			resolver := resolve.NewResolver(nil)
			root := ccontext.CreateRoot(ctx, scope, typeMap, resolver)
			Expect(root.Resolver).To(Equal(resolver))
			Expect(root.WriterID).To(Equal(0))
		})
	})

	Describe("Child", func() {
		It("Should propagate all fields except AST", func() {
			root := ccontext.CreateRoot(ctx, scope, typeMap, nil)
			root.Hint = types.I32()
			root.OutputMemoryBase = 42
			child := ccontext.Child[antlr.ParserRuleContext, antlr.ParserRuleContext](root, nil)
			Expect(child.Scope).To(Equal(root.Scope))
			Expect(child.Writer).To(Equal(root.Writer))
			Expect(child.Module).To(Equal(root.Module))
			Expect(child.TypeMap).To(Equal(root.TypeMap))
			Expect(child.Hint).To(Equal(types.I32()))
			Expect(child.OutputMemoryBase).To(Equal(uint32(42)))
			Expect(child.WriterID).To(Equal(root.WriterID))
		})
	})

	Describe("WithHint", func() {
		It("Should return a context with the hint set", func() {
			root := ccontext.CreateRoot(ctx, scope, typeMap, nil)
			withHint := root.WithHint(types.F64())
			Expect(withHint.Hint).To(Equal(types.F64()))
		})

		It("Should not modify the original context", func() {
			root := ccontext.CreateRoot(ctx, scope, typeMap, nil)
			root.WithHint(types.F64())
			Expect(root.Hint).To(Equal(types.Type{}))
		})
	})

	Describe("WithScope", func() {
		It("Should return a context with the scope replaced", func() {
			root := ccontext.CreateRoot(ctx, scope, typeMap, nil)
			newScope := symbol.CreateRootScope(nil)
			withScope := root.WithScope(newScope)
			Expect(withScope.Scope).To(Equal(newScope))
		})

		It("Should not modify the original context", func() {
			root := ccontext.CreateRoot(ctx, scope, typeMap, nil)
			newScope := symbol.CreateRootScope(nil)
			root.WithScope(newScope)
			Expect(root.Scope).To(Equal(scope))
		})
	})

	Describe("WithNewWriter", func() {
		It("Should return a context with a fresh writer", func() {
			root := ccontext.CreateRoot(ctx, scope, typeMap, nil)
			originalWriter := root.Writer
			withNew := root.WithNewWriter()
			Expect(withNew.Writer).ToNot(BeNil())
			Expect(withNew.Writer).ToNot(BeIdenticalTo(originalWriter))
		})

		It("Should not modify the original context", func() {
			root := ccontext.CreateRoot(ctx, scope, typeMap, nil)
			originalWriter := root.Writer
			root.WithNewWriter()
			Expect(root.Writer).To(BeIdenticalTo(originalWriter))
		})

		It("Should track the new writer when a resolver is present", func() {
			resolver := resolve.NewResolver(nil)
			root := ccontext.CreateRoot(ctx, scope, typeMap, resolver)
			Expect(root.WriterID).To(Equal(0))
			withNew := root.WithNewWriter()
			Expect(withNew.WriterID).To(Equal(1))
		})

		It("Should produce distinct writers from the module writer", func() {
			root := ccontext.CreateRoot(ctx, scope, typeMap, nil)
			w1 := root.WithNewWriter()
			w2 := root.WithNewWriter()
			Expect(w1.Writer).ToNot(BeIdenticalTo(w2.Writer))
			w1.Writer.WriteOpcode(wasm.OpI32Add)
			Expect(w2.Writer.Bytes()).To(BeEmpty())
		})
	})
})
