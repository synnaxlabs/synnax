// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/arc/ir/types/v2"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("IR", func() {
	Describe("IsZero", func() {
		DescribeTable(
			"Classification",
			func(ir v2.IR, expected bool) {
				Expect(ir.IsZero()).To(Equal(expected))
			},
			Entry("an empty IR", v2.IR{}, true),
			Entry("a function",
				v2.IR{Functions: v2.Functions{{Key: "add"}}}, false),
			Entry("a node",
				v2.IR{Nodes: v2.Nodes{{Key: "n1"}}}, false),
			Entry("an edge",
				v2.IR{Edges: v2.Edges{{Kind: v2.EdgeKindContinuous}}}, false),
			Entry("a non-zero root",
				v2.IR{Root: v2.Scope{Key: "root"}}, false),
			Entry("symbols",
				v2.IR{Symbols: &symbol.Symbol{}}, false),
			Entry("a type map",
				v2.IR{TypeMap: map[antlr.ParserRuleContext]types.Type{}}, false),
		)
	})

	Describe("String", func() {
		It("Should return an empty string for an empty IR", func() {
			ir := v2.IR{}
			Expect(ir.String()).To(Equal(""))
		})

		It("Should render functions, nodes, edges, and root as tree sections", func() {
			ir := v2.IR{
				Functions: v2.Functions{{Key: "add"}},
				Nodes:     v2.Nodes{{Key: "n1", Type: "add"}},
				Edges: v2.Edges{{
					Source: v2.Handle{Node: "n1", Param: "out"},
					Target: v2.Handle{Node: "n2", Param: "in"},
					Kind:   v2.EdgeKindContinuous,
				}},
				Root: v2.Scope{
					Key:      "root",
					Mode:     v2.ScopeModeParallel,
					Liveness: v2.LivenessAlways,
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
			ir := v2.IR{Nodes: v2.Nodes{{Key: "n1", Type: "add"}}}
			Expect(ir.String()).To(HavePrefix("└── Nodes (1)\n"))
		})
	})
})
