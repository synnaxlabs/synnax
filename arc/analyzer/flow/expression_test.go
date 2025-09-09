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
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/text"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Expression Stage Conversion", func() {
	// Create a resolver with some test channels
	testResolver := ir.MapResolver{
		"temp_sensor": ir.Symbol{
			Name: "temp_sensor",
			Kind: ir.KindChannel,
			Type: ir.Chan{ValueType: ir.F32{}},
		},
		"pressure": ir.Symbol{
			Name: "pressure",
			Kind: ir.KindChannel,
			Type: ir.Chan{ValueType: ir.F64{}},
		},
		"ox_pt_1": ir.Symbol{
			Name: "ox_pt_1",
			Kind: ir.KindChannel,
			Type: ir.Chan{ValueType: ir.F64{}},
		},
		"ox_pt_2": ir.Symbol{
			Name: "ox_pt_2",
			Kind: ir.KindChannel,
			Type: ir.Chan{ValueType: ir.F64{}},
		},
		"alarm": ir.Symbol{
			Name: "alarm",
			Kind: ir.KindStage,
			Type: &ir.Stage{
				Config: types.NewOrderedMap([]string{"input"}, []ir.Type{ir.Chan{ValueType: ir.U8{}}}),
			},
		},
		"logger": ir.Symbol{
			Name: "logger",
			Kind: ir.KindStage,
			Type: &ir.Stage{
				Config: types.NewOrderedMap([]string{"input"}, []ir.Type{ir.Chan{ValueType: ir.F32{}}}),
			},
		},
		"display": ir.Symbol{
			Name: "display",
			Kind: ir.KindStage,
			Type: &ir.Stage{
				Config: types.NewOrderedMap([]string{"input"}, []ir.Type{ir.Chan{ValueType: ir.F64{}}}),
			},
		},
		"warning": ir.Symbol{
			Name: "warning",
			Kind: ir.KindStage,
			Type: &ir.Stage{},
		},
		"alarm_ch": ir.Symbol{
			Name: "alarm_ch",
			Kind: ir.KindChannel,
			Type: ir.Chan{ValueType: ir.U8{}},
		},
	}

	Context("Binary expression with channels", func() {
		It("should convert comparison expression to synthetic stage", func() {
			ast := MustSucceed(text.Parse(`
				ox_pt_1 > 100 -> alarm{}
			`))
			result := analyzer.AnalyzeProgram(ast, text.Options{Resolver: testResolver})
			Expect(result.Diagnostics).To(HaveLen(0))
			taskSymbol := MustSucceed(result.Symbols.Resolve("__expr_0"))
			Expect(taskSymbol.Name).To(Equal("__expr_0"))
			Expect(taskSymbol.Kind).To(Equal(ir.KindStage))
			stageType, ok := taskSymbol.Type.(ir.Stage)
			Expect(ok).To(BeTrue())
			Expect(stageType.Channels.Read).To(HaveLen(1))
			Expect(stageType.Channels.Read.Contains("ox_pt_1")).To(BeTrue())
		})

		It("should extract multiple channels from arithmetic expressions", func() {
			ast := MustSucceed(text.Parse(`
				(ox_pt_1 + ox_pt_2) / 2 -> display{}
			`))
			result := analyzer.AnalyzeProgram(ast, text.Options{Resolver: testResolver})
			Expect(result.Diagnostics).To(HaveLen(0))

			synthTask, err := result.Symbols.Resolve("__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())

			stageType := synthTask.Type.(ir.Stage)
			// Should have both channels in Channels.Read
			Expect(stageType.Channels.Read).To(HaveLen(2))
			Expect(stageType.Channels.Read.Keys()).To(ContainElements("ox_pt_1", "ox_pt_2"))

			// Return type should be F64 (result of arithmetic)
			Expect(stageType.Return).To(Equal(ir.F64{}))
		})
	})

	Context("Complex logical expressions", func() {
		It("should handle logical AND with multiple channels", func() {
			ast := MustSucceed(text.Parse(`
				ox_pt_1 > 100 && pressure > 50 -> alarm_ch
			`))
			result := analyzer.AnalyzeProgram(ast, text.Options{Resolver: testResolver})
			Expect(result.Diagnostics).To(HaveLen(0))

			synthTask, err := result.Symbols.Resolve("__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())

			stageType := synthTask.Type.(ir.Stage)
			// Should have both channels in Channels.Read
			Expect(stageType.Channels.Read).To(HaveLen(2))
			Expect(stageType.Channels.Read.Keys()).To(ContainElements("ox_pt_1", "pressure"))
		})
	})

	Context("Error cases", func() {
		It("should reject unknown channels in expressions", func() {
			ast := MustSucceed(text.Parse(`
				unknown_channel > 100 -> alarm{}
			`))
			result := analyzer.AnalyzeProgram(ast, text.Options{Resolver: testResolver})
			// Should have error for unknown channel
			Expect(result.Diagnostics).ToNot(BeEmpty())
			Expect(result.Diagnostics[0].Message).To(ContainSubstring("unknown"))
		})
	})

	Context("Multiple expressions in flow", func() {
		It("should create separate tasks for each expression", func() {
			ast := MustSucceed(text.Parse(`
				ox_pt_1 > 100 -> alarm{}
				pressure < 50 -> warning{}
				temp_sensor * f32(1.8) + f32(32) -> display{}
			`))
			result := analyzer.AnalyzeProgram(ast, text.Options{Resolver: testResolver})

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
			ast := MustSucceed(text.Parse(`
				((ox_pt_1 + ox_pt_2) * 2) > 100 -> alarm{}
			`))
			result := analyzer.AnalyzeProgram(ast, text.Options{Resolver: testResolver})
			Expect(result.Diagnostics).To(HaveLen(0))

			synthTask, err := result.Symbols.Resolve("__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())

			stageType := synthTask.Type.(ir.Stage)
			// Should extract both channels despite nesting
			Expect(stageType.Channels.Read.Len()).To(Equal(2))
			Expect(stageType.Channels.Read.Keys()).To(ContainElements("ox_pt_1", "ox_pt_2"))
		})
	})

	Context("Channel type preservation", func() {
		It("should preserve channel types in config", func() {
			ast := MustSucceed(text.Parse(`
				(temp_sensor) -> display{}
			`))
			result := analyzer.AnalyzeProgram(ast, text.Options{Resolver: testResolver})
			// May have type warning since display expects f64 but temp_sensor is f32

			synthTask, err := result.Symbols.Resolve("__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())

			stageType := synthTask.Type.(ir.Stage)
			// Should have the channel in Channels.Read
			Expect(stageType.Channels.Read).To(HaveLen(1))
			Expect(stageType.Channels.Read.Contains("temp_sensor")).To(BeTrue())
		})
	})
})
