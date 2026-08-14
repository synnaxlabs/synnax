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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
)

var _ = Describe("Scope", func() {
	Describe("IsZero", func() {
		DescribeTable(
			"Classification",
			func(s ir.Scope, expected bool) {
				Expect(s.IsZero()).To(Equal(expected))
			},
			Entry("an empty scope", ir.Scope{}, true),
			Entry("a key", ir.Scope{Key: "main"}, false),
			Entry("a mode", ir.Scope{Mode: ir.ScopeModeParallel}, false),
			Entry("a liveness", ir.Scope{Liveness: ir.LivenessAlways}, false),
			Entry("an activation",
				ir.Scope{Activation: &ir.Handle{Node: "n", Param: "p"}}, false),
			Entry("strata",
				ir.Scope{Strata: []ir.Members{{ir.NodeMember("n")}}}, false),
			Entry("steps", ir.Scope{Steps: ir.Members{ir.NodeMember("n")}}, false),
			Entry("transitions",
				ir.Scope{Transitions: []ir.Transition{{}}}, false),
		)
	})

	Describe("String", func() {
		It("Should render parallel scope with named stratum entries", func() {
			s := ir.Scope{
				Key:      "root",
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessAlways,
				Strata: []ir.Members{
					{ir.NodeMember("a"), ir.NodeMember("b")},
					{ir.NodeMember("c")},
				},
			}
			out := s.String()
			Expect(out).To(ContainSubstring("root"))
			Expect(out).To(ContainSubstring("stratum 0"))
			Expect(out).To(ContainSubstring("stratum 1"))
			Expect(out).To(ContainSubstring("a"))
			Expect(out).To(ContainSubstring("b"))
			Expect(out).To(ContainSubstring("c"))
		})

		It("Should render an unnamed scope with the (scope) placeholder", func() {
			s := ir.Scope{
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessAlways,
				Strata:   []ir.Members{{ir.NodeMember("x")}},
			}
			Expect(s.String()).To(ContainSubstring("(scope)"))
		})

		It("Should render sequential scope steps and transitions", func() {
			run := "run"
			s := ir.Scope{
				Key:      "main",
				Mode:     ir.ScopeModeSequential,
				Liveness: ir.LivenessGated,
				Steps:    ir.Members{ir.NodeMember("init"), ir.NodeMember("run")},
				Transitions: []ir.Transition{
					{On: ir.Handle{Node: "init", Param: "done"}, TargetKey: &run},
					{On: ir.Handle{Node: "run", Param: "done"}},
				},
			}
			out := s.String()
			Expect(out).To(ContainSubstring("main"))
			Expect(out).To(ContainSubstring("init"))
			Expect(out).To(ContainSubstring("run"))
			Expect(out).To(ContainSubstring("on init/done => run"))
			Expect(out).To(ContainSubstring("on run/done => exit"))
		})

		It("Should render nested scope members", func() {
			inner := ir.Scope{
				Key:      "inner",
				Mode:     ir.ScopeModeSequential,
				Liveness: ir.LivenessGated,
				Steps:    ir.Members{ir.NodeMember("step1")},
			}
			outer := ir.Scope{
				Key:      "outer",
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessAlways,
				Strata:   []ir.Members{{ir.ScopeMember(inner)}},
			}
			out := outer.String()
			Expect(out).To(ContainSubstring("outer"))
			Expect(out).To(ContainSubstring("inner"))
			Expect(out).To(ContainSubstring("step1"))
		})
	})
})
