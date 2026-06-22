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
})
