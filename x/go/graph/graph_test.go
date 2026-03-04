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

	"github.com/synnaxlabs/x/graph"
)

var _ = Describe("TarjanSCC", func() {
	It("Should return empty for an empty graph", func() {
		adj := map[string][]string{}
		sccs := graph.TarjanSCC(adj)
		Expect(sccs).To(BeEmpty())
	})

	It("Should return singletons for a DAG", func() {
		adj := map[string][]string{
			"a": {"b"},
			"b": {"c"},
			"c": {},
		}
		sccs := graph.TarjanSCC(adj)
		Expect(sccs).To(HaveLen(3))
		for _, scc := range sccs {
			Expect(scc).To(HaveLen(1))
		}
	})

	It("Should detect a self-loop", func() {
		adj := map[string][]string{
			"a": {"a"},
		}
		sccs := graph.TarjanSCC(adj)
		Expect(sccs).To(HaveLen(1))
		Expect(sccs[0]).To(ConsistOf("a"))
	})

	It("Should detect a simple two-node cycle", func() {
		adj := map[string][]string{
			"a": {"b"},
			"b": {"a"},
		}
		sccs := graph.TarjanSCC(adj)
		var cycleSCC []string
		for _, scc := range sccs {
			if len(scc) > 1 {
				cycleSCC = scc
			}
		}
		Expect(cycleSCC).To(ConsistOf("a", "b"))
	})

	It("Should detect a three-node cycle", func() {
		adj := map[string][]string{
			"a": {"b"},
			"b": {"c"},
			"c": {"a"},
		}
		sccs := graph.TarjanSCC(adj)
		Expect(sccs).To(HaveLen(1))
		Expect(sccs[0]).To(ConsistOf("a", "b", "c"))
	})

	It("Should separate independent cycles", func() {
		adj := map[string][]string{
			"a": {"b"},
			"b": {"a"},
			"c": {"d"},
			"d": {"c"},
		}
		sccs := graph.TarjanSCC(adj)
		Expect(sccs).To(HaveLen(2))
		for _, scc := range sccs {
			Expect(scc).To(HaveLen(2))
		}
	})

	It("Should work with integer nodes", func() {
		adj := map[int][]int{
			1: {2},
			2: {3},
			3: {1},
		}
		sccs := graph.TarjanSCC(adj)
		Expect(sccs).To(HaveLen(1))
		Expect(sccs[0]).To(ConsistOf(1, 2, 3))
	})

	It("Should handle a diamond that is not a cycle", func() {
		adj := map[string][]string{
			"a": {"b", "c"},
			"b": {"d"},
			"c": {"d"},
			"d": {},
		}
		sccs := graph.TarjanSCC(adj)
		Expect(sccs).To(HaveLen(4))
		for _, scc := range sccs {
			Expect(scc).To(HaveLen(1))
		}
	})
})
