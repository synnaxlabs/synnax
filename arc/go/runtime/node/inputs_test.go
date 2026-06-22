// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ResolvedInputs", func() {
	It("overlays an edge-fed numeric input onto the static value map", func() {
		prog := ir.IR{
			Nodes: ir.Nodes{
				{Key: "src", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}},
				{Key: "tgt", Inputs: types.Params{
					{Name: "static_p", Type: types.I64(), Value: int64(7)},
					{Name: "edge_p", Type: types.I64()},
				}},
			},
			Edges: ir.Edges{{
				Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "tgt", Param: "edge_p"},
			}},
		}
		s := node.New(prog)
		src, tgt := s.Node("src"), s.Node("tgt")
		*src.Output(0) = telem.NewSeriesV[int64](42)
		*src.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
		tgt.RefreshInputs()

		ri := MustSucceed(node.ResolveInputs(tgt, prog.Nodes.Get("tgt")))
		Expect(ri.HasEdges()).To(BeTrue())
		m := ri.ValueMap(tgt)
		Expect(m["static_p"]).To(Equal(int64(7)))
		Expect(m["edge_p"]).To(Equal(int64(42)))
	})

	It("reads an edge-fed string input", func() {
		prog := ir.IR{
			Nodes: ir.Nodes{
				{Key: "src", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}}},
				{Key: "tgt", Inputs: types.Params{{Name: "name", Type: types.String()}}},
			},
			Edges: ir.Edges{{
				Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "tgt", Param: "name"},
			}},
		}
		s := node.New(prog)
		src, tgt := s.Node("src"), s.Node("tgt")
		*src.Output(0) = telem.NewSeriesV[string]("My_range_3")
		*src.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
		tgt.RefreshInputs()

		ri := MustSucceed(node.ResolveInputs(tgt, prog.Nodes.Get("tgt")))
		Expect(ri.ValueMap(tgt)["name"]).To(Equal("My_range_3"))
	})

	It("reports no edges when every input is static", func() {
		prog := ir.IR{Nodes: ir.Nodes{
			{Key: "tgt", Inputs: types.Params{{Name: "p", Type: types.I64(), Value: int64(1)}}},
		}}
		s := node.New(prog)
		ri := MustSucceed(node.ResolveInputs(s.Node("tgt"), prog.Nodes.Get("tgt")))
		Expect(ri.HasEdges()).To(BeFalse())
		Expect(ri.ValueMap(s.Node("tgt"))["p"]).To(Equal(int64(1)))
	})

	It("does not treat a channel-typed input as edge-fed", func() {
		prog := ir.IR{Nodes: ir.Nodes{
			{Key: "tgt", Inputs: types.Params{
				{Name: "ch", Type: types.Chan(types.I64())},
				{Name: "static_p", Type: types.I64(), Value: int64(5)},
			}},
		}}
		s := node.New(prog)
		ri := MustSucceed(node.ResolveInputs(s.Node("tgt"), prog.Nodes.Get("tgt")))
		Expect(ri.HasEdges()).To(BeFalse())
	})

	It("returns ErrInputNotFound when an edge-fed param is absent from state", func() {
		prog := ir.IR{Nodes: ir.Nodes{
			{Key: "tgt", Inputs: types.Params{{Name: "x", Type: types.I64(), Value: int64(1)}}},
		}}
		s := node.New(prog)
		// A node carrying an edge-fed input the state was not built with.
		mismatched := ir.Node{Key: "tgt", Inputs: types.Params{{Name: "phantom", Type: types.I64()}}}
		Expect(node.ResolveInputs(s.Node("tgt"), mismatched)).
			Error().To(MatchError(node.ErrInputNotFound))
	})

	Describe("ValidationMap", func() {
		It("substitutes a typed zero for an edge-fed input and keeps static values", func() {
			prog := ir.IR{
				Nodes: ir.Nodes{
					{Key: "src", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}},
					{Key: "tgt", Inputs: types.Params{
						{Name: "static_p", Type: types.String(), Value: "kept"},
						{Name: "edge_p", Type: types.I64()},
					}},
				},
				Edges: ir.Edges{{
					Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "tgt", Param: "edge_p"},
				}},
			}
			s := node.New(prog)
			ri := MustSucceed(node.ResolveInputs(s.Node("tgt"), prog.Nodes.Get("tgt")))
			m := ri.ValidationMap()
			Expect(m["static_p"]).To(Equal("kept"))
			Expect(m["edge_p"]).To(Equal(int64(0)))
		})

		DescribeTable("yields the Go-typed zero matching the edge-fed param type",
			func(t types.Type, want any) {
				prog := ir.IR{
					Nodes: ir.Nodes{
						{Key: "src", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: t}}},
						{Key: "tgt", Inputs: types.Params{{Name: "p", Type: t}}},
					},
					Edges: ir.Edges{{
						Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "tgt", Param: "p"},
					}},
				}
				s := node.New(prog)
				ri := MustSucceed(node.ResolveInputs(s.Node("tgt"), prog.Nodes.Get("tgt")))
				Expect(ri.ValidationMap()["p"]).To(Equal(want))
			},
			Entry("string", types.String(), ""),
			Entry("f32", types.F32(), float32(0)),
			Entry("f64", types.F64(), float64(0)),
			Entry("u8", types.U8(), uint8(0)),
			Entry("u16", types.U16(), uint16(0)),
			Entry("u32", types.U32(), uint32(0)),
			Entry("u64", types.U64(), uint64(0)),
			Entry("i8", types.I8(), int8(0)),
			Entry("i16", types.I16(), int16(0)),
			Entry("i32", types.I32(), int32(0)),
			Entry("i64", types.I64(), int64(0)),
			Entry("timestamp falls through to i64", types.TimeStamp(), int64(0)),
		)
	})
})
