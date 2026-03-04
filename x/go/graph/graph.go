// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph

import (
	"cmp"
	"slices"

	"github.com/synnaxlabs/x/set"
)

// TarjanSCC returns all strongly connected components of the directed graph
// represented by the adjacency list adj. Each SCC is a slice of nodes. Singleton
// nodes that do not participate in a cycle are still returned as single-element
// SCCs. The order of SCCs is reverse topological. Node visitation order and SCC
// member order are sorted for deterministic output. O(V log V + E) time, O(V + E) space.
func TarjanSCC[T cmp.Ordered](adj map[T][]T) [][]T {
	var (
		idx      int
		stack    []T
		onStack  = make(set.Set[T])
		indices  = make(map[T]int)
		lowlinks = make(map[T]int)
		defined  = make(set.Set[T])
		sccs     [][]T
	)
	var strongconnect func(v T)
	strongconnect = func(v T) {
		indices[v] = idx
		lowlinks[v] = idx
		idx++
		defined.Add(v)
		stack = append(stack, v)
		onStack.Add(v)
		neighbors := make([]T, len(adj[v]))
		copy(neighbors, adj[v])
		slices.Sort(neighbors)
		for _, w := range neighbors {
			if !defined.Contains(w) {
				strongconnect(w)
				if lowlinks[w] < lowlinks[v] {
					lowlinks[v] = lowlinks[w]
				}
			} else if onStack.Contains(w) {
				if indices[w] < lowlinks[v] {
					lowlinks[v] = indices[w]
				}
			}
		}
		if lowlinks[v] == indices[v] {
			var scc []T
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack.Remove(w)
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}
	keys := make([]T, 0, len(adj))
	for v := range adj {
		keys = append(keys, v)
	}
	slices.Sort(keys)
	for _, v := range keys {
		if !defined.Contains(v) {
			strongconnect(v)
		}
	}
	for _, scc := range sccs {
		slices.Sort(scc)
	}
	return sccs
}
