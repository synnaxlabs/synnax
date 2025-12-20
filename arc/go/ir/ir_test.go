// Copyright 2025 Synnax Labs, Inc.
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
			ir := &ir.IR{}
			Expect(ir.IsZero()).To(BeTrue())
		})

		It("Should return false when Functions is non-empty", func() {
			irWithFuncs := &ir.IR{
				Functions: ir.Functions{{Key: "test"}},
			}
			Expect(irWithFuncs.IsZero()).To(BeFalse())
		})

		It("Should return false when Sequences is non-empty", func() {
			irWithSeqs := &ir.IR{
				Sequences: ir.Sequences{{Key: "main"}},
			}
			Expect(irWithSeqs.IsZero()).To(BeFalse())
		})

		It("Should return false when Nodes is non-empty", func() {
			irWithNodes := &ir.IR{
				Nodes: ir.Nodes{{Key: "node1"}},
			}
			Expect(irWithNodes.IsZero()).To(BeFalse())
		})

		It("Should return false when Edges is non-empty", func() {
			irWithEdges := &ir.IR{
				Edges: ir.Edges{{Kind: ir.Continuous}},
			}
			Expect(irWithEdges.IsZero()).To(BeFalse())
		})

		It("Should return false when Symbols is set", func() {
			irWithSymbols := &ir.IR{
				Symbols: symbol.CreateRootScope(nil),
			}
			Expect(irWithSymbols.IsZero()).To(BeFalse())
		})
	})

	Describe("JSON Marshaling", func() {
		It("Should marshal and unmarshal a complete IR structure", func() {
			inputs := types.Params{}
			inputs = append(inputs, types.Param{Name: "a", Type: types.I64()})
			inputs = append(inputs, types.Param{Name: "b", Type: types.I64()})

			outputs := types.Params{}
			outputs = append(outputs, types.Param{Name: ir.DefaultOutputParam, Type: types.I64()})

			original := &ir.IR{
				Functions: ir.Functions{
					{
						Key:     "add",
						Inputs:  inputs,
						Outputs: outputs,
					},
				},
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
				Strata: ir.Strata{
					{"input_a", "input_b"},
					{"node1"},
				},
			}

			// Marshal to JSON
			data := MustSucceed(json.Marshal(original))
			Expect(data).ToNot(BeEmpty())

			// Unmarshal from JSON
			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored.Functions).To(HaveLen(1))
			Expect(restored.Functions[0].Key).To(Equal("add"))
			Expect(restored.Functions[0].Inputs).To(HaveLen(2))
			Expect(restored.Functions[0].Outputs).To(HaveLen(1))

			Expect(restored.Nodes).To(HaveLen(1))
			Expect(restored.Nodes[0].Key).To(Equal("node1"))
			Expect(restored.Nodes[0].Type).To(Equal("add"))
			Expect(restored.Nodes[0].Config[0].Value).To(Equal(2.0))

			Expect(restored.Edges).To(HaveLen(2))
			Expect(restored.Edges[0].Source.Node).To(Equal("input_a"))
			Expect(restored.Edges[0].Target.Node).To(Equal("node1"))

			Expect(restored.Strata).To(HaveLen(2))
			Expect(restored.Strata[0]).To(HaveLen(2))
			Expect(restored.Strata[1]).To(HaveLen(1))
		})

		It("Should handle empty IR", func() {
			original := &ir.IR{
				Functions: ir.Functions{},
				Nodes:     ir.Nodes{},
				Edges:     ir.Edges{},
				Strata:    ir.Strata{},
			}

			data := MustSucceed(json.Marshal(original))
			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Functions).To(BeEmpty())
			Expect(restored.Nodes).To(BeEmpty())
			Expect(restored.Edges).To(BeEmpty())
			Expect(restored.Strata).To(BeEmpty())
		})

		It("Should exclude Symbols and TypeMap from JSON (json:\"-\" tag)", func() {
			original := &ir.IR{
				Functions: ir.Functions{},
				Nodes:     ir.Nodes{},
				Edges:     ir.Edges{},
				Strata:    ir.Strata{},
				Symbols:   symbol.CreateRootScope(nil),
			}

			data := MustSucceed(json.Marshal(original))

			jsonStr := string(data)
			Expect(jsonStr).ToNot(ContainSubstring("\"symbols\""))
			Expect(jsonStr).ToNot(ContainSubstring("\"TypeMap\""))
		})
	})

	Describe("Complete IR Construction", func() {
		It("Should build a complete IR with all components", func() {
			inputs := types.Params{
				{Name: ir.LHSInputParam, Type: types.I64()},
				{Name: ir.RHSInputParam, Type: types.I64()},
			}

			outputs := types.Params{
				{Name: ir.DefaultOutputParam, Type: types.I64()},
			}

			program := &ir.IR{
				Functions: ir.Functions{
					{
						Key:     "add",
						Inputs:  inputs,
						Outputs: outputs,
					},
				},
				Nodes: ir.Nodes{
					{Key: "input_a", Type: "input", Outputs: types.Params{}},
					{Key: "input_b", Type: "input", Outputs: types.Params{}},
					{Key: "add_node", Type: "add", Inputs: inputs, Outputs: outputs},
					{Key: "output_c", Type: "output", Inputs: types.Params{}},
				},
				Edges: ir.Edges{
					{
						Source: ir.Handle{Node: "input_a", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "add_node", Param: ir.LHSInputParam},
					},
					{
						Source: ir.Handle{Node: "input_b", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "add_node", Param: ir.RHSInputParam},
					},
					{
						Source: ir.Handle{Node: "add_node", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "output_c", Param: ir.DefaultInputParam},
					},
				},
				Strata: ir.Strata{
					{"input_a", "input_b"},
					{"add_node"},
					{"output_c"},
				},
			}

			Expect(program.Functions).To(HaveLen(1))
			Expect(program.Nodes).To(HaveLen(4))
			Expect(program.Edges).To(HaveLen(3))
			Expect(program.Strata).To(HaveLen(3))

			Expect(program.Strata.Get("input_a")).To(Equal(0))
			Expect(program.Strata.Get("input_b")).To(Equal(0))
			Expect(program.Strata.Get("add_node")).To(Equal(1))
			Expect(program.Strata.Get("output_c")).To(Equal(2))

			addInputs := program.Edges.GetInputs("add_node")
			Expect(addInputs).To(HaveLen(2))

			addOutputs := program.Edges.GetOutputs("add_node")
			Expect(addOutputs).To(HaveLen(1))
		})
	})

	Describe("IR with Sequences", func() {
		It("Should marshal and unmarshal complete IR with sequences", func() {
			original := &ir.IR{
				Functions: ir.Functions{
					{Key: "controller", Body: ir.Body{Raw: "..."}},
				},
				Sequences: ir.Sequences{
					{
						Key: "main",
						Stages: []ir.Stage{
							{Key: "init", Nodes: []string{"timer_1", "ctrl_1"}},
							{Key: "run", Nodes: []string{"ctrl_2"}},
							{Key: "done", Nodes: nil},
						},
					},
				},
				Nodes: ir.Nodes{
					{Key: "timer_1", Type: "interval"},
					{Key: "ctrl_1", Type: "controller"},
					{Key: "ctrl_2", Type: "controller"},
					{Key: "main_init_entry", Type: "stage_entry"},
					{Key: "main_run_entry", Type: "stage_entry"},
					{Key: "condition_1", Type: "comparison"},
				},
				Edges: ir.Edges{
					{
						Source: ir.Handle{Node: "timer_1", Param: "output"},
						Target: ir.Handle{Node: "ctrl_1", Param: "input"},
						Kind:   ir.Continuous,
					},
					{
						Source: ir.Handle{Node: "condition_1", Param: "output"},
						Target: ir.Handle{Node: "main_run_entry", Param: "activate"},
						Kind:   ir.OneShot,
					},
				},
				Strata: ir.Strata{{"timer_1"}, {"ctrl_1", "ctrl_2"}},
			}

			data := MustSucceed(json.Marshal(original))

			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			// Verify sequences preserved
			Expect(restored.Sequences).To(HaveLen(1))
			Expect(restored.Sequences[0].Key).To(Equal("main"))
			Expect(restored.Sequences[0].Stages).To(HaveLen(3))

			// Verify entry point works after deserialization
			main := restored.Sequences.Get("main")
			entry := main.Entry()
			Expect(entry.Key).To(Equal("init"))
			Expect(entry.Nodes).To(Equal([]string{"timer_1", "ctrl_1"}))

			// Verify NextStage works
			next, ok := main.NextStage("init")
			Expect(ok).To(BeTrue())
			Expect(next.Key).To(Equal("run"))

			// Verify edge kinds preserved
			Expect(restored.Edges[0].Kind).To(Equal(ir.Continuous))
			Expect(restored.Edges[1].Kind).To(Equal(ir.OneShot))

			// Verify GetByKind works
			continuous := restored.Edges.GetByKind(ir.Continuous)
			oneShot := restored.Edges.GetByKind(ir.OneShot)
			Expect(continuous).To(HaveLen(1))
			Expect(oneShot).To(HaveLen(1))
		})

		It("Should handle IR with no sequences", func() {
			original := &ir.IR{
				Functions: ir.Functions{{Key: "func1"}},
				Nodes:     ir.Nodes{{Key: "node1", Type: "func1"}},
				Edges:     ir.Edges{},
				Strata:    ir.Strata{{"node1"}},
			}

			data := MustSucceed(json.Marshal(original))

			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Sequences).To(BeEmpty())
		})

		It("Should handle IR with multiple sequences", func() {
			original := &ir.IR{
				Sequences: ir.Sequences{
					{
						Key: "main",
						Stages: []ir.Stage{
							{Key: "run", Nodes: []string{"m_1"}},
						},
					},
					{
						Key: "abort",
						Stages: []ir.Stage{
							{Key: "safing", Nodes: []string{"a_1"}},
							{Key: "safed", Nodes: nil},
						},
					},
					{
						Key: "recovery",
						Stages: []ir.Stage{
							{Key: "assess", Nodes: []string{"r_1"}},
						},
					},
				},
				Nodes: ir.Nodes{
					{Key: "m_1", Type: "controller"},
					{Key: "a_1", Type: "controller"},
					{Key: "r_1", Type: "controller"},
				},
				Edges:  ir.Edges{},
				Strata: ir.Strata{},
			}

			data := MustSucceed(json.Marshal(original))

			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored.Sequences).To(HaveLen(3))

			// Test FindStage across sequences
			stage, seq, ok := restored.Sequences.FindStage("safing")
			Expect(ok).To(BeTrue())
			Expect(stage.Key).To(Equal("safing"))
			Expect(seq.Key).To(Equal("abort"))
		})

		It("Should support realistic sequence state machine", func() {
			// A realistic hotfire sequence IR
			// Note: Arc uses U8 for boolean types
			program := &ir.IR{
				Functions: ir.Functions{
					{Key: "interval", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}}},
					{Key: "gt", Inputs: types.Params{{Name: ir.LHSInputParam}, {Name: ir.RHSInputParam}}, Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}}},
					{Key: "stage_entry", Inputs: types.Params{{Name: "activate", Type: types.U8()}}},
				},
				Sequences: ir.Sequences{
					{
						Key: "hotfire",
						Stages: []ir.Stage{
							{Key: "precheck", Nodes: []string{"timer_1", "pressure_check", "precheck_entry"}},
							{Key: "pressurization", Nodes: []string{"valve_ctrl", "pressure_monitor", "pressurization_entry"}},
							{Key: "ignition", Nodes: []string{"igniter", "ignition_entry"}},
							{Key: "mainstage", Nodes: []string{"throttle_ctrl", "mainstage_entry"}},
							{Key: "shutdown", Nodes: []string{"shutdown_seq", "shutdown_entry"}},
						},
					},
				},
				Nodes: ir.Nodes{
					{Key: "timer_1", Type: "interval"},
					{Key: "pressure_check", Type: "gt"},
					{Key: "precheck_entry", Type: "stage_entry"},
					{Key: "valve_ctrl", Type: "controller"},
					{Key: "pressure_monitor", Type: "monitor"},
					{Key: "pressurization_entry", Type: "stage_entry"},
					{Key: "igniter", Type: "controller"},
					{Key: "ignition_entry", Type: "stage_entry"},
					{Key: "throttle_ctrl", Type: "controller"},
					{Key: "mainstage_entry", Type: "stage_entry"},
					{Key: "shutdown_seq", Type: "sequence"},
					{Key: "shutdown_entry", Type: "stage_entry"},
				},
				Edges: ir.Edges{
					// Continuous dataflow
					{Source: ir.Handle{Node: "timer_1", Param: "output"}, Target: ir.Handle{Node: "pressure_check", Param: ir.LHSInputParam}, Kind: ir.Continuous},
					// Stage transitions (OneShot)
					{Source: ir.Handle{Node: "pressure_check", Param: "output"}, Target: ir.Handle{Node: "pressurization_entry", Param: "activate"}, Kind: ir.OneShot},
					{Source: ir.Handle{Node: "pressure_monitor", Param: "threshold"}, Target: ir.Handle{Node: "ignition_entry", Param: "activate"}, Kind: ir.OneShot},
					{Source: ir.Handle{Node: "igniter", Param: "complete"}, Target: ir.Handle{Node: "mainstage_entry", Param: "activate"}, Kind: ir.OneShot},
					{Source: ir.Handle{Node: "timer_1", Param: "timeout"}, Target: ir.Handle{Node: "shutdown_entry", Param: "activate"}, Kind: ir.OneShot},
				},
				Strata: ir.Strata{
					{"timer_1", "valve_ctrl", "igniter", "throttle_ctrl", "shutdown_seq"},
					{"pressure_check", "pressure_monitor"},
				},
			}

			// Verify sequence structure
			hotfire := program.Sequences.Get("hotfire")
			Expect(hotfire.Stages).To(HaveLen(5))

			// Verify entry point
			Expect(hotfire.Entry().Key).To(Equal("precheck"))

			// Verify stage navigation
			stages := []string{"precheck", "pressurization", "ignition", "mainstage", "shutdown"}
			current := stages[0]
			for i := 1; i < len(stages); i++ {
				next, ok := hotfire.NextStage(current)
				Expect(ok).To(BeTrue())
				Expect(next.Key).To(Equal(stages[i]))
				current = next.Key
			}

			// Verify last stage has no next
			_, ok := hotfire.NextStage("shutdown")
			Expect(ok).To(BeFalse())

			// Verify edge classification
			continuous := program.Edges.GetByKind(ir.Continuous)
			oneShot := program.Edges.GetByKind(ir.OneShot)
			Expect(continuous).To(HaveLen(1))
			Expect(oneShot).To(HaveLen(4))

			// All OneShot edges should target stage entries
			for _, e := range oneShot {
				Expect(e.Target.Node).To(ContainSubstring("_entry"))
			}

			// Verify node ownership via stages
			precheckStage, _, ok := program.Sequences.FindStage("precheck")
			Expect(ok).To(BeTrue())
			Expect(precheckStage.Nodes).To(ContainElements("timer_1", "pressure_check", "precheck_entry"))
		})
	})
})
