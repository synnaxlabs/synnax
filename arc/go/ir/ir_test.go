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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("IR", func() {
	Describe("IsZero", func() {
		It("Should return true for zero-value IR", func() {
			program := &ir.IR{}
			Expect(program.IsZero()).To(BeTrue())
		})

		It("Should return false when Functions is non-empty", func() {
			program := &ir.IR{Functions: ir.Functions{{Key: "test"}}}
			Expect(program.IsZero()).To(BeFalse())
		})

		It("Should return false when Root has members", func() {
			program := &ir.IR{
				Root: ir.Scope{
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata: []ir.Members{
						{ir.NodeMember("n1")},
					},
				},
			}
			Expect(program.IsZero()).To(BeFalse())
		})

		It("Should return false when Nodes is non-empty", func() {
			program := &ir.IR{Nodes: ir.Nodes{{Key: "node1"}}}
			Expect(program.IsZero()).To(BeFalse())
		})

		It("Should return false when Edges is non-empty", func() {
			program := &ir.IR{Edges: ir.Edges{{Kind: ir.EdgeKindContinuous}}}
			Expect(program.IsZero()).To(BeFalse())
		})

		It("Should return false when Symbols is set", func() {
			program := &ir.IR{Symbols: symbol.CreateRootScope(nil)}
			Expect(program.IsZero()).To(BeFalse())
		})
	})

	Describe("Scope IsZero", func() {
		It("Should return true for an uninitialized scope", func() {
			Expect(ir.Scope{}.IsZero()).To(BeTrue())
		})

		It("Should return false when a stratum carries members", func() {
			s := ir.Scope{
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessAlways,
				Strata:   []ir.Members{{ir.NodeMember("n1")}},
			}
			Expect(s.IsZero()).To(BeFalse())
		})

		It("Should return false when a sequential scope carries steps", func() {
			s := ir.Scope{
				Key:      "main",
				Mode:     ir.ScopeModeSequential,
				Liveness: ir.LivenessGated,
				Steps:    ir.Members{ir.NodeMember("n1")},
			}
			Expect(s.IsZero()).To(BeFalse())
		})
	})

	Describe("JSON Marshaling", func() {
		It("Should marshal and unmarshal a complete IR structure", func() {
			inputs := types.Params{
				{Name: "a", Type: types.I64()},
				{Name: "b", Type: types.I64()},
			}
			outputs := types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}

			original := &ir.IR{
				Functions: ir.Functions{{Key: "add", Inputs: inputs, Outputs: outputs}},
				Nodes: ir.Nodes{
					{
						Key:     "node1",
						Type:    "add",
						Config:  types.Params{{Name: "multiplier", Type: types.F64(), Value: 2.0}},
						Inputs:  inputs,
						Outputs: outputs,
					},
				},
				Edges: ir.Edges{
					{
						Source: ir.Handle{Node: "input_a", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "node1", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "input_b", Param: "value"},
						Target: ir.Handle{Node: "node1", Param: "b"},
					},
				},
				Root: ir.Scope{
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata: []ir.Members{
						{ir.NodeMember("input_a"), ir.NodeMember("input_b")},
						{ir.NodeMember("node1")},
					},
				},
			}

			data := MustSucceed(json.Marshal(original))
			Expect(data).ToNot(BeEmpty())

			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored.Functions).To(HaveLen(1))
			Expect(restored.Functions[0].Key).To(Equal("add"))
			Expect(restored.Nodes).To(HaveLen(1))
			Expect(restored.Edges).To(HaveLen(2))
			Expect(restored.Root.Mode).To(Equal(ir.ScopeModeParallel))
			Expect(restored.Root.Strata).To(HaveLen(2))
			Expect(restored.Root.Strata[0]).To(HaveLen(2))
		})

		It("Should handle empty IR", func() {
			original := &ir.IR{
				Functions: ir.Functions{},
				Nodes:     ir.Nodes{},
				Edges:     ir.Edges{},
				Root:      ir.Scope{Mode: ir.ScopeModeParallel, Liveness: ir.LivenessAlways},
			}
			data := MustSucceed(json.Marshal(original))
			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Functions).To(BeEmpty())
			Expect(restored.Root.Strata).To(BeEmpty())
		})

		It("Should exclude Symbols and TypeMap from JSON (json:\"-\" tag)", func() {
			original := &ir.IR{Symbols: symbol.CreateRootScope(nil)}
			data := MustSucceed(json.Marshal(original))
			jsonStr := string(data)
			Expect(jsonStr).ToNot(ContainSubstring("\"symbols\""))
			Expect(jsonStr).ToNot(ContainSubstring("\"TypeMap\""))
		})

		It("Should round-trip a sequential scope with transitions", func() {
			stepKey := "run"
			original := &ir.IR{
				Root: ir.Scope{
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata: []ir.Members{{
						{Scope: &ir.Scope{
							Key:      "main",
							Mode:     ir.ScopeModeSequential,
							Liveness: ir.LivenessGated,
							Steps:    ir.Members{ir.NodeMember("init"), ir.NodeMember("run")},
							Transitions: []ir.Transition{
								{
									On:        ir.Handle{Node: "init", Param: "done"},
									TargetKey: &stepKey,
								},
								{
									On:        ir.Handle{Node: "run", Param: "done"},
									TargetKey: nil,
								},
							},
						}},
					}},
				},
			}

			data := MustSucceed(json.Marshal(original))
			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			main := restored.Root.Strata[0][0].Scope
			Expect(main).ToNot(BeNil())
			Expect(main.Mode).To(Equal(ir.ScopeModeSequential))
			Expect(main.Steps).To(HaveLen(2))
			Expect(main.Transitions).To(HaveLen(2))
			Expect(main.Transitions[0].TargetKey).ToNot(BeNil())
			Expect(*main.Transitions[0].TargetKey).To(Equal("run"))
			Expect(main.Transitions[1].TargetKey).To(BeNil())
		})
	})

	Describe("Edge Helpers Against Root Scope", func() {
		It("Should expose dataflow edges independent of the Scope tree", func() {
			program := &ir.IR{
				Functions: ir.Functions{
					{
						Key: "add",
						Inputs: types.Params{
							{Name: ir.LHSInputParam, Type: types.I64()},
							{Name: ir.RHSInputParam, Type: types.I64()},
						},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
					},
				},
				Nodes: ir.Nodes{
					{Key: "input_a", Type: "input"},
					{Key: "input_b", Type: "input"},
					{Key: "add_node", Type: "add"},
					{Key: "output_c", Type: "output"},
				},
				Edges: ir.Edges{
					{
						Source: ir.Handle{Node: "input_a", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "add_node", Param: ir.LHSInputParam},
						Kind:   ir.EdgeKindContinuous,
					},
					{
						Source: ir.Handle{Node: "input_b", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "add_node", Param: ir.RHSInputParam},
						Kind:   ir.EdgeKindContinuous,
					},
					{
						Source: ir.Handle{Node: "add_node", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "output_c", Param: ir.DefaultInputParam},
						Kind:   ir.EdgeKindContinuous,
					},
				},
				Root: ir.Scope{
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata: []ir.Members{
						{ir.NodeMember("input_a"), ir.NodeMember("input_b")},
						{ir.NodeMember("add_node")},
						{ir.NodeMember("output_c")},
					},
				},
			}

			Expect(program.Edges.GetInputs("add_node")).To(HaveLen(2))
			Expect(program.Edges.GetOutputs("add_node")).To(HaveLen(1))
			Expect(program.Edges.GetByKind(ir.EdgeKindContinuous)).To(HaveLen(3))
		})
	})
})
