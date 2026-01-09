// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("EdgeKind", func() {
	Describe("Constants", func() {
		It("Should have Continuous as zero value", func() {
			var kind ir.EdgeKind
			Expect(kind).To(Equal(ir.EdgeKindContinuous))
		})

		It("Should distinguish Continuous and EdgeKindOneShot", func() {
			Expect(ir.EdgeKindContinuous).ToNot(Equal(ir.EdgeKindOneShot))
		})

		It("Should have Continuous = 0 and EdgeKindOneShot = 1", func() {
			Expect(int(ir.EdgeKindContinuous)).To(Equal(0))
			Expect(int(ir.EdgeKindOneShot)).To(Equal(1))
		})
	})

	Describe("JSON Serialization", func() {
		It("Should marshal Continuous as 0", func() {
			edge := ir.Edge{
				Source: ir.Handle{Node: "a", Param: "out"},
				Target: ir.Handle{Node: "b", Param: "in"},
				Kind:   ir.EdgeKindContinuous,
			}
			data := MustSucceed(json.Marshal(edge))
			Expect(string(data)).To(ContainSubstring(`"kind":0`))
		})

		It("Should marshal EdgeKindOneShot as 1", func() {
			edge := ir.Edge{
				Source: ir.Handle{Node: "a", Param: "out"},
				Target: ir.Handle{Node: "b", Param: "in"},
				Kind:   ir.EdgeKindOneShot,
			}
			data := MustSucceed(json.Marshal(edge))
			Expect(string(data)).To(ContainSubstring(`"kind":1`))
		})

		It("Should unmarshal Continuous edge", func() {
			data := []byte(`{"source":{"node":"a","param":"out"},"target":{"node":"b","param":"in"},"kind":0}`)
			var edge ir.Edge
			Expect(json.Unmarshal(data, &edge)).To(Succeed())
			Expect(edge.Kind).To(Equal(ir.EdgeKindContinuous))
		})

		It("Should unmarshal EdgeKindOneShot edge", func() {
			data := []byte(`{"source":{"node":"cond","param":"output"},"target":{"node":"stage_entry","param":"activate"},"kind":1}`)
			var edge ir.Edge
			Expect(json.Unmarshal(data, &edge)).To(Succeed())
			Expect(edge.Kind).To(Equal(ir.EdgeKindOneShot))
		})

		It("Should default to Continuous when kind is omitted", func() {
			data := []byte(`{"source":{"node":"a","param":"out"},"target":{"node":"b","param":"in"}}`)
			var edge ir.Edge
			Expect(json.Unmarshal(data, &edge)).To(Succeed())
			Expect(edge.Kind).To(Equal(ir.EdgeKindContinuous))
		})
	})
})

