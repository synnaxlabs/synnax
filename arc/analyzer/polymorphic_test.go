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
	"github.com/synnaxlabs/x/maps"
)

func NewMockPolymorphicResolver() ir.SymbolResolver {
	simpleParams := &maps.Ordered[string, ir.Type]{}
	simpleParams.Put("a", ir.NewTypeVariable("T", ir.NumericConstraint{}))
	return &ir.MapResolver{
		"simple": {
			Name: "simple",
			Kind: ir.KindStage,
			Type: ir.Stage{
				Key:    "simple",
				Params: *simpleParams,
				Return: ir.NewTypeVariable("T", ir.NumericConstraint{}),
			},
		},
		"sensor_f32": {
			Name: "sensor_f32",
			Kind: ir.KindChannel,
			Type: ir.Chan{ValueType: ir.F32{}},
		},
	}
}

var _ = Describe("Polymorphic Stage Analysis", func() {
	resolver := NewMockPolymorphicResolver()

	Context("Simple Polymorphic Flow", func() {
		It("should infer types for add stage from channel inputs", func() {
			src := `sensor_f32 -> simple{}`
			ast, err := parser.Parse(src)
			Expect(err).NotTo(HaveOccurred())

			ctx := acontext.CreateRoot(context.Background(), ast, resolver)
			ok := analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(ok).To(BeTrue())
			// After analysis, the simple stage definition remains polymorphic
			simpleSymbol, err := ctx.Scope.Resolve(ctx, "simple")
			Expect(err).NotTo(HaveOccurred())
			simpleStage := simpleSymbol.Type.(ir.Stage)

			// The stage definition keeps TypeVariables - that's what makes it polymorphic
			// We need to apply substitutions to get the concrete types for this specific use
			aType, ok := simpleStage.Params.Get("a")
			Expect(ok).To(BeTrue())

			resolvedParam := ctx.Constraints.ApplySubstitutions(aType)
			resolvedReturn := ctx.Constraints.ApplySubstitutions(simpleStage.Return)

			Expect(resolvedParam).To(Equal(ir.F32{}))
			Expect(resolvedReturn).To(Equal(ir.F32{}))
		})

		It("should infer types from expression inputs", func() {
			// Test: (1.5 + 2.5) -> simple{}
			src := `(f32(1.5) + f32(2.5)) -> simple{}`
			ast, err := parser.Parse(src)
			Expect(err).NotTo(HaveOccurred())

			ctx := acontext.CreateRoot(context.Background(), ast, resolver)
			ok := analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(ok).To(BeTrue())

			// After analysis, the simple stage should have F32 types
			simpleSymbol, err := ctx.Scope.Resolve(ctx, "simple")
			Expect(err).NotTo(HaveOccurred())
			simpleStage := simpleSymbol.Type.(ir.Stage)

			// Check that the parameter and return types resolve to F32 for this use
			aType, ok := simpleStage.Params.Get("a")
			Expect(ok).To(BeTrue())

			// Apply substitutions to get concrete types for this specific use
			resolvedParam := ctx.Constraints.ApplySubstitutions(aType)
			resolvedReturn := ctx.Constraints.ApplySubstitutions(simpleStage.Return)

			Expect(resolvedParam).To(Equal(ir.F32{}))
			Expect(resolvedReturn).To(Equal(ir.F32{}))
		})
	})
})
