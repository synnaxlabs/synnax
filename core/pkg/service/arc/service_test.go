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
	. "github.com/synnaxlabs/x/testutil"
)

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
		_, err := svc.CompileModule(ctx, nonExistentKey)
		Expect(err).To(MatchError(query.ErrNotFound))
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

		_, err := svc.CompileModule(ctx, a.Key)
		Expect(err).To(HaveOccurred())
	})
})
