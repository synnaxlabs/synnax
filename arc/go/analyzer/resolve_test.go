// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

var _ = Describe("ResolveNodeTypes", func() {
	var (
		cs   *constraints.System
		diag *diagnostics.Diagnostics
	)

	BeforeEach(func() {
		cs = constraints.New()
		diag = &diagnostics.Diagnostics{}
	})

	It("Should resolve type variables across edges", func() {
		nodes := ir.Nodes{
			{
				Key:     "source",
				Type:    "on",
				Outputs: types.Params{{Name: "output", Type: types.F64()}},
			},
			{
				Key:  "stat",
				Type: "avg",
				Inputs: types.Params{
					{Name: "input", Type: types.Variable("avg_0_T", nil)},
				},
				Outputs: types.Params{
					{Name: "output", Type: types.Variable("avg_0_T", nil)},
				},
			},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "source", Param: "output"},
			Target: ir.Handle{Node: "stat", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeTrue())
		Expect(diag.Ok()).To(BeTrue())
		Expect(nodes[1].Inputs[0].Type).To(Equal(types.F64()))
		Expect(nodes[1].Outputs[0].Type).To(Equal(types.F64()))
	})

	It("Should error on missing source node", func() {
		nodes := ir.Nodes{
			{
				Key:    "target",
				Type:   "sink",
				Inputs: types.Params{{Name: "input", Type: types.F64()}},
			},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "nonexistent", Param: "output"},
			Target: ir.Handle{Node: "target", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeFalse())
		Expect(
			diag.String(),
		).To(ContainSubstring("source node 'nonexistent' not found"))
	})

	It("Should error on missing source output param", func() {
		nodes := ir.Nodes{
			{
				Key:     "source",
				Type:    "on",
				Outputs: types.Params{{Name: "output", Type: types.F64()}},
			},
			{
				Key:    "target",
				Type:   "sink",
				Inputs: types.Params{{Name: "input", Type: types.F64()}},
			},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "source", Param: "bad_param"},
			Target: ir.Handle{Node: "target", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeFalse())
		Expect(diag.String()).To(ContainSubstring("output 'bad_param' not found"))
	})

	It("Should error on missing target node", func() {
		nodes := ir.Nodes{
			{
				Key:     "source",
				Type:    "on",
				Outputs: types.Params{{Name: "output", Type: types.F64()}},
			},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "source", Param: "output"},
			Target: ir.Handle{Node: "nonexistent", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeFalse())
		Expect(
			diag.String(),
		).To(ContainSubstring("target node 'nonexistent' not found"))
	})

	It("Should skip edges with missing target input param", func() {
		nodes := ir.Nodes{
			{
				Key:     "source",
				Type:    "on",
				Outputs: types.Params{{Name: "output", Type: types.F64()}},
			},
			{
				Key:  "target",
				Type: "func",
				Inputs: types.Params{
					{Name: "value", Type: types.F64(), Value: float64(0)},
				},
			},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "source", Param: "output"},
			Target: ir.Handle{Node: "target", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeTrue())
		Expect(diag.Ok()).To(BeTrue())
	})

	It("Should error when a required input has no incoming edge", func() {
		nodes := ir.Nodes{
			{
				Key:    "sel",
				Type:   "select",
				Inputs: types.Params{{Name: "output", Type: types.U8()}},
			},
		}
		Expect(analyzer.ResolveNodeTypes(nodes, ir.Edges{}, cs, diag)).To(BeFalse())
		Expect(diag.String()).To(ContainSubstring(
			"node 'sel' (select) missing required input 'output'",
		))
	})

	It("Should not error when an unconnected input has a default value", func() {
		nodes := ir.Nodes{
			{
				Key:  "gen",
				Type: "gen",
				Inputs: types.Params{
					{Name: "seed", Type: types.I64(), Value: int64(1)},
				},
			},
		}
		Expect(analyzer.ResolveNodeTypes(nodes, ir.Edges{}, cs, diag)).To(BeTrue())
		Expect(diag.Ok()).To(BeTrue())
	})

	It("Should not error when a required input is satisfied by an edge", func() {
		nodes := ir.Nodes{
			{
				Key:     "source",
				Type:    "on",
				Outputs: types.Params{{Name: "output", Type: types.U8()}},
			},
			{
				Key:    "sel",
				Type:   "select",
				Inputs: types.Params{{Name: "input", Type: types.U8()}},
			},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "source", Param: "output"},
			Target: ir.Handle{Node: "sel", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeTrue())
		Expect(diag.Ok()).To(BeTrue())
	})

	DescribeTable("Should error on type mismatch between concrete types",
		func(src, target types.Type) {
			nodes := ir.Nodes{
				{
					Key:     "source",
					Type:    "on",
					Outputs: types.Params{{Name: "output", Type: src}},
				},
				{
					Key:    "target",
					Type:   "sink",
					Inputs: types.Params{{Name: "input", Type: target}},
				},
			}
			edges := ir.Edges{{
				Source: ir.Handle{Node: "source", Param: "output"},
				Target: ir.Handle{Node: "target", Param: "input"},
			}}
			Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring(
				"type mismatch in edge from 'on' output 'output' " +
					"to 'sink' input 'input': expected " + target.String() +
					", got " + src.String(),
			))
		},
		Entry("f64 into u8", types.F64(), types.U8()),
		Entry("f32 into f64", types.F32(), types.F64()),
		Entry("u8 into i32", types.U8(), types.I32()),
		Entry("i64 into u8", types.I64(), types.U8()),
		Entry("f32 into bool", types.F32(), types.Bool()),
		Entry("bool into f32", types.Bool(), types.F32()),
		Entry("f32 into str", types.F32(), types.String()),
		Entry("str into f64", types.String(), types.F64()),
	)

	DescribeTable("Should describe the edge when a resolved type variable mismatches",
		func(src types.Type) {
			nodes := ir.Nodes{
				{
					Key:     "source",
					Type:    "on",
					Outputs: types.Params{{Name: "output", Type: src}},
				},
				{
					Key:  "stat",
					Type: "avg",
					Inputs: types.Params{
						{Name: "input", Type: types.Variable("avg_0_T", nil)},
					},
					Outputs: types.Params{
						{Name: "output", Type: types.Variable("avg_0_T", nil)},
					},
				},
				{
					Key:    "gate",
					Type:   "select",
					Inputs: types.Params{{Name: "output", Type: types.Bool()}},
				},
			}
			edges := ir.Edges{
				{
					Source: ir.Handle{Node: "source", Param: "output"},
					Target: ir.Handle{Node: "stat", Param: "input"},
				},
				{
					Source: ir.Handle{Node: "stat", Param: "output"},
					Target: ir.Handle{Node: "gate", Param: "output"},
				},
			}
			Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring(
				"type mismatch in edge from 'avg' output 'output' " +
					"to 'select' input 'output': " + src.String() +
					" is not compatible with bool",
			))
		},
		Entry("f32", types.F32()),
		Entry("f64", types.F64()),
		Entry("u8", types.U8()),
		Entry("i32", types.I32()),
		Entry("str", types.String()),
	)

	It("Should succeed with no edges", func() {
		nodes := ir.Nodes{
			{
				Key:     "source",
				Type:    "on",
				Outputs: types.Params{{Name: "output", Type: types.F64()}},
			},
		}
		Expect(analyzer.ResolveNodeTypes(nodes, ir.Edges{}, cs, diag)).To(BeTrue())
		Expect(diag.Ok()).To(BeTrue())
	})

	It("Should apply substitutions to input params", func() {
		nodes := ir.Nodes{
			{
				Key:     "source",
				Type:    "on",
				Outputs: types.Params{{Name: "output", Type: types.F32()}},
			},
			{
				Key:  "func",
				Type: "transform",
				Inputs: types.Params{
					{
						Name:  "threshold",
						Type:  types.Variable("T_0", nil),
						Value: float32(0),
					},
					{Name: "input", Type: types.Variable("T_0", nil)},
				},
				Outputs: types.Params{
					{Name: "output", Type: types.Variable("T_0", nil)},
				},
			},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "source", Param: "output"},
			Target: ir.Handle{Node: "func", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeTrue())
		Expect(nodes[1].Inputs[0].Type).To(Equal(types.F32()))
	})

	DescribeTable("stable.for input types",
		func(input types.Type, wantOK bool) {
			nodes := ir.Nodes{
				{
					Key:     "source",
					Type:    "on",
					Outputs: types.Params{{Name: "output", Type: input}},
				},
				{
					Key:  "stable",
					Type: "stable_for",
					Inputs: types.Params{
						{Name: ir.DefaultInputParam, Type: input},
						{Name: "duration", Type: types.TimeSpan(), Value: int64(1)},
					},
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: input},
					},
				},
			}
			edges := ir.Edges{{
				Source: ir.Handle{Node: "source", Param: "output"},
				Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
			}}
			Expect(
				analyzer.ResolveNodeTypes(nodes, edges, cs, diag),
			).To(Equal(wantOK))
			if !wantOK {
				Expect(diag.String()).To(
					ContainSubstring("must be a numeric or bool type"),
				)
			}
		},
		Entry("accepts f64", types.F64(), true),
		Entry("accepts bool", types.Bool(), true),
		Entry("rejects str", types.String(), false),
		Entry("rejects series str", types.Series(types.String()), false),
	)
})
