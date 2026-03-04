// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph

import "github.com/synnaxlabs/x/set"

// TarjanSCC returns all strongly connected components of the directed graph
// represented by the adjacency list adj. Each SCC is a slice of nodes. Singleton
// nodes that do not participate in a cycle are still returned as single-element
// SCCs. The order of SCCs is reverse topological. O(V + E) time and space.
func TarjanSCC[T comparable](adj map[T][]T) [][]T {
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
		for _, w := range adj[v] {
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
	for v := range adj {
		if !defined.Contains(v) {
			strongconnect(v)
		}
	}
	return sccs
}
