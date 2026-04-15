// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stratifier computes per-scope execution phases for an Arc IR. For
// every parallel scope in the program's Scope tree, the stratifier rewrites
// the scope's Phases so that phase N's members depend only on phases 0..N-1
// under the scope's intra-scope dataflow edges. Sequential scopes carry no
// phasing; the stratifier only recurses into their nested scope members.
package stratifier

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// Stratify walks the Scope tree rooted at prog.Root and assigns phases to
// every parallel scope in depth-first order. Any pre-existing phase layout
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

// stratifyScope dispatches on a scope's mode: parallel scopes are re-phased
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
		for i := range s.Members {
			if s.Members[i].Scope != nil {
				if d := stratifyScope(s.Members[i].Scope, edges, diag); d != nil && !d.Ok() {
					return d
				}
			}
		}
	}
	return diag
}

// stratifyParallel assigns members of a parallel scope to phases using a
// longest-path relaxation over the scope's intra-scope dataflow edges. Any
// previous phasing of the scope is discarded and rebuilt.
func stratifyParallel(
	s *ir.Scope,
	edges []ir.Edge,
	diag *diagnostics.Diagnostics,
) *diagnostics.Diagnostics {
	// Flatten the pre-existing membership. The analyzer populates a single
	// catch-all phase; older constructions may have split their members
	// across phases that no longer reflect dependency order.
	members := make([]ir.Member, 0)
	for _, p := range s.Phases {
		members = append(members, p.Members...)
	}
	members = append(members, s.Members...)
	s.Members = nil
	if len(members) == 0 {
		s.Phases = nil
		return diag
	}

	// ownership maps every node key reachable through the scope's members to
	// the index of the owning member. Nested scopes are treated as atomic:
	// any node they contain (directly or transitively) is owned by the
	// member that wraps them at this level.
	ownership := collectOwnership(members)

	// Longest-path phase assignment. Each member starts at phase 0; for each
	// cross-member edge, push the target's phase past the source's phase.
	// Activation handles on nested gated scopes count as implicit
	// dependencies: the handle's source must run before the scope can
	// activate, so the scope lands after its source. Converges in at most
	// len(members) passes over the constraint set; a failure to converge
	// indicates a cycle.
	phase := make([]int, len(members))
	maxPasses := len(members) + 1
	for pass := 0; pass <= maxPasses; pass++ {
		changed := false
		for _, e := range edges {
			src, srcOK := ownership[e.Source.Node]
			tgt, tgtOK := ownership[e.Target.Node]
			if !srcOK || !tgtOK || src == tgt {
				continue
			}
			if phase[src] >= phase[tgt] {
				phase[tgt] = phase[src] + 1
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
			if phase[src] >= phase[i] {
				phase[i] = phase[src] + 1
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

	// Bucket members by computed phase, preserving source order within each
	// phase. Empty phases are dropped so the resulting slice is dense.
	maxPhase := 0
	for _, p := range phase {
		if p > maxPhase {
			maxPhase = p
		}
	}
	buckets := make([]ir.Phase, maxPhase+1)
	for i, m := range members {
		buckets[phase[i]].Members = append(buckets[phase[i]].Members, m)
	}
	dense := buckets[:0]
	for _, p := range buckets {
		if len(p.Members) > 0 {
			dense = append(dense, p)
		}
	}
	s.Phases = dense

	// Recurse into nested scope members. Transitions and activations are
	// not dataflow edges and do not participate in phasing; they are handled
	// at runtime by the scheduler.
	for pi := range s.Phases {
		for mi := range s.Phases[pi].Members {
			if s.Phases[pi].Members[mi].Scope != nil {
				if d := stratifyScope(s.Phases[pi].Members[mi].Scope, edges, diag); d != nil && !d.Ok() {
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
	if m.NodeRef != nil {
		own[m.NodeRef.Key] = idx
		return
	}
	if m.Scope != nil {
		collectScopeOwnership(*m.Scope, idx, own)
	}
}

// collectScopeOwnership walks a nested scope and attributes every node key
// it contains to the outer owner idx. The nested scope's own phasing does
// not matter at this level of recursion.
func collectScopeOwnership(s ir.Scope, idx int, own map[string]int) {
	for _, p := range s.Phases {
		for _, m := range p.Members {
			if m.NodeRef != nil {
				own[m.NodeRef.Key] = idx
			} else if m.Scope != nil {
				collectScopeOwnership(*m.Scope, idx, own)
			}
		}
	}
	for _, m := range s.Members {
		if m.NodeRef != nil {
			own[m.NodeRef.Key] = idx
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
	if m.Key != "" {
		return m.Key
	}
	if m.NodeRef != nil {
		return m.NodeRef.Key
	}
	if m.Scope != nil {
		return m.Scope.Key
	}
	return "(unknown)"
}
