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
	"fmt"
	"slices"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// entryKey computes the entry node key for a step using the established naming
// convention. This must match the key generation in text.KeyGenerator.Entry.
func entryKey(seqName, stepKey string) string {
	return "entry_" + seqName + "_" + stepKey
}

// BoundaryKey computes the synthetic boundary key for an execution context
// (stage or sequence step) within its parent's strata.
func BoundaryKey(stepKey string) string {
	return fmt.Sprintf("boundary_%s", stepKey)
}

// Stratify computes execution strata for nodes in a dataflow graph using a
// multi-tier stratification model:
//
//  1. Global strata: Nodes outside any sequence/stage
//  2. Per-stage strata: Reactive flow nodes stratified by data dependencies
//  3. Per-sequence strata: All flow step nodes stratified together, with
//     execution context boundaries for stage/sequence steps
//
// Returns global strata and diagnostics. Per-step strata are populated directly
// into the sequences parameter (modified in place).
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

	nodeByKey := make(map[string]ir.Node)
	for _, node := range nodes {
		nodeByKey[node.Key] = node
	}

	// Collect all nodes owned by any execution context (stages, flow steps).
	// Also collect all entry node keys.
	ownedNodes := make(set.Set[string])
	allEntryNodes := make(set.Set[string])
	for _, seq := range sequences {
		collectOwnedNodes(seq, &ownedNodes, &allEntryNodes)
	}

	// Step 1: Stratify global nodes (not owned by any execution context).
	entryNodesWithGlobalInput := make(set.Set[string])
	var globalNodes []ir.Node
	globalNodeSet := make(set.Set[string])
	for _, node := range nodes {
		if !ownedNodes.Contains(node.Key) {
			globalNodes = append(globalNodes, node)
			globalNodeSet.Add(node.Key)
		}
	}
	for _, edge := range edges {
		if globalNodeSet.Contains(edge.Source.Node) && allEntryNodes.Contains(edge.Target.Node) {
			entryNodesWithGlobalInput.Add(edge.Target.Node)
		}
	}
	var filteredGlobalNodes []ir.Node
	filteredGlobalNodeSet := make(set.Set[string])
	for _, node := range globalNodes {
		if allEntryNodes.Contains(node.Key) && !entryNodesWithGlobalInput.Contains(node.Key) {
			continue
		}
		filteredGlobalNodes = append(filteredGlobalNodes, node)
		filteredGlobalNodeSet.Add(node.Key)
	}
	var globalEdges []ir.Edge
	for _, edge := range edges {
		if filteredGlobalNodeSet.Contains(edge.Source.Node) {
			globalEdges = append(globalEdges, edge)
		}
	}
	globalStrata, cycleDiag := stratifySubgraph(filteredGlobalNodes, globalEdges, diag)
	if cycleDiag != nil && !cycleDiag.Ok() {
		return ir.Strata{}, cycleDiag
	}

	// Step 2: Stratify each sequence recursively.
	for i := range sequences {
		if err := stratifySequence(&sequences[i], nodeByKey, edges, allEntryNodes, diag); err != nil {
			return ir.Strata{}, err
		}
	}

	return globalStrata, nil
}

// collectOwnedNodes recursively marks all nodes owned by stages and flow steps,
// and collects entry node keys.
func collectOwnedNodes(seq ir.Sequence, owned *set.Set[string], entryNodes *set.Set[string]) {
	for _, step := range seq.Steps {
		entryNodes.Add(entryKey(seq.Key, step.Key))
		switch {
		case step.Stage != nil:
			for _, nodeKey := range step.Stage.Nodes {
				owned.Add(nodeKey)
			}
			for _, subSeq := range step.Stage.Sequences {
				collectOwnedNodes(subSeq, owned, entryNodes)
			}
		case step.Flow != nil:
			for _, nodeKey := range step.Flow.Nodes {
				owned.Add(nodeKey)
			}
		case step.Sequence != nil:
			collectOwnedNodes(*step.Sequence, owned, entryNodes)
		}
	}
}

// stratifySequence computes strata for a sequence and its children.
func stratifySequence(
	seq *ir.Sequence,
	nodeByKey map[string]ir.Node,
	edges []ir.Edge,
	allEntryNodes set.Set[string],
	diag *diagnostics.Diagnostics,
) *diagnostics.Diagnostics {
	hasFlowSteps := false
	for _, step := range seq.Steps {
		if step.Flow != nil {
			hasFlowSteps = true
			break
		}
	}

	if hasFlowSteps {
		if err := stratifySequenceWithFlowSteps(seq, nodeByKey, edges, allEntryNodes, diag); err != nil {
			return err
		}
	}

	// Recurse into stage and sequence steps.
	for i, step := range seq.Steps {
		if step.Stage != nil {
			if err := stratifyStage(&seq.Steps[i], seq.Key, nodeByKey, edges, allEntryNodes, diag); err != nil {
				return err
			}
		}
		if step.Sequence != nil {
			if err := stratifySequence(step.Sequence, nodeByKey, edges, allEntryNodes, diag); err != nil {
				return err
			}
		}
	}

	return nil
}

