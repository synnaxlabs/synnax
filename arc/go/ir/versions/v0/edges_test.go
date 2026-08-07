// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	v0 "github.com/synnaxlabs/arc/ir/versions/v0"
)

var _ = Describe("Edges", func() {
	var (
		aToAdd, bToAdd, addToOut v0.Edge
		edges                    v0.Edges
	)

	BeforeEach(func() {
		aToAdd = v0.Edge{
			Source: v0.Handle{Node: "input_a", Param: ir.DefaultOutputParam},
			Target: v0.Handle{Node: "add", Param: ir.LHSInputParam},
			Kind:   v0.EdgeKindContinuous,
		}
		bToAdd = v0.Edge{
			Source: v0.Handle{Node: "input_b", Param: ir.DefaultOutputParam},
			Target: v0.Handle{Node: "add", Param: ir.RHSInputParam},
			Kind:   v0.EdgeKindContinuous,
		}
		addToOut = v0.Edge{
			Source: v0.Handle{Node: "add", Param: ir.DefaultOutputParam},
			Target: v0.Handle{Node: "out", Param: ir.DefaultInputParam},
			Kind:   v0.EdgeKindConditional,
		}
		edges = v0.Edges{aToAdd, bToAdd, addToOut}
	})

	Describe("GetBySource", func() {
		It("Should get the edge with the given source handle", func() {
			Expect(edges.GetBySource(bToAdd.Source)).To(Equal(bToAdd))
		})

		It("Should panic for a non-existent source", func() {
			Expect(func() {
				edges.GetBySource(v0.Handle{Node: "missing", Param: "x"})
			}).To(Panic())
		})
	})

	Describe("GetByTarget", func() {
		It("Should get the edge with the given target handle", func() {
			Expect(edges.GetByTarget(addToOut.Target)).To(Equal(addToOut))
		})

		It("Should panic for a non-existent target", func() {
			Expect(func() {
				edges.GetByTarget(v0.Handle{Node: "missing", Param: "x"})
			}).To(Panic())
		})
	})

	Describe("FindBySource", func() {
		It("Should find the edge with the given source handle", func() {
			edge, found := edges.FindBySource(aToAdd.Source)
			Expect(found).To(BeTrue())
			Expect(edge).To(Equal(aToAdd))
		})

		It("Should return false for a non-existent source", func() {
			_, found := edges.FindBySource(v0.Handle{Node: "missing", Param: "x"})
			Expect(found).To(BeFalse())
		})
	})

	Describe("FindByTarget", func() {
		It("Should find the edge with the given target handle", func() {
			edge, found := edges.FindByTarget(bToAdd.Target)
			Expect(found).To(BeTrue())
			Expect(edge).To(Equal(bToAdd))
		})

		It("Should return false for a non-existent target", func() {
			_, found := edges.FindByTarget(v0.Handle{Node: "missing", Param: "x"})
			Expect(found).To(BeFalse())
		})
	})

	Describe("GetInputs", func() {
		It("Should return all edges targeting the given node", func() {
			Expect(edges.GetInputs("add")).To(ConsistOf(aToAdd, bToAdd))
		})

		It("Should return no edges for a node without inputs", func() {
			Expect(edges.GetInputs("input_a")).To(BeEmpty())
		})
	})

	Describe("GetOutputs", func() {
		It("Should return all edges sourced from the given node", func() {
			Expect(edges.GetOutputs("add")).To(ConsistOf(addToOut))
		})

		It("Should return no edges for a node without outputs", func() {
			Expect(edges.GetOutputs("out")).To(BeEmpty())
		})
	})

	Describe("GetByKind", func() {
		It("Should return all edges with the given kind", func() {
			Expect(edges.GetByKind(v0.EdgeKindContinuous)).To(
				ConsistOf(aToAdd, bToAdd))
			Expect(edges.GetByKind(v0.EdgeKindConditional)).To(ConsistOf(addToOut))
		})

		It("Should return no edges for an unused kind", func() {
			Expect(edges.GetByKind(v0.EdgeKindUnspecified)).To(BeEmpty())
		})
	})
})
