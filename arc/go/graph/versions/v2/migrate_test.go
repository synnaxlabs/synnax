// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/arc/graph/versions/v1"
	v2 "github.com/synnaxlabs/arc/graph/versions/v2"
	irv1 "github.com/synnaxlabs/arc/ir/versions/v1"
	ir "github.com/synnaxlabs/arc/ir/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateGraph", func() {
	It("Should move node types and configs into inputs", func(ctx SpecContext) {
		migrated := MustSucceed(v2.MigrateGraph(ctx, v1.Graph{
			Functions: irv1.Functions{{Key: "f"}},
			Edges:     irv1.Edges{{Kind: irv1.EdgeKindContinuous}},
			Nodes: v1.Nodes{{
				Key:    "n1",
				Type:   "add",
				Config: msgpack.EncodedJSON{"rate": "10"},
			}},
		}))
		Expect(migrated.Functions).To(HaveLen(1))
		Expect(migrated.Edges).To(HaveLen(1))
		Expect(migrated.Edges[0].Key).ToNot(BeEmpty())
		Expect(migrated.Nodes).To(HaveLen(1))
		Expect(migrated.Inputs).To(HaveKey("n1"))
		Expect(migrated.Inputs["n1"]).To(HaveKeyWithValue("type", "add"))
		Expect(migrated.Inputs["n1"]).To(HaveKeyWithValue("rate", "10"))
	})

	It("Should carry edge endpoints and assign fresh edge keys", func(ctx SpecContext) {
		migrated := MustSucceed(v2.MigrateGraph(ctx, v1.Graph{
			Edges: irv1.Edges{{
				Source: irv1.Handle{Node: "a", Param: "out"},
				Target: irv1.Handle{Node: "b", Param: "in"},
				Kind:   irv1.EdgeKindContinuous,
			}},
		}))
		edge := migrated.Edges[0]
		Expect(edge.Source).To(Equal(ir.Handle{Node: "a", Param: "out"}))
		Expect(edge.Target).To(Equal(ir.Handle{Node: "b", Param: "in"}))
		Expect(edge.Kind).To(Equal(ir.EdgeKindContinuous))
		Expect(edge.Key).ToNot(BeEmpty())
	})

	It("Should carry node keys and positions", func(ctx SpecContext) {
		migrated := MustSucceed(v2.MigrateGraph(ctx, v1.Graph{
			Nodes: v1.Nodes{{Key: "n1", Position: spatial.XY{X: 1, Y: 2}}},
		}))
		node := migrated.Nodes[0]
		Expect(node.Key).To(Equal("n1"))
		Expect(node.Position).To(Equal(spatial.XY{X: 1, Y: 2}))
	})
})
