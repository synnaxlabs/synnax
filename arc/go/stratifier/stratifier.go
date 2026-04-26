// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stratifier computes per-scope execution strata for an Arc IR. For
// every parallel scope in the program's Scope tree, the stratifier rewrites
// the scope's Strata so that stratum N's members depend only on strata 0..N-1
// under the scope's intra-scope dataflow edges. Sequential scopes carry no
// strata; the stratifier only recurses into their nested scope members.
package stratifier

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// Stratify walks the Scope tree rooted at prog.Root and assigns strata to
// every parallel scope in depth-first order. Any pre-existing strata layout
// on a parallel scope is discarded in favor of the freshly-computed one.
// The program is modified in place. Diagnostics are appended to diag; the
// returned value is diag for convenient chaining.
func Stratify(
	_ context.Context,
	prog *ir.IR,
	diag *diagnostics.Diagnostics,
) *diagnostics.Diagnostics {
	if prog == nil || prog.Root.IsZero() {
		return diag
	}
	return stratifyScope(&prog.Root, prog.Edges, diag)
}

// stratifyScope dispatches on a scope's mode: parallel scopes are re-stratified
// from the dataflow edges among their members; sequential scopes pass
// through, recursing into nested scope members.
func stratifyScope(
	s *ir.Scope,
	edges []ir.Edge,
	diag *diagnostics.Diagnostics,
) *diagnostics.Diagnostics {
	switch s.Mode {
	case ir.ScopeModeParallel:
		return stratifyParallel(s, edges, diag)
	case ir.ScopeModeSequential:
		for i := range s.Steps {
			if s.Steps[i].Scope != nil {
				if d := stratifyScope(s.Steps[i].Scope, edges, diag); d != nil && !d.Ok() {
					return d
				}
			}
		}
	}
	return diag
}

// stratifyParallel assigns members of a parallel scope to strata using a
// longest-path relaxation over the scope's intra-scope dataflow edges. Any
// previous stratification of the scope is discarded and rebuilt.
func stratifyParallel(
	s *ir.Scope,
	edges []ir.Edge,
	diag *diagnostics.Diagnostics,
) *diagnostics.Diagnostics {
	// Flatten the pre-existing membership. The analyzer populates a single
	// catch-all stratum; older constructions may have split their members
	// across strata that no longer reflect dependency order.
	members := make(ir.Members, 0)
	for _, stratum := range s.Strata {
		members = append(members, stratum...)
	}
	members = append(members, s.Steps...)
	s.Steps = nil
	if len(members) == 0 {
		s.Strata = nil
		return diag
	}

	// ownership maps every node key reachable through the scope's members to
	// the index of the owning member. Nested scopes are treated as atomic:
	// any node they contain (directly or transitively) is owned by the
	// member that wraps them at this level.
	ownership := collectOwnership(members)

	// Longest-path stratum assignment. Each member starts at stratum 0; for
	// each cross-member edge, push the target's stratum past the source's.
	// Activation handles on nested gated scopes count as implicit
	// dependencies: the handle's source must run before the scope can
	// activate, so the scope lands after its source. Converges in at most
	// len(members) passes over the constraint set; a failure to converge
	// indicates a cycle.
	stratum := make([]int, len(members))
	maxPasses := len(members) + 1
	for pass := 0; pass <= maxPasses; pass++ {
		changed := false
		for _, e := range edges {
			src, srcOK := ownership[e.Source.Node]
			tgt, tgtOK := ownership[e.Target.Node]
			if !srcOK || !tgtOK || src == tgt {
				continue
			}
			if stratum[src] >= stratum[tgt] {
				stratum[tgt] = stratum[src] + 1
				changed = true
			}
		}
		for i, m := range members {
			if m.Scope == nil || m.Scope.Activation == nil {
				continue
			}
			src, ok := ownership[m.Scope.Activation.Node]
			if !ok || src == i {
				continue
			}
			if stratum[src] >= stratum[i] {
				stratum[i] = stratum[src] + 1
				changed = true
			}
		}
		if !changed {
			break
		}
		if pass == maxPasses {
			cycle := findCycle(members, edges, ownership)
			diag.Add(diagnostics.Errorf(
				nil,
				"cycle detected in dataflow graph within scope '%s': %v",
				s.Key, cycle,
			))
			return diag
		}
	}

	// Bucket members by computed stratum, preserving source order within
	// each stratum. Empty strata are dropped so the resulting slice is dense.
	maxStratum := 0
	for _, p := range stratum {
		if p > maxStratum {
			maxStratum = p
		}
	}
	buckets := make([]ir.Members, maxStratum+1)
	for i, m := range members {
		buckets[stratum[i]] = append(buckets[stratum[i]], m)
	}
	dense := buckets[:0]
	for _, b := range buckets {
		if len(b) > 0 {
			dense = append(dense, b)
		}
	}
	s.Strata = dense

	// Recurse into nested scope members. Transitions and activations are
	// not dataflow edges and do not participate in stratification; they are
	// handled at runtime by the scheduler.
	for si := range s.Strata {
		for mi := range s.Strata[si] {
			if s.Strata[si][mi].Scope != nil {
				if d := stratifyScope(s.Strata[si][mi].Scope, edges, diag); d != nil && !d.Ok() {
					return d
				}
			}
		}
	}
	return diag
}

