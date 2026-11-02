// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/x/testutil"
)

func findNode[T antlr.Tree](t antlr.Tree) T {
	if found, ok := t.(T); ok {
		return found
	}
	for i := 0; i < t.GetChildCount(); i++ {
		if found := findNode[T](t.GetChild(i)); any(found) != nil {
			return found
		}
	}
	var zero T
	return zero
}

var _ = Describe("Arguments", func() {
	Describe("CallArguments", func() {
		callOf := func(code string) parser.IFunctionCallSuffixContext {
			GinkgoHelper()
			expr := MustSucceed(parser.ParseExpression(code))
			call := findNode[parser.IFunctionCallSuffixContext](expr)
			Expect(call).ToNot(BeNil())
			return call
		}

		It("Should index positional arguments in order", func() {
			args := symbol.CallArguments(callOf("f(1, x)"))
			Expect(args).To(HaveLen(2))
			Expect(args[1].Index).To(Equal(1))
			Expect(args[1].Name).To(BeEmpty())
			Expect(args[1].Expr.GetText()).To(Equal("x"))
		})

		It("Should return nothing for an empty call", func() {
			Expect(symbol.CallArguments(callOf("f()"))).To(BeEmpty())
		})
	})

	Describe("FunctionArguments", func() {
		functionOf := func(code string) parser.IFunctionContext {
			GinkgoHelper()
			prog := MustSucceed(parser.Parse(code))
			fn := findNode[parser.IFunctionContext](prog)
			Expect(fn).ToNot(BeNil())
			return fn
		}

		It("Should index positional arguments in order", func() {
			args := symbol.FunctionArguments(functionOf("x -> f(1, y)"))
			Expect(args).To(HaveLen(2))
			Expect(args[1].Index).To(Equal(1))
			Expect(args[1].Name).To(BeEmpty())
			Expect(args[1].Expr.GetText()).To(Equal("y"))
		})

		It("Should name named arguments", func() {
			args := symbol.FunctionArguments(functionOf("x -> f(a=1, b=y)"))
			Expect(args).To(HaveLen(2))
			Expect(args[1].Name).To(Equal("b"))
			Expect(args[1].Expr.GetText()).To(Equal("y"))
		})

		It("Should return nothing for an empty call", func() {
			Expect(symbol.FunctionArguments(functionOf("x -> f()"))).To(BeEmpty())
		})
	})
})
