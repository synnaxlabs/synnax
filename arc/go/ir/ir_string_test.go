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
	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("IR", func() {
	Describe("IsZero", func() {
		DescribeTable(
			"Classification",
			func(ir ir.IR, expected bool) {
				Expect(ir.IsZero()).To(Equal(expected))
			},
			Entry("an empty IR", ir.IR{}, true),
			Entry("a function",
				ir.IR{Functions: ir.Functions{{Key: "add"}}}, false),
			Entry("a node",
				ir.IR{Nodes: ir.Nodes{{Key: "n1"}}}, false),
			Entry("an edge",
				ir.IR{Edges: ir.Edges{{Kind: ir.EdgeKindContinuous}}}, false),
			Entry("a non-zero root",
				ir.IR{Root: ir.Scope{Key: "root"}}, false),
			Entry("symbols",
				ir.IR{Symbols: &symbol.Symbol{}}, false),
			Entry("a type map",
				ir.IR{TypeMap: map[antlr.ParserRuleContext]types.Type{}}, false),
		)
	})

	Describe("String", func() {
		It("Should return an empty string for an empty IR", func() {
			ir := ir.IR{}
			Expect(ir.String()).To(Equal(""))
		})

		It("Should render functions, nodes, edges, and root as tree sections", func() {
			ir := ir.IR{
				Functions: ir.Functions{{Key: "add"}},
				Nodes:     ir.Nodes{{Key: "n1", Type: "add"}},
				Edges: ir.Edges{{
					Source: ir.Handle{Node: "n1", Param: "out"},
					Target: ir.Handle{Node: "n2", Param: "in"},
					Kind:   ir.EdgeKindContinuous,
				}},
				Root: ir.Scope{
					Key:      "root",
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
				},
			}
			out := ir.String()
			Expect(out).To(HavePrefix("├── Functions (1)\n"))
			Expect(out).To(ContainSubstring("├── Nodes (1)\n"))
			Expect(out).To(ContainSubstring("n1 (type: add)"))
			Expect(out).To(ContainSubstring("├── Edges (1)\n"))
			Expect(out).To(ContainSubstring("n1.out -> n2.in (EdgeKindContinuous)"))
			Expect(out).To(ContainSubstring("└── Root\n"))
			Expect(out).To(ContainSubstring("root [ScopeModeParallel, LivenessAlways]"))
		})

		It("Should mark the only section as the last tree item", func() {
			ir := ir.IR{Nodes: ir.Nodes{{Key: "n1", Type: "add"}}}
			Expect(ir.String()).To(HavePrefix("└── Nodes (1)\n"))
		})
	})
})