// stratifySequenceWithFlowSteps computes the sequence-level strata for sequences
// that contain flow steps. All flow step nodes are stratified together into
// seq.Strata. Stage/sequence steps appear as synthetic boundary keys.
func stratifySequenceWithFlowSteps(
	seq *ir.Sequence,
	nodeByKey map[string]ir.Node,
	edges []ir.Edge,
	allEntryNodes set.Set[string],
	diag *diagnostics.Diagnostics,
) *diagnostics.Diagnostics {
	// Collect flow step data nodes for stratification. Entry nodes are
	// deliberately excluded: step 0's entry lives in the parent (global)
	// strata, and entries for steps > 0 must not be placed in stratum 0
	// (which always executes) — otherwise they would fire on every tick of
	// the active sequence and prematurely transition to a later step. They
	// are appended in their own stratum after the data nodes so they only
	// execute when an upstream conditional edge marks them as changed.
	seqNodeSet := make(set.Set[string])
	var seqNodes []ir.Node
	for _, step := range seq.Steps {
		if step.Flow != nil {
			for _, nodeKey := range step.Flow.Nodes {
				if node, ok := nodeByKey[nodeKey]; ok {
					seqNodes = append(seqNodes, node)
					seqNodeSet.Add(nodeKey)
				}
			}
		}
	}

	// Filter edges for this sequence's subgraph.
	var seqEdges []ir.Edge
	for _, edge := range edges {
		if seqNodeSet.Contains(edge.Source.Node) {
			seqEdges = append(seqEdges, edge)
		}
	}

	seqStrata, cycleDiag := stratifySubgraph(seqNodes, seqEdges, diag)
	if cycleDiag != nil && !cycleDiag.Ok() {
		return cycleDiag
	}

	// Insert boundary keys for non-flow steps at stratum 0 (definition order).
	// Boundary keys don't participate in data dependency stratification. They
	// are positioned at stratum 0 alongside other source nodes.
	for _, step := range seq.Steps {
		if step.Stage != nil || step.Sequence != nil {
			bk := BoundaryKey(step.Key)
			if len(seqStrata) == 0 {
				seqStrata = ir.Strata{{bk}}
			} else {
				seqStrata[0] = append(seqStrata[0], bk)
			}
		}
	}

	// Append a trailing stratum containing entry nodes for steps > 0. These
	// only execute when explicitly marked changed by an upstream conditional
	// edge from the prior step's transition.
	var entryStratum []string
	for i, step := range seq.Steps {
		if i == 0 {
			continue
		}
		ek := entryKey(seq.Key, step.Key)
		if _, ok := nodeByKey[ek]; ok {
			entryStratum = append(entryStratum, ek)
		}
	}
	if len(entryStratum) > 0 {
		seqStrata = append(seqStrata, entryStratum)
	}

	seq.Strata = seqStrata
	return nil
}

