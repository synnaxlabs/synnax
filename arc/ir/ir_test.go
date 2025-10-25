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
)

var _ = Describe("IR", func() {
	Describe("JSON Marshaling", func() {
		It("Should marshal and unmarshal a complete IR structure", func() {
			// Create a complete IR with all components
			inputs := types.Params{}
			inputs.Put("a", types.I64())
			inputs.Put("b", types.I64())

			outputs := types.Params{}
			outputs.Put(ir.DefaultOutputParam, types.I64())

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
						Key:          "node1",
						Type:         "add",
						ConfigValues: map[string]any{"multiplier": 2.0},
						Inputs:       inputs,
						Outputs:      outputs,
					},
				},
				Edges: ir.Edges{
					{
						Source: ir.Handle{Node: "input_a", Param: "value"},
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
			data, err := json.Marshal(original)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).ToNot(BeEmpty())

			// Unmarshal from JSON
			var restored ir.IR
			err = json.Unmarshal(data, &restored)
			Expect(err).ToNot(HaveOccurred())

			// Verify Functions
			Expect(restored.Functions).To(HaveLen(1))
			Expect(restored.Functions[0].Key).To(Equal("add"))
			Expect(restored.Functions[0].Inputs.Count()).To(Equal(2))
			Expect(restored.Functions[0].Outputs.Count()).To(Equal(1))

			// Verify Nodes
			Expect(restored.Nodes).To(HaveLen(1))
			Expect(restored.Nodes[0].Key).To(Equal("node1"))
			Expect(restored.Nodes[0].Type).To(Equal("add"))
			Expect(restored.Nodes[0].ConfigValues["multiplier"]).To(Equal(2.0))

			// Verify Edges
			Expect(restored.Edges).To(HaveLen(2))
			Expect(restored.Edges[0].Source.Node).To(Equal("input_a"))
			Expect(restored.Edges[0].Target.Node).To(Equal("node1"))

			// Verify Strata
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

			data, err := json.Marshal(original)
			Expect(err).ToNot(HaveOccurred())

			var restored ir.IR
			err = json.Unmarshal(data, &restored)
			Expect(err).ToNot(HaveOccurred())

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

			data, err := json.Marshal(original)
			Expect(err).ToNot(HaveOccurred())

			// Verify that the JSON doesn't contain "symbols" or "TypeMap" fields
			jsonStr := string(data)
			Expect(jsonStr).ToNot(ContainSubstring("\"symbols\""))
			Expect(jsonStr).ToNot(ContainSubstring("\"TypeMap\""))
		})
	})

	Describe("Complete IR Construction", func() {
		It("Should build a complete IR with all components", func() {
			// Create a simple dataflow: input -> add -> output
			inputs := types.Params{}
			inputs.Put(ir.LHSInputParam, types.I64())
			inputs.Put(ir.RHSInputParam, types.I64())

			outputs := types.Params{}
			outputs.Put(ir.DefaultOutputParam, types.I64())

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

			// Verify the IR is well-formed
			Expect(program.Functions).To(HaveLen(1))
			Expect(program.Nodes).To(HaveLen(4))
			Expect(program.Edges).To(HaveLen(3))
			Expect(program.Strata).To(HaveLen(3))

			// Verify stratification is correct
			Expect(program.Strata.Get("input_a")).To(Equal(0))
			Expect(program.Strata.Get("input_b")).To(Equal(0))
			Expect(program.Strata.Get("add_node")).To(Equal(1))
			Expect(program.Strata.Get("output_c")).To(Equal(2))

			// Verify edges connect properly
			addInputs := program.Edges.GetInputs("add_node")
			Expect(addInputs).To(HaveLen(2))

			addOutputs := program.Edges.GetOutputs("add_node")
			Expect(addOutputs).To(HaveLen(1))
		})
	})
})