// collectOwnership builds a map from every node key reachable through the
// given members (directly or through nested scopes) to the index of the
// member that owns it.
func collectOwnership(members []ir.Member) map[string]int {
	own := make(map[string]int)
	for i, m := range members {
		collectMemberOwnership(m, i, own)
	}
	return own
}

func collectMemberOwnership(m ir.Member, idx int, own map[string]int) {
	if m.NodeKey != nil {
		own[*m.NodeKey] = idx
		return
	}
	if m.Scope != nil {
		collectScopeOwnership(*m.Scope, idx, own)
	}
}

// collectScopeOwnership walks a nested scope and attributes every node key
// it contains to the outer owner idx. The nested scope's own stratification
// does not matter at this level of recursion.
func collectScopeOwnership(s ir.Scope, idx int, own map[string]int) {
	for _, stratum := range s.Strata {
		for _, m := range stratum {
			if m.NodeKey != nil {
				own[*m.NodeKey] = idx
			} else if m.Scope != nil {
				collectScopeOwnership(*m.Scope, idx, own)
			}
		}
	}
	for _, m := range s.Steps {
		if m.NodeKey != nil {
			own[*m.NodeKey] = idx
		} else if m.Scope != nil {
			collectScopeOwnership(*m.Scope, idx, own)
		}
	}
}

// findCycle returns a list of member keys that form a cycle among the
// given members under the ownership-projected edge set. Used only to
// produce a human-readable diagnostic after the relaxation loop fails to
// converge.
func findCycle(members []ir.Member, edges []ir.Edge, ownership map[string]int) []string {
	graph := make(map[int][]int, len(members))
	for _, e := range edges {
		src, srcOK := ownership[e.Source.Node]
		tgt, tgtOK := ownership[e.Target.Node]
		if !srcOK || !tgtOK || src == tgt {
			continue
		}
		graph[src] = append(graph[src], tgt)
	}
	var (
		visited  = make(set.Set[int])
		recStack = make(set.Set[int])
		path     []int
		cycle    []int
	)
	var dfs func(int) bool
	dfs = func(n int) bool {
		visited.Add(n)
		recStack.Add(n)
		path = append(path, n)
		for _, nb := range graph[n] {
			if !visited.Contains(nb) {
				if dfs(nb) {
					return true
				}
			} else if recStack.Contains(nb) {
				start := -1
				for i, v := range path {
					if v == nb {
						start = i
						break
					}
				}
				if start >= 0 {
					cycle = append(append(cycle[:0], path[start:]...), nb)
					return true
				}
			}
		}
		recStack.Remove(n)
		path = path[:len(path)-1]
		return false
	}
	for i := range members {
		if !visited.Contains(i) {
			if dfs(i) {
				break
			}
		}
	}
	labels := make([]string, 0, len(cycle))
	for _, idx := range cycle {
		labels = append(labels, memberLabel(members[idx]))
	}
	return labels
}

func memberLabel(m ir.Member) string {
	if k := m.Key(); k != "" {
		return k
	}
	return "(unknown)"
}
