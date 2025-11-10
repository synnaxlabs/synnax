// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

func NewMockPolymorphicResolver() symbol.Resolver {
	simpleInputs := &types.Params{}
	constraint := types.NumericConstraint()
	simpleInputs.Put("a", types.Variable("T", &constraint))
	return &symbol.MapResolver{
		"simple": {
			Name: "simple",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Inputs: simpleInputs,
				Outputs: &types.Params{
					Keys:   []string{ir.DefaultOutputParam},
					Values: []types.Type{types.Variable("T", &constraint)},
				},
			}),
		},
		"sensor_f32": {
			Name: "sensor_f32",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F32()),
		},
	}
}

var _ = Describe("Polymorphic func Analysis", func() {
	resolver := NewMockPolymorphicResolver()

	Context("Simple Polymorphic Flow", func() {
		It("should infer types for add func from channel inputs", func() {
			src := `sensor_f32 -> simple{}`
			ast := MustSucceed(parser.Parse(src))
			ctx := acontext.CreateRoot(context.Background(), ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			simpleSymbol := MustSucceed(ctx.Scope.Resolve(ctx, "simple"))
			aType := MustBeOk(simpleSymbol.Type.Inputs.Get("a"))
			resolvedParam := ctx.Constraints.ApplySubstitutions(aType)
			returnType := MustBeOk(simpleSymbol.Type.Outputs.Get(ir.DefaultOutputParam))
			resolvedReturn := ctx.Constraints.ApplySubstitutions(returnType)
			Expect(resolvedParam).To(Equal(types.F32()))
			Expect(resolvedReturn).To(Equal(types.F32()))
		})

		It("should infer types from expression inputs", func() {
			src := `(f32(1.5) + f32(2.5)) -> simple{}`
			ast := MustSucceed(parser.Parse(src))
			ctx := acontext.CreateRoot(context.Background(), ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
			simpleSymbol := MustSucceed(ctx.Scope.Resolve(ctx, "simple"))
			aType := MustBeOk(simpleSymbol.Type.Inputs.Get("a"))
			resolvedParam := ctx.Constraints.ApplySubstitutions(aType)
			returnType := MustBeOk(simpleSymbol.Type.Outputs.Get(ir.DefaultOutputParam))
			resolvedReturn := ctx.Constraints.ApplySubstitutions(returnType)
			Expect(resolvedParam).To(Equal(types.F32()))
			Expect(resolvedReturn).To(Equal(types.F32()))
		})
	})
})
