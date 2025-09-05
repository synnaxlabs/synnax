// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package flow_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Expression Task Conversion", func() {
	// Create a resolver with some test channels
	testResolver := symbol.MapResolver{
		"temp_sensor": symbol.Symbol{
			Name: "temp_sensor",
			Kind: symbol.KindChannel,
			Type: types.Chan{ValueType: types.F32{}},
		},
		"pressure": symbol.Symbol{
			Name: "pressure",
			Kind: symbol.KindChannel,
			Type: types.Chan{ValueType: types.F64{}},
		},
		"ox_pt_1": symbol.Symbol{
			Name: "ox_pt_1",
			Kind: symbol.KindChannel,
			Type: types.Chan{ValueType: types.F64{}},
		},
		"ox_pt_2": symbol.Symbol{
			Name: "ox_pt_2",
			Kind: symbol.KindChannel,
			Type: types.Chan{ValueType: types.F64{}},
		},
		"alarm": symbol.Symbol{
			Name: "alarm",
			Kind: symbol.KindTask,
			Type: &types.Task{
				Config: types.NewOrderedMap([]string{"input"}, []types.Type{types.Chan{ValueType: types.U8{}}}),
			},
		},
		"logger": symbol.Symbol{
			Name: "logger",
			Kind: symbol.KindTask,
			Type: &types.Task{
				Config: types.NewOrderedMap([]string{"input"}, []types.Type{types.Chan{ValueType: types.F32{}}}),
			},
		},
		"display": symbol.Symbol{
			Name: "display",
			Kind: symbol.KindTask,
			Type: &types.Task{
				Config: types.NewOrderedMap([]string{"input"}, []types.Type{types.Chan{ValueType: types.F64{}}}),
			},
		},
		"warning": symbol.Symbol{
			Name: "warning",
			Kind: symbol.KindTask,
			Type: &types.Task{},
		},
		"alarm_ch": symbol.Symbol{
			Name: "alarm_ch",
			Kind: symbol.KindChannel,
			Type: types.Chan{ValueType: types.U8{}},
		},
	}

	Context("Binary expression with channels", func() {
		It("should convert comparison expression to synthetic task", func() {
			ast := MustSucceed(parser.Parse(`
				ox_pt_1 > 100 -> alarm{}
			`))
			result := analyzer.Analyze(ast, analyzer.Options{Resolver: testResolver})
			Expect(result.Diagnostics).To(HaveLen(0))
			taskSymbol := MustSucceed(result.Symbols.Resolve("__expr_0"))
			Expect(taskSymbol.Name).To(Equal("__expr_0"))
			Expect(taskSymbol.Kind).To(Equal(symbol.KindTask))
			taskType, ok := taskSymbol.Type.(types.Task)
			Expect(ok).To(BeTrue())
			Expect(taskType.Config.Count()).To(Equal(1))
			first := taskType.Config.At(0)
			Expect(first).To(Equal(types.F64{}))
		})

		It("should extract multiple channels from arithmetic expressions", func() {
			ast := MustSucceed(parser.Parse(`
				(ox_pt_1 + ox_pt_2) / 2 -> display{}
			`))
			result := analyzer.Analyze(ast, analyzer.Options{Resolver: testResolver})
			Expect(result.Diagnostics).To(HaveLen(0))

			synthTask, err := result.Symbols.Resolve("__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())

			taskType := synthTask.Type.(types.Task)
			// Should have both channels as config
			Expect(taskType.Config.Count()).To(Equal(2))
			_, hasOx1 := taskType.Config.Get("__ox_pt_1")
			_, hasOx2 := taskType.Config.Get("__ox_pt_2")
			Expect(hasOx1).To(BeTrue())
			Expect(hasOx2).To(BeTrue())

			// Return type should be F64 (result of arithmetic)
			Expect(taskType.Return).To(Equal(types.F64{}))
		})
	})

	Context("Complex logical expressions", func() {
		It("should handle logical AND with multiple channels", func() {
			ast := MustSucceed(parser.Parse(`
				ox_pt_1 > 100 && pressure > 50 -> alarm_ch
			`))
			result := analyzer.Analyze(ast, analyzer.Options{Resolver: testResolver})
			Expect(result.Diagnostics).To(HaveLen(0))

			synthTask, err := result.Symbols.Resolve("__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())

			taskType := synthTask.Type.(types.Task)
			// Should have both channels
			Expect(taskType.Config.Count()).To(Equal(2))
			_, hasOx := taskType.Config.Get("__ox_pt_1")
			_, hasPressure := taskType.Config.Get("__pressure")
			Expect(hasOx).To(BeTrue())
			Expect(hasPressure).To(BeTrue())
		})
	})

	Context("Error cases", func() {
		It("should reject unknown channels in expressions", func() {
			ast := MustSucceed(parser.Parse(`
				unknown_channel > 100 -> alarm{}
			`))
			result := analyzer.Analyze(ast, analyzer.Options{Resolver: testResolver})
			// Should have error for unknown channel
			Expect(result.Diagnostics).ToNot(BeEmpty())
			Expect(result.Diagnostics[0].Message).To(ContainSubstring("unknown"))
		})
	})

	Context("Multiple expressions in flow", func() {
		It("should create separate tasks for each expression", func() {
			ast := MustSucceed(parser.Parse(`
				ox_pt_1 > 100 -> alarm{}
				pressure < 50 -> warning{}
				temp_sensor * f32(1.8) + f32(32) -> display{}
			`))
			result := analyzer.Analyze(ast, analyzer.Options{Resolver: testResolver})

			// May have warnings about type mismatches, but should still create tasks
			// Check that multiple synthetic tasks were created
			task0, err0 := result.Symbols.Resolve("__expr_0")
			task1, err1 := result.Symbols.Resolve("__expr_1")
			task2, err2 := result.Symbols.Resolve("__expr_2")

			Expect(err0).To(BeNil())
			Expect(err1).To(BeNil())
			Expect(err2).To(BeNil())

			Expect(task0).ToNot(BeNil())
			Expect(task1).ToNot(BeNil())
			Expect(task2).ToNot(BeNil())
		})
	})

	Context("Parenthesized expressions", func() {
		It("should handle nested parentheses correctly", func() {
			ast := MustSucceed(parser.Parse(`
				((ox_pt_1 + ox_pt_2) * 2) > 100 -> alarm{}
			`))
			result := analyzer.Analyze(ast, analyzer.Options{Resolver: testResolver})
			Expect(result.Diagnostics).To(HaveLen(0))

			synthTask, err := result.Symbols.Resolve("__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())

			taskType := synthTask.Type.(types.Task)
			// Should extract both channels despite nesting
			Expect(taskType.Config.Count()).To(Equal(2))
		})
	})

	Context("Channel type preservation", func() {
		It("should preserve channel types in config", func() {
			ast := MustSucceed(parser.Parse(`
				(temp_sensor) -> display{}
			`))
			result := analyzer.Analyze(ast, analyzer.Options{Resolver: testResolver})
			// May have type warning since display expects f64 but temp_sensor is f32

			synthTask, err := result.Symbols.Resolve("__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())

			taskType := synthTask.Type.(types.Task)
			chanType, exists := taskType.Config.Get("__temp_sensor")
			Expect(exists).To(BeTrue())

			Expect(chanType).To(Equal(types.F32{}))
		})
	})
})
