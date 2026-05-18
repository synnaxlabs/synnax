// Copyright 2026 Synnax Labs, Inc.
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
	Describe("Strata", func() {
		It("Should layer node members across the Root scope's strata", func() {
			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Node("C").
				Strata([][]string{{"A", "B"}, {"C"}}).
				Build()

			Expect(prog.Root.Mode).To(Equal(ir.ScopeModeParallel))
			Expect(prog.Root.Liveness).To(Equal(ir.LivenessAlways))
			Expect(prog.Root.Strata).To(HaveLen(2))
			Expect(prog.Root.Strata[0]).To(HaveLen(2))
			Expect(prog.Root.Strata[0][0].NodeKey).ToNot(BeNil())
			Expect(*prog.Root.Strata[0][0].NodeKey).To(Equal("A"))
			Expect(*prog.Root.Strata[1][0].NodeKey).To(Equal("C"))
		})
	})

	Describe("Sequence", func() {
		It("Should append a sequential gated Scope with parallel child scopes", func() {
			prog := testutil.NewIRBuilder().
				Sequence("main", []testutil.ScopeSpec{
					{Key: "stage_a", Strata: [][]string{{"A", "B"}, {"C"}}},
					{Key: "stage_b", Strata: [][]string{{"D"}}},
				}).
				Build()

			Expect(prog.Root.Strata).To(HaveLen(1))
			members := prog.Root.Strata[0]
			Expect(members).To(HaveLen(1))
			Expect(members[0].Scope).ToNot(BeNil())

			main := members[0].Scope
			Expect(main.Key).To(Equal("main"))
			Expect(main.Mode).To(Equal(ir.ScopeModeSequential))
			Expect(main.Liveness).To(Equal(ir.LivenessGated))
			Expect(main.Steps).To(HaveLen(2))

			stageA := main.Steps[0].Scope
			Expect(stageA).ToNot(BeNil())
			Expect(stageA.Mode).To(Equal(ir.ScopeModeParallel))
			Expect(stageA.Strata).To(HaveLen(2))
			Expect(stageA.Strata[0]).To(HaveLen(2))
			Expect(*stageA.Strata[0][0].NodeKey).To(Equal("A"))

			stageB := main.Steps[1].Scope
			Expect(stageB.Strata).To(HaveLen(1))
			Expect(*stageB.Strata[0][0].NodeKey).To(Equal("D"))
		})

		It("Should accept sequential child scopes via Steps", func() {
			prog := testutil.NewIRBuilder().
				Sequence("main", []testutil.ScopeSpec{
					{Key: "flow_a", Steps: []string{"N1", "N2"}},
				}).
				Build()

			main := prog.Root.Strata[0][0].Scope
			Expect(main.Steps).To(HaveLen(1))

			flowA := main.Steps[0].Scope
			Expect(flowA).ToNot(BeNil())
			Expect(flowA.Mode).To(Equal(ir.ScopeModeSequential))
			Expect(flowA.Steps).To(HaveLen(2))
			Expect(*flowA.Steps[0].NodeKey).To(Equal("N1"))
			Expect(*flowA.Steps[1].NodeKey).To(Equal("N2"))
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
			Expect(prog.Edges[0].Kind).To(Equal(ir.EdgeKindContinuous))
			Expect(prog.Edges[0].Source).To(Equal(ir.Handle{Node: "A", Param: "out"}))
			Expect(prog.Edges[0].Target).To(Equal(ir.Handle{Node: "B", Param: "in"}))
		})

		It("Should create conditional edges with Conditional()", func() {
			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Conditional("A", "trigger", "B", "activate").
				Build()

			Expect(prog.Edges).To(HaveLen(1))
			Expect(prog.Edges[0].Kind).To(Equal(ir.EdgeKindConditional))
		})
	})
})
