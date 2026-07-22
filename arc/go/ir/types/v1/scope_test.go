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
)

var _ = Describe("Scope", func() {
	Describe("IsZero", func() {
		It("Should return true for an uninitialized scope", func() {
			Expect(v1.Scope{}.IsZero()).To(BeTrue())
		})

		It("Should return false when a stratum carries members", func() {
			s := v1.Scope{
				Mode:     v1.ScopeModeParallel,
				Liveness: v1.LivenessAlways,
				Strata:   []v1.Members{{{NodeKey: new(string("n1"))}}},
			}
			Expect(s.IsZero()).To(BeFalse())
		})

		It("Should return false when a sequential scope carries steps", func() {
			s := v1.Scope{
				Key:      "main",
				Mode:     v1.ScopeModeSequential,
				Liveness: v1.LivenessGated,
				Steps:    v1.Members{{NodeKey: new(string("n1"))}},
			}
			Expect(s.IsZero()).To(BeFalse())
		})
	})

	Describe("String", func() {
		It("Should render a parallel scope's strata as a tree", func() {
			s := v1.Scope{
				Key:      "root",
				Mode:     v1.ScopeModeParallel,
				Liveness: v1.LivenessAlways,
				Strata: []v1.Members{
					{{NodeKey: new(string("n1"))}},
					{{NodeKey: new(string("n2"))}},
				},
			}
			out := s.String()
			Expect(out).To(HavePrefix("root [ScopeModeParallel, LivenessAlways]\n"))
			Expect(out).To(ContainSubstring("stratum 0"))
			Expect(out).To(ContainSubstring("stratum 1"))
			Expect(out).To(ContainSubstring("n1"))
			Expect(out).To(ContainSubstring("n2"))
		})

		It("Should render a sequential scope's steps and transitions", func() {
			s := v1.Scope{
				Key:      "startup",
				Mode:     v1.ScopeModeSequential,
				Liveness: v1.LivenessGated,
				Steps: v1.Members{
					{NodeKey: new(string("pressurize"))},
					{NodeKey: new(string("vent"))},
				},
				Transitions: []v1.Transition{{
					On:        v1.Handle{Node: "pressurize", Param: "done"},
					TargetKey: new(string("vent")),
				}},
			}
			out := s.String()
			Expect(out).To(HavePrefix("startup [ScopeModeSequential, LivenessGated]\n"))
			Expect(out).To(ContainSubstring("pressurize"))
			Expect(out).To(ContainSubstring("vent"))
		})

		It("Should label a keyless scope as (scope)", func() {
			Expect(v1.Scope{}.String()).To(HavePrefix("(scope) "))
		})
	})

	Describe("StringWithPrefix", func() {
		It("Should indent nested lines by the given prefix", func() {
			s := v1.Scope{
				Key:      "root",
				Mode:     v1.ScopeModeParallel,
				Liveness: v1.LivenessAlways,
				Strata:   []v1.Members{{{NodeKey: new(string("n1"))}}},
			}
			out := s.StringWithPrefix("    ")
			Expect(out).To(ContainSubstring("\n    └── stratum 0"))
		})
	})
})
