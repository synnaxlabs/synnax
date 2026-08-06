// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package console_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	graph "github.com/synnaxlabs/arc/graph/versions/v0"
	ir "github.com/synnaxlabs/arc/ir/versions/v0"
	types "github.com/synnaxlabs/arc/types/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/console"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
)

var _ = Describe("Lift", func() {
	It("Should carry the viewport, edges, and nodes across", func() {
		g := console.Graph{
			Viewport: graph.Viewport{Position: spatial.XY{X: 5, Y: 6}, Zoom: 2},
			Edges: ir.Edges{{
				Source: ir.Handle{Node: "n1", Param: "out"},
				Target: ir.Handle{Node: "n2", Param: "in"},
				Kind:   ir.EdgeKindContinuous,
			}},
			Nodes: graph.Nodes{{
				Key:      "n1",
				Type:     "constant",
				Position: spatial.XY{X: 1, Y: 2},
			}},
		}

		out := g.Lift()

		Expect(out.Edges).To(Equal(g.Edges))
		Expect(out.Nodes).To(Equal(g.Nodes))
		Expect(out.Viewport).To(Equal(g.Viewport))
	})

	DescribeTable("Should default the edge kind, keeping one already set",
		func(in, expected ir.EdgeKind) {
			out := console.Graph{Edges: ir.Edges{{
				Source: ir.Handle{Node: "n1", Param: "out"},
				Target: ir.Handle{Node: "n2", Param: "in"},
				Kind:   in,
			}}}.Lift()

			Expect(out.Edges[0].Kind).To(Equal(expected))
		},
		// Payloads before 0.54 wrote no kind field at all.
		Entry("unspecified", ir.EdgeKindUnspecified, ir.EdgeKindContinuous),
		Entry("continuous", ir.EdgeKindContinuous, ir.EdgeKindContinuous),
		Entry("conditional", ir.EdgeKindConditional, ir.EdgeKindConditional),
	)

	It("Should lift a function's params into the typed form", func() {
		out := console.Graph{Functions: []console.Function{
			{
				Key:    "f1",
				Body:   ir.Body{Raw: "out = a"},
				Config: console.Params{{Name: "rate", Value: float64(5)}},
				Inputs: console.Params{
					{Name: "a", Type: console.Type{Kind: types.KindF64}},
				},
				Outputs: console.Params{
					{Name: "out", Type: console.Type{Kind: types.KindF64}},
				},
				Channels: types.Channels{Read: map[uint32]string{1: "a"}},
			},
		}}.Lift()

		f := out.Functions[0]
		Expect(f.Key).To(Equal("f1"))
		Expect(f.Body).To(Equal(ir.Body{Raw: "out = a"}))
		Expect(f.Channels).To(Equal(types.Channels{Read: map[uint32]string{1: "a"}}))
		Expect(f.Config[0].Name).To(Equal("rate"))
		Expect(f.Config[0].Value).To(Equal(float64(5)))
		Expect(f.Inputs[0].Name).To(Equal("a"))
		Expect(f.Inputs[0].Type.Kind).To(Equal(types.KindF64))
		Expect(f.Outputs[0].Name).To(Equal("out"))
		Expect(f.Outputs[0].Type.Kind).To(Equal(types.KindF64))
	})

	It("Should recurse through the element and constraint types", func() {
		elem := console.Type{Kind: types.KindF32}
		constraint := console.Type{Kind: types.KindNumericConstant}
		out := console.Graph{Functions: []console.Function{{
			Inputs: console.Params{{Name: "a", Type: console.Type{
				Kind:       types.KindSeries,
				Elem:       &elem,
				Constraint: &constraint,
			}}},
		}}}.Lift()

		lifted := out.Functions[0].Inputs[0].Type
		Expect(lifted.Kind).To(Equal(types.KindSeries))
		Expect(lifted.Elem.Kind).To(Equal(types.KindF32))
		Expect(lifted.Constraint.Kind).To(Equal(types.KindNumericConstant))
	})

	It("Should leave an absent element and constraint nil", func() {
		out := console.Graph{Functions: []console.Function{
			{
				Inputs: console.Params{
					{Name: "a", Type: console.Type{Kind: types.KindF64}},
				},
			},
		}}.Lift()

		lifted := out.Functions[0].Inputs[0].Type
		Expect(lifted.Elem).To(BeNil())
		Expect(lifted.Constraint).To(BeNil())
	})

	It("Should read the Console's camelCase chanDirection key", func() {
		var g console.Graph
		Expect(json.Unmarshal([]byte(`{"functions":[{"inputs":[{"name":"a",
			"type":{"kind":12,"chanDirection":1}}]}]}`), &g)).To(Succeed())

		lifted := g.Lift().Functions[0].Inputs[0].Type
		Expect(lifted.Kind).To(Equal(types.KindChan))
		Expect(lifted.ChanDirection).To(Equal(types.ChanDirectionRead))
	})

	It("Should produce an empty graph from an empty export", func() {
		out := console.Graph{}.Lift()

		Expect(out.Functions).To(BeEmpty())
		Expect(out.Edges).To(BeEmpty())
		Expect(out.Nodes).To(BeEmpty())
	})
})

var _ = Describe("Data", func() {
	It("Should decode the mode and text alongside the graph", func() {
		var d console.Data
		Expect(json.Unmarshal(
			[]byte(`{"mode":"text","text":{"raw":"a = 1"},"graph":{"nodes":[]}}`),
			&d,
		)).To(Succeed())

		Expect(d.Mode).To(Equal("text"))
		Expect(d.Text.Raw).To(Equal("a = 1"))
	})
})
