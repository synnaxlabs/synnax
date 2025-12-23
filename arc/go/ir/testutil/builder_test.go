// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/ir/testutil"
)

var _ = Describe("IRBuilder", func() {
	Describe("Sequence", func() {
		It("Should collect nodes from strata into stage nodes list", func() {
			prog := testutil.NewIRBuilder().
				Sequence("main", []testutil.StageSpec{
					{Key: "stage_a", Strata: ir.Strata{{"A", "B"}, {"C"}}},
				}).
				Build()

			Expect(prog.Sequences).To(HaveLen(1))
			Expect(prog.Sequences[0].Key).To(Equal("main"))
			Expect(prog.Sequences[0].Stages).To(HaveLen(1))

			stage := prog.Sequences[0].Stages[0]
			Expect(stage.Key).To(Equal("stage_a"))
			Expect(stage.Nodes).To(ConsistOf("A", "B", "C"))
			Expect(stage.Strata).To(Equal(ir.Strata{{"A", "B"}, {"C"}}))
		})

		It("Should handle multiple stages", func() {
			prog := testutil.NewIRBuilder().
				Sequence("seq", []testutil.StageSpec{
					{Key: "first", Strata: ir.Strata{{"X"}}},
					{Key: "second", Strata: ir.Strata{{"Y"}, {"Z"}}},
				}).
				Build()

			Expect(prog.Sequences[0].Stages).To(HaveLen(2))
			Expect(prog.Sequences[0].Stages[0].Nodes).To(Equal([]string{"X"}))
			Expect(prog.Sequences[0].Stages[1].Nodes).To(ConsistOf("Y", "Z"))
		})

		It("Should handle empty strata", func() {
			prog := testutil.NewIRBuilder().
				Sequence("empty", []testutil.StageSpec{
					{Key: "stage", Strata: nil},
				}).
				Build()

			Expect(prog.Sequences[0].Stages[0].Nodes).To(BeNil())
		})
	})

	Describe("Edge kinds", func() {
		It("Should create continuous edges with Edge()", func() {
			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Edge("A", "out", "B", "in").
				Build()

			Expect(prog.Edges).To(HaveLen(1))
			Expect(prog.Edges[0].Kind).To(Equal(ir.Continuous))
			Expect(prog.Edges[0].Source).To(Equal(ir.Handle{Node: "A", Param: "out"}))
			Expect(prog.Edges[0].Target).To(Equal(ir.Handle{Node: "B", Param: "in"}))
		})

		It("Should create one-shot edges with OneShot()", func() {
			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				OneShot("A", "trigger", "B", "activate").
				Build()

			Expect(prog.Edges).To(HaveLen(1))
			Expect(prog.Edges[0].Kind).To(Equal(ir.OneShot))
		})
	})
})
