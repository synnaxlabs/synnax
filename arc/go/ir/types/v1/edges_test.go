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
	v1 "github.com/synnaxlabs/arc/ir/types/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Edges", func() {
	var (
		edge1, edge2, edge3 v1.Edge
		edges               v1.Edges
	)

	BeforeEach(func() {
		edge1 = v1.Edge{
			Source: v1.Handle{Node: "node1", Param: "output"},
			Target: v1.Handle{Node: "node2", Param: "a"},
		}
		edge2 = v1.Edge{
			Source: v1.Handle{Node: "node1", Param: "output"},
			Target: v1.Handle{Node: "node3", Param: "b"},
		}
		edge3 = v1.Edge{
			Source: v1.Handle{Node: "node2", Param: "output"},
			Target: v1.Handle{Node: "node4", Param: "input"},
		}
		edges = v1.Edges{edge1, edge2, edge3}
	})

	Describe("FindBySource", func() {
		It("Should find edge by source handle", func() {
			handle := v1.Handle{Node: "node2", Param: "output"}
			edge := MustBeOk(edges.FindBySource(handle))
			Expect(edge.Source.Node).To(Equal("node2"))
			Expect(edge.Target.Node).To(Equal("node4"))
		})

		It("Should return false for non-existent source", func() {
			handle := v1.Handle{Node: "nonexistent", Param: "output"}
			_, found := edges.FindBySource(handle)
			Expect(found).To(BeFalse())
		})
	})

	Describe("FindByTarget", func() {
		It("Should find edge by target handle", func() {
			handle := v1.Handle{Node: "node3", Param: "b"}
			edge := MustBeOk(edges.FindByTarget(handle))
			Expect(edge.Source.Node).To(Equal("node1"))
			Expect(edge.Target.Node).To(Equal("node3"))
		})

		It("Should return false for non-existent target", func() {
			handle := v1.Handle{Node: "nonexistent", Param: "x"}
			_, found := edges.FindByTarget(handle)
			Expect(found).To(BeFalse())
		})
	})

	Describe("GetBySource", func() {
		It("Should get edge by source handle", func() {
			handle := v1.Handle{Node: "node1", Param: "output"}
			edge := edges.GetBySource(handle)
			Expect(edge.Source.Node).To(Equal("node1"))
		})

		It("Should panic for non-existent source", func() {
			handle := v1.Handle{Node: "nonexistent", Param: "output"}
			Expect(func() {
				_ = edges.GetBySource(handle)
			}).To(Panic())
		})
	})

	Describe("GetByTarget", func() {
		It("Should get edge by target handle", func() {
			handle := v1.Handle{Node: "node2", Param: "a"}
			edge := edges.GetByTarget(handle)
			Expect(edge.Target.Node).To(Equal("node2"))
		})

		It("Should panic for non-existent target", func() {
			handle := v1.Handle{Node: "nonexistent", Param: "x"}
			Expect(func() {
				_ = edges.GetByTarget(handle)
			}).To(Panic())
		})
	})

	Describe("GetInputs", func() {
		It("Should return all edges targeting a node", func() {
			inputs := edges.GetInputs("node2")
			Expect(inputs).To(HaveLen(1))
			Expect(inputs[0].Target.Node).To(Equal("node2"))
			Expect(inputs[0].Source.Node).To(Equal("node1"))
		})

		It("Should return empty slice for node with no inputs", func() {
			inputs := edges.GetInputs("node1")
			Expect(inputs).To(BeEmpty())
		})

		It("Should return multiple edges for multi-input node", func() {
			edge4 := v1.Edge{
				Source: v1.Handle{Node: "node5", Param: "output"},
				Target: v1.Handle{Node: "node3", Param: "a"},
			}
			edges = append(edges, edge4)
			inputs := edges.GetInputs("node3")
			Expect(inputs).To(HaveLen(2))
		})
	})

	Describe("GetOutputs", func() {
		It("Should return all edges sourced from a node", func() {
			outputs := edges.GetOutputs("node1")
			Expect(outputs).To(HaveLen(2))
			Expect(outputs[0].Source.Node).To(Equal("node1"))
			Expect(outputs[1].Source.Node).To(Equal("node1"))
		})

		It("Should return empty slice for node with no outputs", func() {
			outputs := edges.GetOutputs("node4")
			Expect(outputs).To(BeEmpty())
		})

		It("Should return single edge for single-output node", func() {
			outputs := edges.GetOutputs("node2")
			Expect(outputs).To(HaveLen(1))
			Expect(outputs[0].Target.Node).To(Equal("node4"))
		})
	})

	Describe("GetByKind", func() {
		var mixedEdges v1.Edges

		BeforeEach(func() {
			mixedEdges = v1.Edges{
				{
					Source: v1.Handle{Node: "timer", Param: "output"},
					Target: v1.Handle{Node: "controller", Param: "input"},
					Kind:   v1.EdgeKindContinuous,
				},
				{
					Source: v1.Handle{Node: "condition", Param: "output"},
					Target: v1.Handle{Node: "stage_entry", Param: "activate"},
					Kind:   v1.EdgeKindConditional,
				},
				{
					Source: v1.Handle{Node: "sensor", Param: "output"},
					Target: v1.Handle{Node: "filter", Param: "input"},
					Kind:   v1.EdgeKindContinuous,
				},
				{
					Source: v1.Handle{Node: "timeout", Param: "output"},
					Target: v1.Handle{Node: "abort_entry", Param: "activate"},
					Kind:   v1.EdgeKindConditional,
				},
				{
					Source: v1.Handle{Node: "pid", Param: "output"},
					Target: v1.Handle{Node: "actuator", Param: "input"},
					Kind:   v1.EdgeKindContinuous,
				},
			}
		})

		It("Should filter Continuous edges", func() {
			continuous := mixedEdges.GetByKind(v1.EdgeKindContinuous)
			Expect(continuous).To(HaveLen(3))
			for _, e := range continuous {
				Expect(e.Kind).To(Equal(v1.EdgeKindContinuous))
			}
		})

		It("Should filter Conditional edges", func() {
			conditional := mixedEdges.GetByKind(v1.EdgeKindConditional)
			Expect(conditional).To(HaveLen(2))
			for _, e := range conditional {
				Expect(e.Kind).To(Equal(v1.EdgeKindConditional))
			}
		})

		It("Should return empty for no matches", func() {
			allContinuous := v1.Edges{
				{Kind: v1.EdgeKindContinuous},
				{Kind: v1.EdgeKindContinuous},
				{Kind: v1.EdgeKindContinuous},
			}
			Expect(allContinuous.GetByKind(v1.EdgeKindConditional)).To(BeEmpty())
		})

		It("Should return empty from empty collection", func() {
			empty := v1.Edges{}
			Expect(empty.GetByKind(v1.EdgeKindContinuous)).To(BeEmpty())
			Expect(empty.GetByKind(v1.EdgeKindConditional)).To(BeEmpty())
		})

		It("Should preserve source and target handles when filtering", func() {
			conditional := mixedEdges.GetByKind(v1.EdgeKindConditional)
			Expect(conditional).To(HaveLen(2))
			// Verify first Conditional edge
			Expect(conditional[0].Source.Node).To(Equal("condition"))
			Expect(conditional[0].Target.Node).To(Equal("stage_entry"))
			// Verify second Conditional edge
			Expect(conditional[1].Source.Node).To(Equal("timeout"))
			Expect(conditional[1].Target.Node).To(Equal("abort_entry"))
		})

		It("Should return edges in original order", func() {
			continuous := mixedEdges.GetByKind(v1.EdgeKindContinuous)
			Expect(continuous[0].Source.Node).To(Equal("timer"))
			Expect(continuous[1].Source.Node).To(Equal("sensor"))
			Expect(continuous[2].Source.Node).To(Equal("pid"))
		})
	})

	Describe("Empty Collection", func() {
		It("Should handle FindBySource on empty collection", func() {
			empty := v1.Edges{}
			handle := v1.Handle{Node: "node1", Param: "output"}
			_, found := empty.FindBySource(handle)
			Expect(found).To(BeFalse())
		})

		It("Should handle FindByTarget on empty collection", func() {
			empty := v1.Edges{}
			handle := v1.Handle{Node: "node1", Param: "input"}
			_, found := empty.FindByTarget(handle)
			Expect(found).To(BeFalse())
		})

		It("Should panic on GetBySource with empty collection", func() {
			empty := v1.Edges{}
			handle := v1.Handle{Node: "node1", Param: "output"}
			Expect(func() {
				_ = empty.GetBySource(handle)
			}).To(Panic())
		})

		It("Should panic on GetByTarget with empty collection", func() {
			empty := v1.Edges{}
			handle := v1.Handle{Node: "node1", Param: "input"}
			Expect(func() {
				_ = empty.GetByTarget(handle)
			}).To(Panic())
		})

		It("Should return empty slice for GetInputs on empty collection", func() {
			empty := v1.Edges{}
			inputs := empty.GetInputs("node1")
			Expect(inputs).To(BeEmpty())
		})

		It("Should return empty slice for GetOutputs on empty collection", func() {
			empty := v1.Edges{}
			outputs := empty.GetOutputs("node1")
			Expect(outputs).To(BeEmpty())
		})
	})
})
