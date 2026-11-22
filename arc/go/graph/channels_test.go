// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("ChannelInputs", func() {
	It("Should name the channel inputs of each node's function", func(ctx SpecContext) {
		nodes, inputs := buildNodes(
			nodeSpec{key: "source", typ: "on", cfg: map[string]any{"channel": 12}},
			nodeSpec{key: "sink", typ: "write", cfg: map[string]any{"channel": 13}},
			nodeSpec{key: "scaled", typ: "scale", cfg: map[string]any{"gain": 2}},
			nodeSpec{key: "fixed", typ: "constant", cfg: map[string]any{"value": 1}},
		)
		g := graph.Graph{
			Functions: ir.Functions{{
				Key: "scale",
				Inputs: types.Params{
					{Name: "gain", Type: types.F32()},
					{Name: "src", Type: types.ReadChan(types.F32())},
				},
			}},
			Nodes:  nodes,
			Inputs: inputs,
		}
		Expect(graph.ChannelInputs(ctx, g, NewGraphRoot(nil))).To(Equal(
			map[string][]string{
				"source": {"channel"},
				"sink":   {"channel"},
				"scaled": {"src"},
			},
		))
	})

	It("Should skip a node with no function type or an undefined function", func(
		ctx SpecContext,
	) {
		nodes, inputs := buildNodes(
			nodeSpec{key: "unknown", typ: "no_such_function"},
		)
		nodes = append(nodes, graph.Node{Key: "untyped"})
		g := graph.Graph{Nodes: nodes, Inputs: inputs}
		Expect(graph.ChannelInputs(ctx, g, NewGraphRoot(nil))).To(BeEmpty())
	})

	It("Should resolve a graph function before a root function of the same name", func(
		ctx SpecContext,
	) {
		nodes, inputs := buildNodes(nodeSpec{key: "n", typ: "on"})
		g := graph.Graph{
			Functions: ir.Functions{{
				Key:    "on",
				Inputs: types.Params{{Name: "threshold", Type: types.F32()}},
			}},
			Nodes:  nodes,
			Inputs: inputs,
		}
		Expect(graph.ChannelInputs(ctx, g, NewGraphRoot(nil))).To(BeEmpty())
	})
})
