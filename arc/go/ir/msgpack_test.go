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
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("DecodeMsgpack", func() {
	Describe("Handle", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := ir.Handle{Node: "node1", Param: "input"}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded ir.Handle
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Node  string
				Param string
			}{Node: "node1", Param: "input"}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded ir.Handle
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Node).To(Equal("node1"))
			Expect(decoded.Param).To(Equal("input"))
		})
	})

	Describe("Edge", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := ir.Edge{
				Source: ir.Handle{Node: "a", Param: "out"},
				Target: ir.Handle{Node: "b", Param: "in"},
				Kind:   ir.EdgeKindContinuous,
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded ir.Edge
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Source ir.Handle
				Target ir.Handle
				Kind   ir.EdgeKind
			}{
				Source: ir.Handle{Node: "a", Param: "out"},
				Target: ir.Handle{Node: "b", Param: "in"},
				Kind:   ir.EdgeKindConditional,
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded ir.Edge
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Source.Node).To(Equal("a"))
			Expect(decoded.Target.Node).To(Equal("b"))
			Expect(decoded.Kind).To(Equal(ir.EdgeKindConditional))
		})
	})

	Describe("Function", func() {
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Key      string
				Body     ir.Body
				Config   types.Params
				Inputs   types.Params
				Outputs  types.Params
				Channels types.Channels
			}{
				Key:  "fn1",
				Body: ir.Body{Raw: "return 1"},
				Inputs: types.Params{
					{Name: "x", Type: types.Type{Kind: types.KindF64}},
				},
				Channels: types.Channels{
					Read:  map[uint32]string{1: "sensor"},
					Write: map[uint32]string{2: "output"},
				},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded ir.Function
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal("fn1"))
			Expect(decoded.Body.Raw).To(Equal("return 1"))
			Expect(decoded.Inputs).To(HaveLen(1))
			Expect(decoded.Channels.Read).To(HaveLen(1))
		})
	})

	Describe("Node", func() {
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Key      string
				Type     string
				Config   types.Params
				Inputs   types.Params
				Outputs  types.Params
				Channels types.Channels
			}{
				Key:  "node1",
				Type: "fn1",
				Config: types.Params{
					{Name: "rate", Type: types.Type{Kind: types.KindF32}},
				},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded ir.Node
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal("node1"))
			Expect(decoded.Type).To(Equal("fn1"))
			Expect(decoded.Config).To(HaveLen(1))
		})
	})

})
