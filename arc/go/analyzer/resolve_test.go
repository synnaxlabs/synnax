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
				Key:     "stat",
				Type:    "avg",
				Inputs:  types.Params{{Name: "input", Type: types.Variable("avg_0_T", nil)}},
				Outputs: types.Params{{Name: "output", Type: types.Variable("avg_0_T", nil)}},
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
			{Key: "target", Type: "sink", Inputs: types.Params{{Name: "input", Type: types.F64()}}},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "nonexistent", Param: "output"},
			Target: ir.Handle{Node: "target", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeFalse())
		Expect(diag.String()).To(ContainSubstring("source node 'nonexistent' not found"))
	})

	It("Should error on missing source output param", func() {
		nodes := ir.Nodes{
			{Key: "source", Type: "on", Outputs: types.Params{{Name: "output", Type: types.F64()}}},
			{Key: "target", Type: "sink", Inputs: types.Params{{Name: "input", Type: types.F64()}}},
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
			{Key: "source", Type: "on", Outputs: types.Params{{Name: "output", Type: types.F64()}}},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "source", Param: "output"},
			Target: ir.Handle{Node: "nonexistent", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeFalse())
		Expect(diag.String()).To(ContainSubstring("target node 'nonexistent' not found"))
	})

	It("Should skip edges with missing target input param", func() {
		nodes := ir.Nodes{
			{Key: "source", Type: "on", Outputs: types.Params{{Name: "output", Type: types.F64()}}},
			{Key: "target", Type: "func", Inputs: types.Params{{Name: "value", Type: types.F64()}}},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "source", Param: "output"},
			Target: ir.Handle{Node: "target", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeTrue())
		Expect(diag.Ok()).To(BeTrue())
	})

	It("Should error on type mismatch between concrete types", func() {
		nodes := ir.Nodes{
			{Key: "source", Type: "on", Outputs: types.Params{{Name: "output", Type: types.F64()}}},
			{Key: "target", Type: "sink", Inputs: types.Params{{Name: "input", Type: types.U8()}}},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "source", Param: "output"},
			Target: ir.Handle{Node: "target", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeFalse())
		Expect(diag.String()).To(ContainSubstring("type mismatch"))
	})

	It("Should succeed with no edges", func() {
		nodes := ir.Nodes{
			{Key: "source", Type: "on", Outputs: types.Params{{Name: "output", Type: types.F64()}}},
		}
		Expect(analyzer.ResolveNodeTypes(nodes, ir.Edges{}, cs, diag)).To(BeTrue())
		Expect(diag.Ok()).To(BeTrue())
	})

	It("Should apply substitutions to config params", func() {
		nodes := ir.Nodes{
			{Key: "source", Type: "on", Outputs: types.Params{{Name: "output", Type: types.F32()}}},
			{
				Key:     "func",
				Type:    "transform",
				Inputs:  types.Params{{Name: "input", Type: types.Variable("T_0", nil)}},
				Outputs: types.Params{{Name: "output", Type: types.Variable("T_0", nil)}},
				Config:  types.Params{{Name: "threshold", Type: types.Variable("T_0", nil)}},
			},
		}
		edges := ir.Edges{{
			Source: ir.Handle{Node: "source", Param: "output"},
			Target: ir.Handle{Node: "func", Param: "input"},
		}}
		Expect(analyzer.ResolveNodeTypes(nodes, edges, cs, diag)).To(BeTrue())
		Expect(nodes[1].Config[0].Type).To(Equal(types.F32()))
	})
})
