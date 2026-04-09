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
				Root: ir.Stage{Sequences: ir.Sequences{{Key: "main"}}},
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
				Edges: ir.Edges{{Kind: ir.EdgeKindContinuous}},
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
				Root: ir.Stage{
					Strata: ir.Strata{
						{"input_a", "input_b"},
						{"node1"},
					},
				},
			}

			data := MustSucceed(json.Marshal(original))
			Expect(data).ToNot(BeEmpty())

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

			Expect(restored.Root.Strata).To(HaveLen(2))
			Expect(restored.Root.Strata[0]).To(HaveLen(2))
			Expect(restored.Root.Strata[1]).To(HaveLen(1))
		})

		It("Should handle empty IR", func() {
			original := &ir.IR{
				Functions: ir.Functions{},
				Nodes:     ir.Nodes{},
				Edges:     ir.Edges{},
				Root:      ir.Stage{Strata: ir.Strata{}},
			}

			data := MustSucceed(json.Marshal(original))
			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Functions).To(BeEmpty())
			Expect(restored.Nodes).To(BeEmpty())
			Expect(restored.Edges).To(BeEmpty())
			Expect(restored.Root.Strata).To(BeEmpty())
		})

		It("Should exclude Symbols and TypeMap from JSON (json:\"-\" tag)", func() {
			original := &ir.IR{
				Functions: ir.Functions{},
				Nodes:     ir.Nodes{},
				Edges:     ir.Edges{},
				Root:      ir.Stage{Strata: ir.Strata{}},
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
				Root: ir.Stage{
					Strata: ir.Strata{
						{"input_a", "input_b"},
						{"add_node"},
						{"output_c"},
					},
				},
			}

			Expect(program.Functions).To(HaveLen(1))
			Expect(program.Nodes).To(HaveLen(4))
			Expect(program.Edges).To(HaveLen(3))
			Expect(program.Root.Strata).To(HaveLen(3))

			Expect(program.Root.Strata.Get("input_a")).To(Equal(0))
			Expect(program.Root.Strata.Get("input_b")).To(Equal(0))
			Expect(program.Root.Strata.Get("add_node")).To(Equal(1))
			Expect(program.Root.Strata.Get("output_c")).To(Equal(2))

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
						Kind:   ir.EdgeKindContinuous,
					},
					{
						Source: ir.Handle{Node: "condition_1", Param: "output"},
						Target: ir.Handle{Node: "main_run_entry", Param: "activate"},
						Kind:   ir.EdgeKindOneShot,
					},
				},
				Root: ir.Stage{
					Strata: ir.Strata{{"timer_1"}, {"ctrl_1", "ctrl_2"}},
					Sequences: ir.Sequences{
						{
							Key: "main",
							Steps: []ir.Step{
								stageStep("init", []string{"timer_1", "ctrl_1"}),
								stageStep("run", []string{"ctrl_2"}),
								stageStep("done", nil),
							},
						},
					},
				},
			}

			data := MustSucceed(json.Marshal(original))

			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored.Root.Sequences).To(HaveLen(1))
			Expect(restored.Root.Sequences[0].Key).To(Equal("main"))
			Expect(restored.Root.Sequences[0].Steps).To(HaveLen(3))

			main := restored.Root.Sequences.Get("main")
			entry := main.Entry()
			Expect(entry.Key).To(Equal("init"))
			Expect(entry.StageNodes()).To(Equal([]string{"timer_1", "ctrl_1"}))

			next, ok := main.NextStep("init")
			Expect(ok).To(BeTrue())
			Expect(next.Key).To(Equal("run"))

			Expect(restored.Edges[0].Kind).To(Equal(ir.EdgeKindContinuous))
			Expect(restored.Edges[1].Kind).To(Equal(ir.EdgeKindOneShot))

			continuous := restored.Edges.GetByKind(ir.EdgeKindContinuous)
			oneShot := restored.Edges.GetByKind(ir.EdgeKindOneShot)
			Expect(continuous).To(HaveLen(1))
			Expect(oneShot).To(HaveLen(1))
		})

		It("Should handle IR with no sequences", func() {
			original := &ir.IR{
				Functions: ir.Functions{{Key: "func1"}},
				Nodes:     ir.Nodes{{Key: "node1", Type: "func1"}},
				Edges:     ir.Edges{},
				Root:      ir.Stage{Strata: ir.Strata{{"node1"}}},
			}

			data := MustSucceed(json.Marshal(original))

			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Root.Sequences).To(BeEmpty())
		})

		It("Should handle IR with multiple sequences", func() {
			original := &ir.IR{
				Nodes: ir.Nodes{
					{Key: "m_1", Type: "controller"},
					{Key: "a_1", Type: "controller"},
					{Key: "r_1", Type: "controller"},
				},
				Edges: ir.Edges{},
				Root: ir.Stage{
					Strata: ir.Strata{},
					Sequences: ir.Sequences{
						{
							Key: "main",
							Steps: []ir.Step{
								stageStep("run", []string{"m_1"}),
							},
						},
						{
							Key: "abort",
							Steps: []ir.Step{
								stageStep("safing", []string{"a_1"}),
								stageStep("safed", nil),
							},
						},
						{
							Key: "recovery",
							Steps: []ir.Step{
								stageStep("assess", []string{"r_1"}),
							},
						},
					},
				},
			}

			data := MustSucceed(json.Marshal(original))

			var restored ir.IR
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored.Root.Sequences).To(HaveLen(3))

			step, seq, ok := restored.Root.Sequences.FindStep("safing")
			Expect(ok).To(BeTrue())
			Expect(step.Key).To(Equal("safing"))
			Expect(seq.Key).To(Equal("abort"))
		})

		It("Should support realistic sequence state machine", func() {
			program := &ir.IR{
				Functions: ir.Functions{
					{Key: "interval", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}}},
					{Key: "gt", Inputs: types.Params{{Name: ir.LHSInputParam}, {Name: ir.RHSInputParam}}, Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}}},
					{Key: "stage_entry", Inputs: types.Params{{Name: "activate", Type: types.U8()}}},
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
					{Source: ir.Handle{Node: "timer_1", Param: "output"}, Target: ir.Handle{Node: "pressure_check", Param: ir.LHSInputParam}, Kind: ir.EdgeKindContinuous},
					{Source: ir.Handle{Node: "pressure_check", Param: "output"}, Target: ir.Handle{Node: "pressurization_entry", Param: "activate"}, Kind: ir.EdgeKindOneShot},
					{Source: ir.Handle{Node: "pressure_monitor", Param: "threshold"}, Target: ir.Handle{Node: "ignition_entry", Param: "activate"}, Kind: ir.EdgeKindOneShot},
					{Source: ir.Handle{Node: "igniter", Param: "complete"}, Target: ir.Handle{Node: "mainstage_entry", Param: "activate"}, Kind: ir.EdgeKindOneShot},
					{Source: ir.Handle{Node: "timer_1", Param: "timeout"}, Target: ir.Handle{Node: "shutdown_entry", Param: "activate"}, Kind: ir.EdgeKindOneShot},
				},
				Root: ir.Stage{
					Strata: ir.Strata{
						{"timer_1", "valve_ctrl", "igniter", "throttle_ctrl", "shutdown_seq"},
						{"pressure_check", "pressure_monitor"},
					},
					Sequences: ir.Sequences{
						{
							Key: "hotfire",
							Steps: []ir.Step{
								stageStep("precheck", []string{"timer_1", "pressure_check", "precheck_entry"}),
								stageStep("pressurization", []string{"valve_ctrl", "pressure_monitor", "pressurization_entry"}),
								stageStep("ignition", []string{"igniter", "ignition_entry"}),
								stageStep("mainstage", []string{"throttle_ctrl", "mainstage_entry"}),
								stageStep("shutdown", []string{"shutdown_seq", "shutdown_entry"}),
							},
						},
					},
				},
			}

			hotfire := program.Root.Sequences.Get("hotfire")
			Expect(hotfire.Steps).To(HaveLen(5))

			Expect(hotfire.Entry().Key).To(Equal("precheck"))

			stages := []string{"precheck", "pressurization", "ignition", "mainstage", "shutdown"}
			current := stages[0]
			for i := 1; i < len(stages); i++ {
				next, ok := hotfire.NextStep(current)
				Expect(ok).To(BeTrue())
				Expect(next.Key).To(Equal(stages[i]))
				current = next.Key
			}

			_, ok := hotfire.NextStep("shutdown")
			Expect(ok).To(BeFalse())

			continuous := program.Edges.GetByKind(ir.EdgeKindContinuous)
			oneShot := program.Edges.GetByKind(ir.EdgeKindOneShot)
			Expect(continuous).To(HaveLen(1))
			Expect(oneShot).To(HaveLen(4))

			for _, e := range oneShot {
				Expect(e.Target.Node).To(ContainSubstring("_entry"))
			}

			precheckStep, _, ok := program.Root.Sequences.FindStep("precheck")
			Expect(ok).To(BeTrue())
			Expect(precheckStep.StageNodes()).To(ContainElements("timer_1", "pressure_check", "precheck_entry"))
		})
	})
})
