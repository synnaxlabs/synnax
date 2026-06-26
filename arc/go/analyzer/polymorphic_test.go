// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

func newMockPolymorphicSymbols() []symbol.Symbol {
	constraint := types.NumericConstraint()
	simpleInputs := types.Params{{Name: "a", Type: types.Variable("T", &constraint)}}
	return []symbol.Symbol{
		{
			Name: "simple",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Inputs: simpleInputs,
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.Variable("T", &constraint)},
				},
			}),
			Trigger: symbol.TriggerInput("a"),
		},
		{
			Name: "sensor_f32",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F32()),
		},
	}
}

var _ = Describe("Polymorphic func Analysis", func() {
	extras := newMockPolymorphicSymbols()
	var root *symbol.Symbol
	BeforeEach(func() { root = NewRoot(nil, extras...) })

	type polymorphicCase struct {
		expectedType types.Type
		source       string
	}

	DescribeTable("Simple Polymorphic Flow",
		func(sCtx SpecContext, tc polymorphicCase) {
			ast := MustSucceed(parser.Parse(tc.source))
			ctx := acontext.NewRoot(sCtx, ast, root)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(ctx.Constraints.Substitutions).To(ContainElement(tc.expectedType))
		},
		Entry("infers types from channel inputs",
			polymorphicCase{
				source:       `sensor_f32 -> simple{}`,
				expectedType: types.F32(),
			}),
		Entry("infers types from expression inputs",
			polymorphicCase{
				source:       `(f32(1.5) + f32(2.5)) -> simple{}`,
				expectedType: types.F32(),
			}),
	)

	It("Should accept two calls to the same polymorphic func with different concrete types", func(sCtx SpecContext) {
		i64Extras := []symbol.Symbol{
			{Name: "sensor_i64", Kind: symbol.KindChannel, Type: types.Chan(types.I64())},
			{Name: "out_f32", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			{Name: "out_i64", Kind: symbol.KindChannel, Type: types.Chan(types.I64())},
		}
		root = NewRoot(nil, append(extras, i64Extras...)...)
		src := `sensor_f32 -> simple{} -> out_f32
sensor_i64 -> simple{} -> out_i64`
		ast := MustSucceed(parser.Parse(src))
		ctx := acontext.NewRoot(sCtx, ast, root)
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
	})

	// Pins the prev-freshening contract: at each func→func boundary, the
	// previous function is re-freshened with the same position-derived key
	// used when it was the current function, so its solved type variables
	// resolve through the constraint store from the earlier node analysis.
	// A chain of three polymorphic functions hits this path twice and would
	// silently mis-propagate types if constraint retention regressed.
	It("Should propagate types through a chain of three polymorphic funcs", func(sCtx SpecContext) {
		chainExtras := []symbol.Symbol{
			{Name: "out_f32", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
		}
		root = NewRoot(nil, append(extras, chainExtras...)...)
		src := `sensor_f32 -> simple{} -> simple{} -> simple{} -> out_f32`
		ast := MustSucceed(parser.Parse(src))
		ctx := acontext.NewRoot(sCtx, ast, root)
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		Expect(ctx.Constraints.Substitutions).To(ContainElement(types.F32()))
	})
})

var _ = Describe("Polymorphic func in module - cross-analysis", func() {
	buildExtras := func() []symbol.Symbol {
		c := types.NumericConstraint()
		simple := &symbol.Symbol{
			Name: "simple",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "a", Type: types.Variable("T", &c)}},
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.Variable("T", &c)},
				},
			}),
			Trigger: symbol.TriggerInput("a"),
		}
		mod := &symbol.Symbol{Name: "mymod", Kind: symbol.KindModule}
		mod.AddChild(simple)
		return []symbol.Symbol{
			*mod,
			{Name: "sensor_f32", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			{Name: "sensor_i64", Kind: symbol.KindChannel, Type: types.Chan(types.I64())},
			{Name: "out_f32", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			{Name: "out_i64", Kind: symbol.KindChannel, Type: types.Chan(types.I64())},
		}
	}

	It("Should not corrupt a module's polymorphic function type across analyses", func(sCtx SpecContext) {
		extras := buildExtras()

		root1 := NewRoot(nil, extras...)
		ast1 := MustSucceed(parser.Parse(`import mymod
sensor_f32 -> mymod.simple{} -> out_f32`))
		ctx1 := acontext.NewRoot(sCtx, ast1, root1)
		analyzer.AnalyzeProgram(ctx1)
		Expect(ctx1.Diagnostics.Ok()).To(BeTrue(), ctx1.Diagnostics.String())

		root2 := NewRoot(nil, extras...)
		ast2 := MustSucceed(parser.Parse(`import mymod
sensor_i64 -> mymod.simple{} -> out_i64`))
		ctx2 := acontext.NewRoot(sCtx, ast2, root2)
		analyzer.AnalyzeProgram(ctx2)
		Expect(ctx2.Diagnostics.Ok()).To(BeTrue(), ctx2.Diagnostics.String())
	})
})