var _ = Describe("Edges", func() {
	var (
		edge1, edge2, edge3 ir.Edge
		edges               ir.Edges
	)

	BeforeEach(func() {
		edge1 = ir.Edge{
			Source: ir.Handle{Node: "node1", Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: "node2", Param: "a"},
		}
		edge2 = ir.Edge{
			Source: ir.Handle{Node: "node1", Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: "node3", Param: "b"},
		}
		edge3 = ir.Edge{
			Source: ir.Handle{Node: "node2", Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: "node4", Param: "input"},
		}
		edges = ir.Edges{edge1, edge2, edge3}
	})

	Describe("FindBySource", func() {
		It("Should find edge by source handle", func() {
			handle := ir.Handle{Node: "node2", Param: ir.DefaultOutputParam}
			edge := MustBeOk(edges.FindBySource(handle))
			Expect(edge.Source.Node).To(Equal("node2"))
			Expect(edge.Target.Node).To(Equal("node4"))
		})

		It("Should return false for non-existent source", func() {
			handle := ir.Handle{Node: "nonexistent", Param: ir.DefaultOutputParam}
			_, found := edges.FindBySource(handle)
			Expect(found).To(BeFalse())
		})
	})

	Describe("FindByTarget", func() {
		It("Should find edge by target handle", func() {
			handle := ir.Handle{Node: "node3", Param: "b"}
			edge := MustBeOk(edges.FindByTarget(handle))
			Expect(edge.Source.Node).To(Equal("node1"))
			Expect(edge.Target.Node).To(Equal("node3"))
		})

		It("Should return false for non-existent target", func() {
			handle := ir.Handle{Node: "nonexistent", Param: "x"}
			_, found := edges.FindByTarget(handle)
			Expect(found).To(BeFalse())
		})
	})

	Describe("GetBySource", func() {
		It("Should get edge by source handle", func() {
			handle := ir.Handle{Node: "node1", Param: ir.DefaultOutputParam}
			edge := edges.GetBySource(handle)
			Expect(edge.Source.Node).To(Equal("node1"))
		})

		It("Should panic for non-existent source", func() {
			handle := ir.Handle{Node: "nonexistent", Param: ir.DefaultOutputParam}
			Expect(func() {
				_ = edges.GetBySource(handle)
			}).To(Panic())
		})
	})

	Describe("GetByTarget", func() {
		It("Should get edge by target handle", func() {
			handle := ir.Handle{Node: "node2", Param: "a"}
			edge := edges.GetByTarget(handle)
			Expect(edge.Target.Node).To(Equal("node2"))
		})

		It("Should panic for non-existent target", func() {
			handle := ir.Handle{Node: "nonexistent", Param: "x"}
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
			Expect(inputs).To(HaveLen(0))
		})

		It("Should return multiple edges for multi-input node", func() {
			edge4 := ir.Edge{
				Source: ir.Handle{Node: "node5", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "node3", Param: "a"},
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
			Expect(outputs).To(HaveLen(0))
		})

		It("Should return single edge for single-output node", func() {
			outputs := edges.GetOutputs("node2")
			Expect(outputs).To(HaveLen(1))
			Expect(outputs[0].Target.Node).To(Equal("node4"))
		})
	})

	Describe("GetByKind", func() {
		var mixedEdges ir.Edges

		BeforeEach(func() {
			mixedEdges = ir.Edges{
				{
					Source: ir.Handle{Node: "timer", Param: "output"},
					Target: ir.Handle{Node: "controller", Param: "input"},
					Kind:   ir.EdgeKindContinuous,
				},
				{
					Source: ir.Handle{Node: "condition", Param: "output"},
					Target: ir.Handle{Node: "stage_entry", Param: "activate"},
					Kind:   ir.EdgeKindOneShot,
				},
				{
					Source: ir.Handle{Node: "sensor", Param: "output"},
					Target: ir.Handle{Node: "filter", Param: "input"},
					Kind:   ir.EdgeKindContinuous,
				},
				{
					Source: ir.Handle{Node: "timeout", Param: "output"},
					Target: ir.Handle{Node: "abort_entry", Param: "activate"},
					Kind:   ir.EdgeKindOneShot,
				},
				{
					Source: ir.Handle{Node: "pid", Param: "output"},
					Target: ir.Handle{Node: "actuator", Param: "input"},
					Kind:   ir.EdgeKindContinuous,
				},
			}
		})

		It("Should filter Continuous edges", func() {
			continuous := mixedEdges.GetByKind(ir.EdgeKindContinuous)
			Expect(continuous).To(HaveLen(3))
			for _, e := range continuous {
				Expect(e.Kind).To(Equal(ir.EdgeKindContinuous))
			}
		})

		It("Should filter EdgeKindOneShot edges", func() {
			EdgeKindOneShot := mixedEdges.GetByKind(ir.EdgeKindOneShot)
			Expect(EdgeKindOneShot).To(HaveLen(2))
			for _, e := range EdgeKindOneShot {
				Expect(e.Kind).To(Equal(ir.EdgeKindOneShot))
			}
		})

		It("Should return empty for no matches", func() {
			allContinuous := ir.Edges{
				{Kind: ir.EdgeKindContinuous},
				{Kind: ir.EdgeKindContinuous},
				{Kind: ir.EdgeKindContinuous},
			}
			Expect(allContinuous.GetByKind(ir.EdgeKindOneShot)).To(BeEmpty())
		})

		It("Should return empty from empty collection", func() {
			empty := ir.Edges{}
			Expect(empty.GetByKind(ir.EdgeKindContinuous)).To(BeEmpty())
			Expect(empty.GetByKind(ir.EdgeKindOneShot)).To(BeEmpty())
		})

		It("Should preserve source and target handles when filtering", func() {
			EdgeKindOneShot := mixedEdges.GetByKind(ir.EdgeKindOneShot)
			Expect(EdgeKindOneShot).To(HaveLen(2))
			// Verify first EdgeKindOneShot edge
			Expect(EdgeKindOneShot[0].Source.Node).To(Equal("condition"))
			Expect(EdgeKindOneShot[0].Target.Node).To(Equal("stage_entry"))
			// Verify second EdgeKindOneShot edge
			Expect(EdgeKindOneShot[1].Source.Node).To(Equal("timeout"))
			Expect(EdgeKindOneShot[1].Target.Node).To(Equal("abort_entry"))
		})

		It("Should return edges in original order", func() {
			continuous := mixedEdges.GetByKind(ir.EdgeKindContinuous)
			Expect(continuous[0].Source.Node).To(Equal("timer"))
			Expect(continuous[1].Source.Node).To(Equal("sensor"))
			Expect(continuous[2].Source.Node).To(Equal("pid"))
		})
	})

	Describe("Empty Collection", func() {
		It("Should handle FindBySource on empty collection", func() {
			empty := ir.Edges{}
			handle := ir.Handle{Node: "node1", Param: ir.DefaultOutputParam}
			_, found := empty.FindBySource(handle)
			Expect(found).To(BeFalse())
		})

		It("Should handle FindByTarget on empty collection", func() {
			empty := ir.Edges{}
			handle := ir.Handle{Node: "node1", Param: "input"}
			_, found := empty.FindByTarget(handle)
			Expect(found).To(BeFalse())
		})

		It("Should panic on GetBySource with empty collection", func() {
			empty := ir.Edges{}
			handle := ir.Handle{Node: "node1", Param: ir.DefaultOutputParam}
			Expect(func() {
				_ = empty.GetBySource(handle)
			}).To(Panic())
		})

		It("Should panic on GetByTarget with empty collection", func() {
			empty := ir.Edges{}
			handle := ir.Handle{Node: "node1", Param: "input"}
			Expect(func() {
				_ = empty.GetByTarget(handle)
			}).To(Panic())
		})

		It("Should return empty slice for GetInputs on empty collection", func() {
			empty := ir.Edges{}
			inputs := empty.GetInputs("node1")
			Expect(inputs).To(HaveLen(0))
		})

		It("Should return empty slice for GetOutputs on empty collection", func() {
			empty := ir.Edges{}
			outputs := empty.GetOutputs("node1")
			Expect(outputs).To(HaveLen(0))
		})
	})
})
