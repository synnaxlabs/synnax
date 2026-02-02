// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// stageEntryKey computes the entry node key for a stage using the established
// naming convention. This must match the key generation in text.KeyGenerator.Entry.
func stageEntryKey(seqName, stageName string) string {
	return "entry_" + seqName + "_" + stageName
}

// Stratify computes execution strata for nodes in a dataflow graph using a two-tier
// stratification model:
//
//  1. Global strata: Nodes outside any sequence stage (including entry nodes receiving
//     initial activation signals)
//  2. Per-stage strata: Each stage is stratified independently, with stage-local sources
//     (constants, channel reads) at stratum 0
//
// This design eliminates implicit dependencies from entry nodes to stage-internal nodes,
// allowing:
//   - Stage-local source nodes to execute every cycle (stratum 0 within their stage)
//   - Cyclic stage transitions (valid state machines) without false cycle detection
//
// The returned Strata contains global stratification. Per-stage strata are populated
// directly into the sequences parameter (which is modified in place).
//
// Returns global strata and diagnostics. If a cycle is detected within any subgraph,
// an error is added to diagnostics and empty strata is returned.
func Stratify(
	ctx context.Context,
	nodes []ir.Node,
	edges []ir.Edge,
	sequences []ir.Sequence,
	diag *diagnostics.Diagnostics,
) (ir.Strata, *diagnostics.Diagnostics) {
	if len(nodes) == 0 {
		return ir.Strata{}, nil
	}

	// Build lookup structures
	nodeByKey := make(map[string]ir.Node)
	for _, node := range nodes {
		nodeByKey[node.Key] = node
	}

	// Build set of all staged node keys and entry node keys
	stagedNodes := make(set.Set[string])
	entryNodes := make(set.Set[string])
	for _, seq := range sequences {
		for _, stage := range seq.Stages {
			for _, nodeKey := range stage.Nodes {
				stagedNodes.Add(nodeKey)
			}
			entryNodes.Add(stageEntryKey(seq.Key, stage.Key))
		}
	}

	// Step 1: Stratify global nodes (nodes not in any stage)
	// Entry nodes are only included in global strata if they receive activation from
	// global sources (e.g., start_cmd => main). Entry nodes with no global incoming
	// edges are excluded to prevent them from being placed at stratum 0 and firing
	// every cycle.

	// First, collect all non-staged nodes
	var globalNodes []ir.Node
	globalNodeSet := make(set.Set[string])
	for _, node := range nodes {
		if !stagedNodes.Contains(node.Key) {
			globalNodes = append(globalNodes, node)
			globalNodeSet.Add(node.Key)
		}
	}

	// Build set of entry nodes that have incoming edges from global sources
	entryNodesWithGlobalInput := make(set.Set[string])
	for _, edge := range edges {
		if globalNodeSet.Contains(edge.Source.Node) && entryNodes.Contains(edge.Target.Node) {
			entryNodesWithGlobalInput.Add(edge.Target.Node)
		}
	}

	// Filter global nodes: exclude entry nodes that don't have global incoming edges
	var filteredGlobalNodes []ir.Node
	filteredGlobalNodeSet := make(set.Set[string])
	for _, node := range globalNodes {
		if entryNodes.Contains(node.Key) && !entryNodesWithGlobalInput.Contains(node.Key) {
			// Entry node with no global incoming edges - exclude from global strata
			continue
		}
		filteredGlobalNodes = append(filteredGlobalNodes, node)
		filteredGlobalNodeSet.Add(node.Key)
	}

	// Filter edges for global subgraph: edges where source is a global node
	// (target can be global node or entry node of a stage)
	var globalEdges []ir.Edge
	for _, edge := range edges {
		if filteredGlobalNodeSet.Contains(edge.Source.Node) {
			// Source is global - include this edge
			// Target can be global or an entry node
			globalEdges = append(globalEdges, edge)
		}
	}

	globalStrata, cycleDiag := stratifySubgraph(filteredGlobalNodes, globalEdges, diag)
	if cycleDiag != nil && !cycleDiag.Ok() {
		return ir.Strata{}, cycleDiag
	}

	// Step 2: Stratify each stage independently
	// Entry nodes that receive edges from a stage are included in that stage's
	// stratification so they fire after their source nodes within the stage.
	for i, seq := range sequences {
		for j, stage := range seq.Stages {
			stageNodeSet := make(set.Set[string])
			for _, nodeKey := range stage.Nodes {
				stageNodeSet.Add(nodeKey)
			}

			// Find entry nodes that receive edges from this stage
			// These need to be included in the stage's strata so they can fire
			// when their source nodes output truthy values
			entryNodesForStage := make(set.Set[string])
			for _, edge := range edges {
				if stageNodeSet.Contains(edge.Source.Node) && entryNodes.Contains(edge.Target.Node) {
					entryNodesForStage.Add(edge.Target.Node)
				}
			}

			// Add entry nodes to stage node set for stratification
			for entryKey := range entryNodesForStage {
				stageNodeSet.Add(entryKey)
			}

			// Collect nodes for this stage (including entry nodes that receive edges)
			var stageNodes []ir.Node
			for _, nodeKey := range stage.Nodes {
				if node, ok := nodeByKey[nodeKey]; ok {
					stageNodes = append(stageNodes, node)
				}
			}
			for entryKey := range entryNodesForStage {
				if node, ok := nodeByKey[entryKey]; ok {
					stageNodes = append(stageNodes, node)
				}
			}

			// Filter edges for this stage:
			// - Edges where source is in this stage (including edges to entry nodes)
			// - Target can be in this stage OR an entry node of another stage
			var stageEdges []ir.Edge
			for _, edge := range edges {
				if stageNodeSet.Contains(edge.Source.Node) {
					stageEdges = append(stageEdges, edge)
				}
			}

			stageStrata, cycleDiag := stratifySubgraph(stageNodes, stageEdges, diag)
			if cycleDiag != nil && !cycleDiag.Ok() {
				return ir.Strata{}, cycleDiag
			}

			sequences[i].Stages[j].Strata = stageStrata
		}
	}

	return globalStrata, nil
}

