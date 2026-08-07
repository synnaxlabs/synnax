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
	v1 "github.com/synnaxlabs/arc/ir/versions/v1"
)

var _ = Describe("Scope", func() {
	Describe("IsZero", func() {
		DescribeTable(
			"Classification",
			func(s v1.Scope, expected bool) {
				Expect(s.IsZero()).To(Equal(expected))
			},
			Entry("an empty scope", v1.Scope{}, true),
			Entry("a key", v1.Scope{Key: "main"}, false),
			Entry("a mode", v1.Scope{Mode: v1.ScopeModeParallel}, false),
			Entry("a liveness", v1.Scope{Liveness: v1.LivenessAlways}, false),
			Entry("an activation",
				v1.Scope{Activation: &v1.Handle{Node: "n", Param: "p"}}, false),
			Entry("strata",
				v1.Scope{Strata: []v1.Members{{v1.NodeMember("n")}}}, false),
			Entry("steps", v1.Scope{Steps: v1.Members{v1.NodeMember("n")}}, false),
			Entry("transitions",
				v1.Scope{Transitions: []v1.Transition{{}}}, false),
		)
	})

	Describe("String", func() {
		It("Should render parallel scope with named stratum entries", func() {
			s := v1.Scope{
				Key:      "root",
				Mode:     v1.ScopeModeParallel,
				Liveness: v1.LivenessAlways,
				Strata: []v1.Members{
					{v1.NodeMember("a"), v1.NodeMember("b")},
					{v1.NodeMember("c")},
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
			s := v1.Scope{
				Mode:     v1.ScopeModeParallel,
				Liveness: v1.LivenessAlways,
				Strata:   []v1.Members{{v1.NodeMember("x")}},
			}
			Expect(s.String()).To(ContainSubstring("(scope)"))
		})

		It("Should render sequential scope steps and transitions", func() {
			run := "run"
			s := v1.Scope{
				Key:      "main",
				Mode:     v1.ScopeModeSequential,
				Liveness: v1.LivenessGated,
				Steps:    v1.Members{v1.NodeMember("init"), v1.NodeMember("run")},
				Transitions: []v1.Transition{
					{On: v1.Handle{Node: "init", Param: "done"}, TargetKey: &run},
					{On: v1.Handle{Node: "run", Param: "done"}},
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
			inner := v1.Scope{
				Key:      "inner",
				Mode:     v1.ScopeModeSequential,
				Liveness: v1.LivenessGated,
				Steps:    v1.Members{v1.NodeMember("step1")},
			}
			outer := v1.Scope{
				Key:      "outer",
				Mode:     v1.ScopeModeParallel,
				Liveness: v1.LivenessAlways,
				Strata:   []v1.Members{{v1.ScopeMember(inner)}},
			}
			out := outer.String()
			Expect(out).To(ContainSubstring("outer"))
			Expect(out).To(ContainSubstring("inner"))
			Expect(out).To(ContainSubstring("step1"))
		})
	})
})
