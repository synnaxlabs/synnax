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

func newMockPolymorphicResolver() symbol.Resolver {
	constraint := types.NumericConstraint()
	simpleInputs := types.Params{{Name: "a", Type: types.Variable("T", &constraint)}}
	return &symbol.MapResolver{
		"simple": {
			Name: "simple",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Inputs: simpleInputs,
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.Variable("T", &constraint)},
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
	resolver := newMockPolymorphicResolver()

	type polymorphicCase struct {
		expectedType types.Type
		source       string
	}

	DescribeTable("Simple Polymorphic Flow",
		func(tc polymorphicCase) {
			ast := MustSucceed(parser.Parse(tc.source))
			ctx := acontext.CreateRoot(context.Background(), ast, resolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())

			simpleSymbol := MustSucceed(ctx.Scope.Resolve(ctx, "simple"))
			aType := MustBeOk(simpleSymbol.Type.Inputs.Get("a"))
			resolvedParam := ctx.Constraints.ApplySubstitutions(aType.Type)
			returnType := MustBeOk(simpleSymbol.Type.Outputs.Get(ir.DefaultOutputParam))
			resolvedReturn := ctx.Constraints.ApplySubstitutions(returnType.Type)
			Expect(resolvedParam).To(Equal(tc.expectedType))
			Expect(resolvedReturn).To(Equal(tc.expectedType))
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
})
