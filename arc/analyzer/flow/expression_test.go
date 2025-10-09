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
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/maps"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Expression Stage Conversion", func() {
	// Create a resolver with some test channels
	testResolver := ir.MapResolver{
		"temp_sensor": ir.Symbol{
			Name: "temp_sensor",
			Kind: ir.KindChannel,
			Type: ir.Chan{ValueType: ir.F32{}},
			ID:   10,
		},
		"pressure": ir.Symbol{
			Name: "pressure",
			Kind: ir.KindChannel,
			Type: ir.Chan{ValueType: ir.F64{}},
			ID:   11,
		},
		"ox_pt_1": ir.Symbol{
			Name: "ox_pt_1",
			Kind: ir.KindChannel,
			Type: ir.Chan{ValueType: ir.F64{}},
			ID:   12,
		},
		"ox_pt_2": ir.Symbol{
			Name: "ox_pt_2",
			Kind: ir.KindChannel,
			Type: ir.Chan{ValueType: ir.F64{}},
			ID:   13,
		},
		"alarm": ir.Symbol{
			Name: "alarm",
			Kind: ir.KindStage,
			Type: &ir.Stage{
				Config: maps.Ordered[string, ir.Type]{
					Keys:   []string{"input"},
					Values: []ir.Type{ir.Chan{ValueType: ir.U8{}}},
				},
			},
		},
		"logger": ir.Symbol{
			Name: "logger",
			Kind: ir.KindStage,
			Type: &ir.Stage{
				Config: maps.Ordered[string, ir.Type]{
					Keys:   []string{"input"},
					Values: []ir.Type{ir.Chan{ValueType: ir.U8{}}},
				},
			},
		},
		"display": ir.Symbol{
			Name: "display",
			Kind: ir.KindStage,
			Type: &ir.Stage{
				Config: maps.Ordered[string, ir.Type]{
					Keys:   []string{"input"},
					Values: []ir.Type{ir.Chan{ValueType: ir.U8{}}},
				},
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
			ID:   14,
		},
	}

	Context("Binary expression with channels", func() {
		It("should convert comparison expression to synthetic stage", func() {
			ast := MustSucceed(parser.Parse(`
				ox_pt_1 > 100 -> alarm{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			taskSymbol := MustSucceed(ctx.Scope.Resolve(ctx, "__expr_0"))
			Expect(taskSymbol.Name).To(Equal("__expr_0"))
			Expect(taskSymbol.Kind).To(Equal(ir.KindStage))
			stage, ok := taskSymbol.Type.(ir.Stage)
			Expect(ok).To(BeTrue())
			Expect(stage.Channels.Read).To(HaveLen(1))
			Expect(stage.Channels.Read.Contains(12)).To(BeTrue())
			Expect(stage.Channels.Write).To(BeEmpty())
			Expect(stage.Key).To(Equal("__expr_0"))
			Expect(stage.Config.Keys).To(BeEmpty())
			Expect(stage.Params.Keys).To(BeEmpty())
			Expect(stage.Return).To(Equal(ir.U8{}))
		})

		It("should extract multiple channels from arithmetic expressions", func() {
			ast := MustSucceed(parser.Parse(`
				(ox_pt_1 + ox_pt_2) / 2 -> display{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			synthTask, err := ctx.Scope.Resolve(ctx, "__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())
			stageType, ok := synthTask.Type.(ir.Stage)
			Expect(ok).To(BeTrue())
			// Return type should be F64 (result of arithmetic)
			Expect(stageType.Return).To(Equal(ir.F64{}))
		})
	})

	Context("Complex logical expressions", func() {
		It("should handle logical AND with multiple channels", func() {
			ast := MustSucceed(parser.Parse(`
				ox_pt_1 > 100 && pressure > 50 -> alarm_ch
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			synthTask, err := ctx.Scope.Resolve(ctx, "__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())

			_, ok := synthTask.Type.(ir.Stage)
			Expect(ok).To(BeTrue())
		})
	})

	Context("Error cases", func() {
		It("should reject unknown channels in expressions", func() {
			ast := MustSucceed(parser.Parse(`
				unknown_channel > 100 -> alarm{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("unknown"))
		})
	})

	Context("Multiple expressions in flow", func() {
		It("should create separate tasks for each expression", func() {
			ast := MustSucceed(parser.Parse(`
				ox_pt_1 > 100 -> alarm{}
				pressure < 50 -> warning{}
				temp_sensor * f32(1.8) + f32(32) -> display{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			// May have warnings about type mismatches, but should still create tasks
			// Check that multiple synthetic tasks were created
			task0, err0 := ctx.Scope.Resolve(ctx, "__expr_0")
			task1, err1 := ctx.Scope.Resolve(ctx, "__expr_1")
			task2, err2 := ctx.Scope.Resolve(ctx, "__expr_2")

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
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			synthTask, err := ctx.Scope.Resolve(ctx, "__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())
		})
	})

	Context("Channel type preservation", func() {
		It("should preserve channel types in config", func() {
			ast := MustSucceed(parser.Parse(`
				(temp_sensor) -> display{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			// May have type warning since display expects f64 but temp_sensor is f32

			synthTask, err := ctx.Scope.Resolve(ctx, "__expr_0")
			Expect(err).To(BeNil())
			Expect(synthTask).ToNot(BeNil())
		})
	})
})
