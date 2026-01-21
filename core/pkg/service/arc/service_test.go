// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AnalyzeCalculation", func() {
	It("Should return the data type for a valid integer expression", func() {
		dataType := MustSucceed(svc.AnalyzeCalculation(ctx, "return 1 + 2"))
		Expect(dataType).To(Equal(telem.Int64T))
	})

	It("Should return the data type for a valid float expression", func() {
		dataType := MustSucceed(svc.AnalyzeCalculation(ctx, "return 1.0 + 2.0"))
		Expect(dataType).To(Equal(telem.Float64T))
	})

	It("Should return parser error for invalid expression syntax", func() {
		Expect(svc.AnalyzeCalculation(ctx, "return 1 +")).
			Error().To(MatchError(ContainSubstring("mismatched input")))
	})

	It("Should return diagnostic error for undefined variable", func() {
		Expect(svc.AnalyzeCalculation(ctx, "return undefined_var + 1")).
			Error().To(MatchError(ContainSubstring("undefined symbol")))
	})
})

var _ = Describe("CompileModule", func() {
	It("Should retrieve and compile an Arc with a valid graph", func() {
		a := arc.Arc{
			Name: "test-arc",
			Graph: graph.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Inputs: types.Params{
							{Name: "a", Type: types.I64()},
							{Name: "b", Type: types.I64()},
						},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
						Body: ir.Body{Raw: "{ return a + b }"},
					},
				},
			},
		}
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())

		result := MustSucceed(svc.CompileModule(ctx, a.Key))
		Expect(result.Key).To(Equal(a.Key))
		Expect(result.Module.IR).ToNot(BeNil())
	})

	It("Should return error when Arc does not exist", func() {
		nonExistentKey := uuid.New()
		Expect(svc.CompileModule(ctx, nonExistentKey)).Error().
			To(MatchError(query.ErrNotFound))
	})

	It("Should return error when graph compilation fails", func() {
		a := arc.Arc{
			Name: "invalid-arc",
			Graph: graph.Graph{
				Functions: []ir.Function{
					{
						Key: "source",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.F32()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "src", Type: "source"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "nonexistent", Param: "input"},
					},
				},
			},
		}
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())

		Expect(svc.CompileModule(ctx, a.Key)).Error().
			To(MatchError(ContainSubstring("edge target node 'nonexistent' not found")))
	})
})
