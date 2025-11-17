// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	. "github.com/synnaxlabs/x/testutil"
)

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