// stratifyStage computes per-stage strata for a stage step.
func stratifyStage(
	step *ir.Step,
	seqName string,
	nodeByKey map[string]ir.Node,
	edges []ir.Edge,
	allEntryNodes set.Set[string],
	diag *diagnostics.Diagnostics,
) *diagnostics.Diagnostics {
	stage := step.Stage

	stageNodeSet := make(set.Set[string])
	for _, nodeKey := range stage.Nodes {
		stageNodeSet.Add(nodeKey)
	}

	orderedStageKeys := append([]string(nil), stage.Nodes...)
	orderedStageSet := make(set.Set[string])
	for _, key := range orderedStageKeys {
		orderedStageSet.Add(key)
	}
	entryInsertAnchor := make(map[string]string)

	entryNodesForStage := make(set.Set[string])
	for _, edge := range edges {
		if stageNodeSet.Contains(edge.Source.Node) && allEntryNodes.Contains(edge.Target.Node) {
			entryNodesForStage.Add(edge.Target.Node)
			if !orderedStageSet.Contains(edge.Target.Node) {
				anchor := edge.Source.Node
				if last, ok := entryInsertAnchor[edge.Source.Node]; ok {
					anchor = last
				}
				idx := slices.Index(orderedStageKeys, anchor)
				if idx == -1 {
					orderedStageKeys = append(orderedStageKeys, edge.Target.Node)
				} else {
					orderedStageKeys = slices.Insert(orderedStageKeys, idx+1, edge.Target.Node)
				}
				orderedStageSet.Add(edge.Target.Node)
				entryInsertAnchor[edge.Source.Node] = edge.Target.Node
			}
		}
	}

	for ek := range entryNodesForStage {
		stageNodeSet.Add(ek)
	}

	var stageNodes []ir.Node
	for _, nodeKey := range orderedStageKeys {
		if node, ok := nodeByKey[nodeKey]; ok {
			stageNodes = append(stageNodes, node)
		}
	}

	var stageEdges []ir.Edge
	for _, edge := range edges {
		if stageNodeSet.Contains(edge.Source.Node) {
			stageEdges = append(stageEdges, edge)
		}
	}

	stageStrata, cycleDiag := stratifySubgraph(stageNodes, stageEdges, diag)
	if cycleDiag != nil && !cycleDiag.Ok() {
		return cycleDiag
	}

	stageStrata = flattenEntryNodes(stageStrata, entryNodesForStage, orderedStageKeys)

	// Insert boundary keys for inline sub-sequences at stratum 0.
	for _, subSeq := range stage.Sequences {
		bk := BoundaryKey(subSeq.Key)
		if len(stageStrata) == 0 {
			stageStrata = ir.Strata{{bk}}
		} else {
			stageStrata[0] = append(stageStrata[0], bk)
		}
	}

	step.Stage.Strata = stageStrata

	// Recurse into sub-sequences.
	for i := range stage.Sequences {
		if err := stratifySequence(&stage.Sequences[i], nodeByKey, edges, allEntryNodes, diag); err != nil {
			return err
		}
	}

	return nil
}

// stratifySubgraph computes strata for a subgraph of nodes.
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
		maxIterations = len(nodes)
		changed       = true
		maxStratum    = 0
	)

	nodeSet := make(set.Set[string])
	for _, node := range nodes {
		nodeSet.Add(node.Key)
	}

	for _, node := range nodes {
		nodeStrata[node.Key] = 0
	}

	for changed {
		changed = false
		iterations++

		if iterations > maxIterations {
			cycle := findCycle(nodes, edges)
			diag.Add(diagnostics.Errorf(nil, "cycle detected in dataflow graph: %v", cycle))
			return ir.Strata{}, diag
		}

		for _, edge := range edges {
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

	strata := make(ir.Strata, maxStratum+1)
	for _, node := range nodes {
		stratum := nodeStrata[node.Key]
		strata[stratum] = append(strata[stratum], node.Key)
	}

	return strata, nil
}

func findCycle(nodes []ir.Node, edges []ir.Edge) []string {
	var (
		graph    = make(map[string][]string)
		visited  = make(set.Set[string])
		recStack = make(set.Set[string])
		path     []string
		dfs      func(node string) bool
	)
	for _, edge := range edges {
		graph[edge.Source.Node] = append(graph[edge.Source.Node], edge.Target.Node)
	}
	dfs = func(node string) bool {
		visited.Add(node)
		recStack.Add(node)
		path = append(path, node)

		for _, neighbor := range graph[node] {
			if !visited.Contains(neighbor) {
				if dfs(neighbor) {
					return true
				}
			} else if recStack.Contains(neighbor) {
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

		recStack.Remove(node)
		path = path[:len(path)-1]
		return false
	}

	for _, node := range nodes {
		if !visited.Contains(node.Key) {
			if dfs(node.Key) {
				return path
			}
		}
	}

	return []string{"unknown cycle"}
}

func flattenEntryNodes(
	strata ir.Strata,
	entryNodes set.Set[string],
	orderedKeys []string,
) ir.Strata {
	if len(entryNodes) <= 1 {
		return strata
	}
	maxStratum := -1
	for i, stratum := range strata {
		for _, key := range stratum {
			if entryNodes.Contains(key) && i > maxStratum {
				maxStratum = i
			}
		}
	}
	if maxStratum == -1 {
		return strata
	}
	for i, stratum := range strata {
		filtered := stratum[:0]
		for _, key := range stratum {
			if !entryNodes.Contains(key) {
				filtered = append(filtered, key)
			}
		}
		strata[i] = filtered
	}
	for _, key := range orderedKeys {
		if entryNodes.Contains(key) {
			strata[maxStratum] = append(strata[maxStratum], key)
		}
	}
	result := strata[:0]
	for _, stratum := range strata {
		if len(stratum) > 0 {
			result = append(result, stratum)
		}
	}
	return result
}
