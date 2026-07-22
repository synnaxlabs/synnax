// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("FormatFunctionSignature", func() {
	Describe("Basic Formatting", func() {
		It("Should format function with no inputs or outputs", func() {
			funcType := types.Function(types.FunctionProperties{})
			sig := ir.FormatFunctionSignature("noOp", funcType)
			Expect(sig).To(Equal("noOp()"))
		})

		It("Should format function with single input", func() {
			funcType := types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: types.I64()}},
			})
			sig := ir.FormatFunctionSignature("square", funcType)
			Expect(sig).To(Equal("square(x i64)"))
		})

		It("Should format function with multiple inputs", func() {
			funcType := types.Function(types.FunctionProperties{
				Inputs: types.Params{
					{Name: "x", Type: types.I64()},
					{Name: "y", Type: types.I64()},
				},
			})
			sig := ir.FormatFunctionSignature("add", funcType)
			Expect(sig).To(Equal("add(x i64, y i64)"))
		})

		It("Should format function with anonymous single output", func() {
			funcType := types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "x", Type: types.F64()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
			})
			sig := ir.FormatFunctionSignature("sqrt", funcType)
			Expect(sig).To(Equal("sqrt(x f64) f64"))
		})

		It("Should format function with named single output", func() {
			funcType := types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "x", Type: types.I64()}},
				Outputs: types.Params{{Name: "result", Type: types.I64()}},
			})
			sig := ir.FormatFunctionSignature("compute", funcType)
			Expect(sig).To(Equal("compute(x i64) result i64"))
		})

		It("Should format function with multiple outputs", func() {
			funcType := types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: types.I64()}},
				Outputs: types.Params{
					{Name: "quotient", Type: types.I64()},
					{Name: "remainder", Type: types.I64()},
				},
			})
			sig := ir.FormatFunctionSignature("divmod", funcType)
			Expect(sig).To(Equal("divmod(x i64) (quotient i64, remainder i64)"))
		})
	})

	Describe("Non-Function Types", func() {
		It("Should return just the name for non-function types", func() {
			sig := ir.FormatFunctionSignature("myVar", types.I64())
			Expect(sig).To(Equal("myVar"))
		})

		It("Should return just the name for channel types", func() {
			sig := ir.FormatFunctionSignature("myChan", types.Chan(types.F64()))
			Expect(sig).To(Equal("myChan"))
		})
	})

	Describe("Mixed Types", func() {
		It("Should format function with various input types", func() {
			funcType := types.Function(types.FunctionProperties{
				Inputs: types.Params{
					{Name: "count", Type: types.I32()},
					{Name: "ratio", Type: types.F64()},
					{Name: "name", Type: types.String()},
				},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			})
			sig := ir.FormatFunctionSignature("process", funcType)
			Expect(sig).To(Equal("process(count i32, ratio f64, name str) u8"))
		})
	})
})
