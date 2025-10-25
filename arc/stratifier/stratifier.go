// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stratifier

import (
	"context"

	"github.com/synnaxlabs/arc/analyzer/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/errors"
)

// Stratify computes execution strata for nodes in a dataflow graph.
// Stratification enables single-pass, glitch-free reactive execution.
//
// Algorithm:
//  1. Initialize all nodes to stratum 0
//  2. Iteratively assign strata: if node A depends on node B, then
//     stratum(A) = max(stratum(A), stratum(B) + 1)
//  3. Detect cycles: if iteration count exceeds node count, a cycle exists
//
// Returns stratification data, or false if a cycle is detected.
// Cycle errors are added to the diagnostics.
func Stratify(
	ctx context.Context,
	nodes []ir.Node,
	edges []ir.Edge,
	diag *diagnostics.Diagnostics,
) (ir.Strata, bool) {

	var (
		nodeStrata    = make(map[string]int)
		iterations    = 0
		maxIterations = len(nodes) // Upper bound for DAG
		changed       = true
		maxStratum    = 0
	)
	// Handle empty graph
	if len(nodes) == 0 {
		return ir.Strata{}, true
	}

	// Step 1: Initialize ALL nodes to stratum 0
	for _, node := range nodes {
		nodeStrata[node.Key] = 0
	}

	// Step 2: Iterative deepening based on dependencies
	// If a node depends on another, it must be in a higher stratum
	for changed {
		changed = false
		iterations++

		if iterations > maxIterations {
			// Cycle detected - find and report it
			cycle := findCycle(nodes, edges)
			diag.AddError(
				errors.Newf("cycle detected in dataflow graph: %v", cycle),
				nil,
			)
			return ir.Strata{}, false
		}

		for _, edge := range edges {
			sourceStratum := nodeStrata[edge.Source.Node]
			targetStratum := nodeStrata[edge.Target.Node]

			// If source stratum >= target stratum, we need to bump target up
			if sourceStratum >= targetStratum {
				newStratum := sourceStratum + 1
				nodeStrata[edge.Target.Node] = newStratum

				// Track maximum stratum
				if newStratum > maxStratum {
					maxStratum = newStratum
				}

				changed = true
			}
		}
	}

	// Step 3: Convert map to [][]string structure
	strata := make(ir.Strata, maxStratum+1)
	for _, node := range nodes {
		stratum := nodeStrata[node.Key]
		strata[stratum] = append(strata[stratum], node.Key)
	}

	return strata, true
}

// findCycle attempts to find a cycle in the graph for better error reporting
func findCycle(nodes []ir.Node, edges []ir.Edge) []string {
	var (
		graph    = make(map[string][]string)
		visited  = make(map[string]bool)
		recStack = make(map[string]bool)
		path     []string
		dfs      func(node string) bool
	)
	// Build adjacency list
	for _, edge := range edges {
		graph[edge.Source.Node] = append(graph[edge.Source.Node], edge.Target.Node)
	}
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				// Found cycle - extract it from path
				cycleStart := -1
				for i, n := range path {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					path = append(path[cycleStart:], neighbor)
					return true
				}
			}
		}

		recStack[node] = false
		path = path[:len(path)-1]
		return false
	}

	for _, node := range nodes {
		if !visited[node.Key] {
			if dfs(node.Key) {
				return path
			}
		}
	}

	return []string{"unknown cycle"}
}
