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
	types "github.com/synnaxlabs/arc/types/versions/v0"
	"github.com/synnaxlabs/x/encoding/orc"
)

var _ = Describe("Codec", func() {
	Describe("IR", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v1.IR) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v1.IR
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v1.IR{
				Functions: v1.Functions{{
					Key:  "double",
					Body: v1.Body{Raw: "output = input * 2"},
					Inputs: types.Params{
						{Name: "input", Type: types.Type{Kind: types.KindF64}},
					},
				}},
				Nodes: v1.Nodes{{Key: "double_1", Type: "double"}},
				Edges: v1.Edges{{
					Source: v1.Handle{Node: "double_1", Param: "output"},
					Target: v1.Handle{Node: "valve_1", Param: "input"},
					Kind:   v1.EdgeKindContinuous,
				}},
				Authorities: v1.Authorities{
					Default:  new(uint8(1)),
					Channels: map[uint32]uint8{3: 255},
				},
				Root: v1.Scope{
					Key:      "root",
					Mode:     v1.ScopeModeParallel,
					Liveness: v1.LivenessAlways,
					Strata: []v1.Members{
						{{NodeKey: new(string("double_1"))}},
					},
				},
			}),
			Entry("zero value", v1.IR{}),
		)
	})

	Describe("Member", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v1.Member) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v1.Member
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("node member", v1.Member{NodeKey: new(string("double_1"))}),
			Entry("scope member", v1.Member{
				Scope: new(v1.Scope{
					Key:      "nested",
					Mode:     v1.ScopeModeSequential,
					Liveness: v1.LivenessGated,
				}),
			}),
			Entry("zero value", v1.Member{}),
		)
	})

	Describe("Scope", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v1.Scope) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v1.Scope
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("parallel scope with strata", v1.Scope{
				Key:      "root",
				Mode:     v1.ScopeModeParallel,
				Liveness: v1.LivenessAlways,
				Strata: []v1.Members{
					{{NodeKey: new(string("double_1"))}},
					{{NodeKey: new(string("valve_1"))}},
				},
			}),
			Entry("sequential gated scope with steps and transitions", v1.Scope{
				Key:        "startup",
				Mode:       v1.ScopeModeSequential,
				Liveness:   v1.LivenessGated,
				Activation: new(v1.Handle{Node: "start", Param: "output"}),
				Steps: v1.Members{
					{NodeKey: new(string("pressurize"))},
					{Scope: new(v1.Scope{Key: "vent"})},
				},
				Transitions: []v1.Transition{{
					On:        v1.Handle{Node: "pressurize", Param: "done"},
					TargetKey: new(string("vent")),
				}},
			}),
			Entry("zero value", v1.Scope{}),
		)
	})

	Describe("Transition", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v1.Transition) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v1.Transition
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("with target", v1.Transition{
				On:        v1.Handle{Node: "n1", Param: "done"},
				TargetKey: new(string("s2")),
			}),
			Entry("exit transition", v1.Transition{
				On: v1.Handle{Node: "n1", Param: "done"},
			}),
		)
	})
})
