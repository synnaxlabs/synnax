// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	irv0 "github.com/synnaxlabs/arc/ir/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/v1"
	"github.com/synnaxlabs/x/spatial"
)

// nonZeroV0 builds a v0.Data with every model field populated so passthrough
// regressions in v1.Migrate surface.
func nonZeroV0() v0.Data {
	return v0.Data{
		Graph: v0.Graph{
			Nodes: []v0.Node{{Key: "n1", Position: spatial.XY{X: 3, Y: 4}}},
			Edges: []v0.Edge{{
				Key:          "e1",
				Source:       "n1",
				Target:       "n2",
				SourceHandle: new("out"),
				TargetHandle: new("in"),
			}},
			Props: map[string]map[string]any{"n1": {"key": "stl.on"}},
		},
		Text: v0.Text{Raw: "chan a = 1"},
		Mode: "graph",
	}
}

var _ = Describe("Migrate", func() {
	It("Should nest the flat ReactFlow handles into Handle objects", func() {
		Expect(v1.Migrate(nonZeroV0()).Graph.Edges).To(ConsistOf(v1.Edge{
			Key:    "e1",
			Source: irv0.Handle{Node: "n1", Param: "out"},
			Target: irv0.Handle{Node: "n2", Param: "in"},
		}))
	})

	It("Should read an absent handle as an empty param", func() {
		in := nonZeroV0()
		in.Graph.Edges[0].SourceHandle, in.Graph.Edges[0].TargetHandle = nil, nil

		edge := v1.Migrate(in).Graph.Edges[0]

		Expect(edge.Source).To(Equal(irv0.Handle{Node: "n1"}))
		Expect(edge.Target).To(Equal(irv0.Handle{Node: "n2"}))
	})

	It("Should pass every other v0 model field through unchanged", func() {
		in := nonZeroV0()

		out := v1.Migrate(in)

		Expect(out.Graph.Nodes).To(Equal(in.Graph.Nodes))
		Expect(out.Graph.Props).To(Equal(in.Graph.Props))
		Expect(out.Text).To(Equal(in.Text))
		Expect(out.Mode).To(Equal(in.Mode))
	})
})
