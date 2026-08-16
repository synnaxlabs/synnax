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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/arc/ir/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("EdgeKind", func() {
	Describe("Constants", func() {
		It("Should distinguish Continuous and Conditional", func() {
			Expect(v0.EdgeKindContinuous).ToNot(Equal(v0.EdgeKindConditional))
		})

		It("Should have Continuous = 1 and Conditional = 2 to match protobuf", func() {
			Expect(int(v0.EdgeKindContinuous)).To(Equal(1))
			Expect(int(v0.EdgeKindConditional)).To(Equal(2))
		})
	})

	Describe("JSON Serialization", func() {
		It("Should marshal Continuous as 1", func() {
			edge := v0.Edge{
				Source: v0.Handle{Node: "a", Param: "out"},
				Target: v0.Handle{Node: "b", Param: "in"},
				Kind:   v0.EdgeKindContinuous,
			}
			data := MustSucceed(json.Marshal(edge))
			Expect(string(data)).To(ContainSubstring(`"kind":1`))
		})

		It("Should marshal Conditional as 2", func() {
			edge := v0.Edge{
				Source: v0.Handle{Node: "a", Param: "out"},
				Target: v0.Handle{Node: "b", Param: "in"},
				Kind:   v0.EdgeKindConditional,
			}
			data := MustSucceed(json.Marshal(edge))
			Expect(string(data)).To(ContainSubstring(`"kind":2`))
		})

		It("Should unmarshal Continuous edge", func() {
			data := []byte(
				`{"source":{"node":"a","param":"out"},"target":{"node":"b","param":"in"},"kind":1}`,
			)
			var edge v0.Edge
			Expect(json.Unmarshal(data, &edge)).To(Succeed())
			Expect(edge.Kind).To(Equal(v0.EdgeKindContinuous))
		})

		It("Should unmarshal Conditional edge", func() {
			data := []byte(
				`{"source":{"node":"cond","param":"output"},"target":{"node":"stage_entry","param":"activate"},"kind":2}`,
			)
			var edge v0.Edge
			Expect(json.Unmarshal(data, &edge)).To(Succeed())
			Expect(edge.Kind).To(Equal(v0.EdgeKindConditional))
		})
	})
})

var _ = Describe("Edge", func() {
	Describe("DecodeMsgpack", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := v0.Edge{
				Source: v0.Handle{Node: "a", Param: "out"},
				Target: v0.Handle{Node: "b", Param: "in"},
				Kind:   v0.EdgeKindContinuous,
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v0.Edge
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Source v0.Handle
				Target v0.Handle
				Kind   v0.EdgeKind
			}{
				Source: v0.Handle{Node: "a", Param: "out"},
				Target: v0.Handle{Node: "b", Param: "in"},
				Kind:   v0.EdgeKindConditional,
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded v0.Edge
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Source.Node).To(Equal("a"))
			Expect(decoded.Target.Node).To(Equal("b"))
			Expect(decoded.Kind).To(Equal(v0.EdgeKindConditional))
		})
	})

	Describe("String", func() {
		It("Should format a continuous edge with an arrow", func() {
			e := v0.Edge{
				Source: v0.Handle{Node: "a", Param: "out"},
				Target: v0.Handle{Node: "b", Param: "in"},
				Kind:   v0.EdgeKindContinuous,
			}
			Expect(e.String()).To(Equal("a.out -> b.in (EdgeKindContinuous)"))
		})
		It("Should format a conditional edge with a double arrow", func() {
			e := v0.Edge{
				Source: v0.Handle{Node: "a", Param: "out"},
				Target: v0.Handle{Node: "b", Param: "in"},
				Kind:   v0.EdgeKindConditional,
			}
			Expect(e.String()).To(Equal("a.out => b.in (EdgeKindConditional)"))
		})
	})
})
