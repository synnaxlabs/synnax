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
	"github.com/synnaxlabs/x/query"

	"github.com/synnaxlabs/x/graph"
	. "github.com/synnaxlabs/x/testutil"
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

	It("Should return SCC members in sorted order", func() {
		adj := map[string][]string{
			"z": {"y"},
			"y": {"x"},
			"x": {"z"},
		}
		sccs := graph.TarjanSCC(adj)
		Expect(sccs).To(HaveLen(1))
		Expect(sccs[0]).To(Equal([]string{"x", "y", "z"}))
	})

	It("Should produce deterministic output across repeated calls", func() {
		adj := map[string][]string{
			"d": {"a"},
			"c": {"d"},
			"b": {"c"},
			"a": {"b"},
		}
		first := graph.TarjanSCC(adj)
		for i := 0; i < 50; i++ {
			Expect(graph.TarjanSCC(adj)).To(Equal(first))
		}
	})

	It("Should return multiple SCCs in deterministic order", func() {
		adj := map[string][]string{
			"a": {"b"},
			"b": {"a"},
			"x": {"y"},
			"y": {"x"},
		}
		sccs := graph.TarjanSCC(adj)
		Expect(sccs).To(HaveLen(2))
		Expect(sccs[0]).To(Equal([]string{"a", "b"}))
		Expect(sccs[1]).To(Equal([]string{"x", "y"}))
	})

	It("Should sort integer SCC members", func() {
		adj := map[int][]int{
			3: {1},
			1: {2},
			2: {3},
		}
		sccs := graph.TarjanSCC(adj)
		Expect(sccs).To(HaveLen(1))
		Expect(sccs[0]).To(Equal([]int{1, 2, 3}))
	})
})

var _ = Describe("TopoSort", func() {
	It("Should return empty for an empty graph", func() {
		sorted := MustSucceed(graph.TopoSort(map[string][]string{}))
		Expect(sorted).To(BeEmpty())
	})

	It("Should return a single node with no dependencies", func() {
		sorted := MustSucceed(graph.TopoSort(map[string][]string{
			"a": {},
		}))
		Expect(sorted).To(Equal([]string{"a"}))
	})

	It("Should order a simple linear chain", func() {
		sorted := MustSucceed(graph.TopoSort(map[string][]string{
			"a": {"b"},
			"b": {"c"},
			"c": {},
		}))
		Expect(sorted).To(Equal([]string{"c", "b", "a"}))
	})

	It("Should sort independent nodes alphabetically", func() {
		sorted := MustSucceed(graph.TopoSort(map[string][]string{
			"c": {},
			"a": {},
			"b": {},
		}))
		Expect(sorted).To(Equal([]string{"a", "b", "c"}))
	})

	It("Should handle a diamond dependency", func() {
		sorted := MustSucceed(graph.TopoSort(map[string][]string{
			"a": {"b", "c"},
			"b": {"d"},
			"c": {"d"},
			"d": {},
		}))
		Expect(sorted).To(Equal([]string{"d", "b", "c", "a"}))
	})

	It("Should handle multiple roots", func() {
		sorted := MustSucceed(graph.TopoSort(map[string][]string{
			"a": {"c"},
			"b": {"c"},
			"c": {},
		}))
		Expect(sorted).To(Equal([]string{"c", "a", "b"}))
	})

	It("Should handle a complex DAG", func() {
		sorted := MustSucceed(graph.TopoSort(map[string][]string{
			"a": {"b", "c"},
			"b": {"d"},
			"c": {},
			"d": {},
			"e": {"a", "d"},
		}))
		for _, tc := range []struct {
			before string
			after  string
		}{
			{"d", "b"},
			{"b", "a"},
			{"c", "a"},
			{"a", "e"},
			{"d", "e"},
		} {
			bi := indexOf(sorted, tc.before)
			ai := indexOf(sorted, tc.after)
			Expect(bi).To(BeNumerically("<", ai),
				"%s should come before %s in %v", tc.before, tc.after, sorted)
		}
	})

	It("Should work with integer nodes", func() {
		sorted := MustSucceed(graph.TopoSort(map[int][]int{
			1: {2, 3},
			2: {},
			3: {},
		}))
		Expect(sorted).To(Equal([]int{2, 3, 1}))
	})

	It("Should return ErrCyclicDependency for a two-node cycle", func() {
		Expect(graph.TopoSort(map[string][]string{
			"a": {"b"},
			"b": {"a"},
		})).Error().To(MatchError(graph.ErrCyclicDependency))
	})

	It("Should return ErrCyclicDependency for a self-loop", func() {
		Expect(graph.TopoSort(map[string][]string{
			"a": {"a"},
		})).Error().To(MatchError(graph.ErrCyclicDependency))
	})

	It("Should return ErrCyclicDependency for a three-node cycle", func() {
		Expect(graph.TopoSort(map[string][]string{
			"a": {"b"},
			"b": {"c"},
			"c": {"a"},
		})).Error().To(MatchError(graph.ErrCyclicDependency))
	})

	It("Should return ErrCyclicDependency when only part of the graph is cyclic", func() {
		Expect(graph.TopoSort(map[string][]string{
			"a": {},
			"b": {"c"},
			"c": {"b"},
		})).Error().To(MatchError(graph.ErrCyclicDependency))
	})

	It("Should return query.ErrNotFound for an unknown dependency", func() {
		Expect(graph.TopoSort(map[string][]string{
			"a": {"b"},
		})).Error().To(MatchError(query.ErrNotFound))
	})

	It("Should include node names in missing dependency error", func() {
		_, err := graph.TopoSort(map[string][]string{
			"a": {"z"},
		})
		Expect(err).To(MatchError(ContainSubstring("a")))
		Expect(err).To(MatchError(ContainSubstring("z")))
	})

	It("Should produce deterministic output across repeated calls", func() {
		adj := map[string][]string{
			"a": {"c"},
			"b": {"c"},
			"c": {"d"},
			"d": {},
		}
		first := MustSucceed(graph.TopoSort(adj))
		for i := 0; i < 50; i++ {
			Expect(MustSucceed(graph.TopoSort(adj))).To(Equal(first))
		}
	})

	It("Should handle a wide graph with many independent nodes", func() {
		adj := map[string][]string{
			"root": {},
		}
		expected := []string{"root"}
		for _, c := range "abcdefghij" {
			name := string(c)
			adj[name] = []string{"root"}
			expected = append(expected, name)
		}
		sorted := MustSucceed(graph.TopoSort(adj))
		Expect(sorted).To(Equal(expected))
	})
})

func indexOf[T comparable](s []T, v T) int {
	for i, e := range s {
		if e == v {
			return i
		}
	}
	return -1
}