// stratifySubgraph computes strata for a subgraph of nodes.
// This is the core stratification algorithm without any implicit dependencies.
func stratifySubgraph(
	nodes []ir.Node,
	edges []ir.Edge,
	diag *diagnostics.Diagnostics,
) (ir.Strata, *diagnostics.Diagnostics) {
	if len(nodes) == 0 {
		return ir.Strata{}, nil
	}

	var (
		nodeStrata    = make(map[string]int)
		iterations    = 0
		maxIterations = len(nodes) // Upper bound for DAG
		changed       = true
		maxStratum    = 0
	)

	// Build set of nodes in this subgraph for filtering
	nodeSet := make(set.Set[string])
	for _, node := range nodes {
		nodeSet.Add(node.Key)
	}

	// Step 1: Initialize ALL nodes to stratum 0
	for _, node := range nodes {
		nodeStrata[node.Key] = 0
	}

	// Step 2: Iterative deepening based on dependencies
	// If a node depends on another (within this subgraph), it must be in a higher stratum
	for changed {
		changed = false
		iterations++

		if iterations > maxIterations {
			// Cycle detected - find and report it
			cycle := findCycle(nodes, edges)
			diag.Add(diagnostics.Errorf(nil, "cycle detected in dataflow graph: %v", cycle))
			return ir.Strata{}, diag
		}

		// Process explicit edge dependencies
		for _, edge := range edges {
			// Only consider edges where both source and target are in this subgraph
			if !nodeSet.Contains(edge.Source.Node) || !nodeSet.Contains(edge.Target.Node) {
				continue
			}

			sourceStratum := nodeStrata[edge.Source.Node]
			targetStratum := nodeStrata[edge.Target.Node]
			if sourceStratum >= targetStratum {
				newStratum := sourceStratum + 1
				nodeStrata[edge.Target.Node] = newStratum
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

	return strata, nil
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
