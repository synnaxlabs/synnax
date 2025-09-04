// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package module_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/module"
	"github.com/synnaxlabs/slate/parser"
)

func TestGraph(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Graph Suite")
}

var _ = Describe("Graph Assembly", func() {
	parseAndAnalyze := func(code string) (parser.IProgramContext, analyzer.Result) {
		prog, err := parser.Parse(code)
		Expect(err).To(BeNil())
		result := analyzer.Analyze(analyzer.Options{
			Program:  prog,
			Resolver: nil,
		})
		Expect(result.Diagnostics).To(BeEmpty())
		return prog, result
	}

	Describe("Basic Flow Assembly", func() {
		It("Should assemble a basic task to task flow", func() {
			code := `
			task first{
				value f64
			} () f64 {}
			task second{
				value f64
			} () {
			}
			first{value: 10.0} -> second{value: 20.0}
			`
			prog, scope := parseAndAnalyze(code)
			g, err := module.Assemble(prog, scope)
			Expect(err).To(BeNil())
			Expect(g).NotTo(BeNil())
			Expect(g.Nodes).To(HaveLen(2))
			Expect(g.Edges).To(HaveLen(1))
			for _, node := range g.Nodes {
				if node.Type == "first" {
					Expect(node.Key).To(Equal("first_1"))
				} else if node.Type == "second" {
					Expect(node.Key).To(Equal("second_2"))
				}
			}
			edge := g.Edges[0]
			Expect(edge.Target.Node).To(Equal("second_2"))
			Expect(edge.Target.Param).To(Equal("output"))
			Expect(edge.Source.Node).To(Equal("first_1"))
			Expect(edge.Source.Param).To(Equal("output"))
		})
	})
})
