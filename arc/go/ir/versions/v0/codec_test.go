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
	v0 "github.com/synnaxlabs/arc/ir/versions/v0"
	types "github.com/synnaxlabs/arc/types/versions/v0"
	"github.com/synnaxlabs/x/encoding/orc"
)

var _ = Describe("Codec", func() {
	Describe("Handle", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Handle) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Handle
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Handle{Node: "sensor", Param: "output"}),
			Entry("zero value", v0.Handle{}),
		)
	})

	Describe("Edge", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Edge) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Edge
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Edge{
				Source: v0.Handle{Node: "sensor", Param: "output"},
				Target: v0.Handle{Node: "valve", Param: "command"},
				Kind:   v0.EdgeKindContinuous,
			}),
			Entry("zero value", v0.Edge{}),
		)
	})

	Describe("Body", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Body) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Body
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Body{Raw: "output = input * 2"}),
			Entry("zero value", v0.Body{}),
		)
	})

	Describe("Function", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Function) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Function
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Function{
				Key:  "double",
				Body: v0.Body{Raw: "output = input * 2"},
				Config: types.Params{
					{Name: "rate", Type: types.Type{Kind: types.KindF32}},
				},
				Inputs: types.Params{
					{Name: "input", Type: types.Type{Kind: types.KindF64}},
				},
				Outputs: types.Params{
					{Name: "output", Type: types.Type{Kind: types.KindF64}},
				},
				Channels: types.Channels{
					Read:  map[uint32]string{1: "sensor"},
					Write: map[uint32]string{2: "valve"},
				},
			}),
			Entry("zero value", v0.Function{}),
		)
	})

	Describe("Node", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Node) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Node
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Node{
				Key:  "double_1",
				Type: "double",
				Config: types.Params{
					{Name: "rate", Type: types.Type{Kind: types.KindF32}, Value: 2.5},
				},
				Inputs: types.Params{
					{Name: "input", Type: types.Type{Kind: types.KindF64}},
				},
				Outputs: types.Params{
					{Name: "output", Type: types.Type{Kind: types.KindF64}},
				},
				Channels: types.Channels{
					Read:  map[uint32]string{1: "sensor"},
					Write: map[uint32]string{2: "valve"},
				},
			}),
			Entry("zero value", v0.Node{}),
		)
	})

	Describe("Stage", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Stage) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Stage
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Stage{
				Key:    "pressurize",
				Nodes:  []string{"double_1", "valve_1"},
				Strata: v0.Strata{{"double_1"}, {"valve_1"}},
			}),
			Entry("zero value", v0.Stage{}),
		)
	})

	Describe("Sequence", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Sequence) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Sequence
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Sequence{
				Key: "startup",
				Stages: []v0.Stage{
					{Key: "pressurize", Nodes: []string{"double_1"}},
					{Key: "vent", Nodes: []string{"valve_1"}},
				},
			}),
			Entry("zero value", v0.Sequence{}),
		)
	})

	Describe("Authorities", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.Authorities) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.Authorities
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.Authorities{
				Default:  new(uint8(42)),
				Channels: map[uint32]uint8{1: 100, 2: 200},
			}),
			Entry("zero value", v0.Authorities{}),
		)
	})

	Describe("IR", func() {
		DescribeTable("should round-trip encode and decode",
			func(original v0.IR) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded v0.IR
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			Entry("fully populated", v0.IR{
				Functions: v0.Functions{{
					Key:  "double",
					Body: v0.Body{Raw: "output = input * 2"},
					Inputs: types.Params{
						{Name: "input", Type: types.Type{Kind: types.KindF64}},
					},
				}},
				Nodes: v0.Nodes{{Key: "double_1", Type: "double"}},
				Edges: v0.Edges{{
					Source: v0.Handle{Node: "double_1", Param: "output"},
					Target: v0.Handle{Node: "valve_1", Param: "input"},
					Kind:   v0.EdgeKindContinuous,
				}},
				Strata: v0.Strata{{"double_1"}, {"valve_1"}},
				Sequences: v0.Sequences{{
					Key:    "startup",
					Stages: []v0.Stage{{Key: "pressurize"}},
				}},
				Authorities: v0.Authorities{
					Default:  new(uint8(1)),
					Channels: map[uint32]uint8{3: 255},
				},
			}),
			Entry("zero value", v0.IR{}),
		)
	})
})
