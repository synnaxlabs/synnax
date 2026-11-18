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
	"slices"

	"github.com/synnaxlabs/arc/analyzer/codes"
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
	prog *ir.IR,
	diag *diagnostics.Diagnostics,
) *diagnostics.Diagnostics {
	if prog == nil || prog.Root.IsZero() {
		return diag
	}
	// Variables hold state like channels do: reads see the held value with no
	// same-pass ordering, and write-read feedback must not stratify as a cycle.
	vars := set.New[string]()
	consts := set.New[string]()
	for _, n := range prog.Nodes {
		switch n.Type {
		case "variable", "stateful_variable":
			vars.Add(n.Key)
		case "constant":
			consts.Add(n.Key)
		}
	}
	ordering := make([]ir.Edge, 0, len(prog.Edges))
	for _, e := range prog.Edges {
		if !vars.Contains(e.Source.Node) && !vars.Contains(e.Target.Node) {
			ordering = append(ordering, e)
		}
	}
	// Stratum 0 dispatch is unconditional and a constant emits whenever
	// dispatched, so an edge-fed constant is floored at stratum 1.
	bumped := set.New[string]()
	for _, e := range prog.Edges {
		if consts.Contains(e.Target.Node) {
			bumped.Add(e.Target.Node)
		}
	}
	return stratifyScope(&prog.Root, ordering, bumped, diag)
}

// stratifyScope dispatches on a scope's mode: parallel scopes are re-stratified
// from the dataflow edges among their members; sequential scopes pass
// through, recursing into nested scope members.
func stratifyScope(
	s *ir.Scope,
	edges []ir.Edge,
	bumped set.Set[string],
	diag *diagnostics.Diagnostics,
) *diagnostics.Diagnostics {
	switch s.Mode {
	case ir.ScopeModeParallel:
		return stratifyParallel(s, edges, bumped, diag)
	case ir.ScopeModeSequential:
		for i := range s.Steps {
			if s.Steps[i].Scope != nil {
				if d := stratifyScope(
					s.Steps[i].Scope,
					edges,
					bumped,
					diag,
				); d != nil &&
					!d.Ok() {
					return d
				}
			}
		}
	case ir.ScopeModeUnspecified:
		panic("scope " + s.Key + " has no mode")
	}
	return diag
}

// stratifyParallel assigns members of a parallel scope to strata using a
// longest-path relaxation over the scope's intra-scope dataflow edges. Any
// previous stratification of the scope is discarded and rebuilt.
func stratifyParallel(
	s *ir.Scope,
	edges []ir.Edge,
	bumped set.Set[string],
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
	stratum, ok := assignStrata(members, edges, ownership, bumped)
	if !ok {
		cycle := findCycle(members, edges, ownership)
		diag.Add(codes.DataflowCycle.Atf(
			nil,
			"cycle detected in dataflow graph within scope '%s': %v",
			s.Key, cycle,
		))
		return diag
	}
	s.Strata = bucketStrata(members, stratum)
	// Recurse into nested scope members. Transitions and activations are not dataflow
	// edges and do not participate in stratification; they are handled at runtime by
	// the scheduler.
	for si := range s.Strata {
		for mi := range s.Strata[si] {
			if s.Strata[si][mi].Scope == nil {
				continue
			}
			if d := stratifyScope(
				s.Strata[si][mi].Scope,
				edges,
				bumped,
				diag,
			); d != nil && !d.Ok() {
				return d
			}
		}
	}
	return diag
}

// assignStrata computes a longest-path stratum for each member. Bumped members start
// at stratum 1. An edge or an activation handle puts its target after its source. It
// returns false when the relaxation does not converge, which means a cycle.
func assignStrata(
	members ir.Members,
	edges []ir.Edge,
	ownership map[string]int,
	bumped set.Set[string],
) ([]int, bool) {
	stratum := make([]int, len(members))
	for i, m := range members {
		if m.NodeKey != nil && bumped.Contains(*m.NodeKey) {
			stratum[i] = 1
		}
	}
	maxPasses := len(members) + 1
	for pass := 0; relaxStrata(members, edges, ownership, stratum); pass++ {
		if pass == maxPasses {
			return nil, false
		}
	}
	return stratum, true
}

// relaxStrata runs one relaxation pass over stratum and reports whether any stratum
// changed.
func relaxStrata(
	members ir.Members,
	edges []ir.Edge,
	ownership map[string]int,
	stratum []int,
) bool {
	changed := false
	raise := func(src, tgt int) {
		if stratum[src] >= stratum[tgt] {
			stratum[tgt] = stratum[src] + 1
			changed = true
		}
	}
	for _, e := range edges {
		if src, tgt, ok := memberEdge(e, ownership); ok {
			raise(src, tgt)
		}
	}
	for i, m := range members {
		if m.Scope == nil {
			continue
		}
		for _, a := range m.Scope.Activations {
			if src, ok := ownership[a.On.Node]; ok && src != i {
				raise(src, i)
			}
		}
	}
	return changed
}

// memberEdge projects e onto the indices of the members that own its ends. It reports
// false when an end has no owner or both ends share one.
func memberEdge(e ir.Edge, ownership map[string]int) (int, int, bool) {
	src, srcOK := ownership[e.Source.Node]
	tgt, tgtOK := ownership[e.Target.Node]
	return src, tgt, srcOK && tgtOK && src != tgt
}

// bucketStrata groups members by stratum, keeping source order within each stratum, and
// drops empty strata other than stratum 0. Stratum 0 stays even when empty: the
// scheduler dispatches index 0 unconditionally, so floored members must not slide back
// into it.
func bucketStrata(members ir.Members, stratum []int) []ir.Members {
	buckets := make([]ir.Members, slices.Max(stratum)+1)
	for i, m := range members {
		buckets[stratum[i]] = append(buckets[stratum[i]], m)
	}
	dense := buckets[:0]
	for i, b := range buckets {
		if len(b) > 0 || i == 0 {
			dense = append(dense, b)
		}
	}
	return dense
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
func findCycle(
	members []ir.Member,
	edges []ir.Edge,
	ownership map[string]int,
) []string {
	f := cycleFinder{
		graph:    make(map[int][]int, len(members)),
		visited:  make(set.Set[int]),
		recStack: make(set.Set[int]),
	}
	for _, e := range edges {
		if src, tgt, ok := memberEdge(e, ownership); ok {
			f.graph[src] = append(f.graph[src], tgt)
		}
	}
	for i := range members {
		if !f.visited.Contains(i) && f.visit(i) {
			break
		}
	}
	labels := make([]string, 0, len(f.cycle))
	for _, idx := range f.cycle {
		labels = append(labels, memberLabel(members[idx]))
	}
	return labels
}

type cycleFinder struct {
	graph    map[int][]int
	visited  set.Set[int]
	recStack set.Set[int]
	path     []int
	cycle    []int
}

// visit runs a depth-first search from n and reports whether it found a cycle.
func (f *cycleFinder) visit(n int) bool {
	f.visited.Add(n)
	f.recStack.Add(n)
	f.path = append(f.path, n)
	for _, nb := range f.graph[n] {
		if !f.visited.Contains(nb) {
			if f.visit(nb) {
				return true
			}
			continue
		}
		if !f.recStack.Contains(nb) {
			continue
		}
		if start := slices.Index(f.path, nb); start >= 0 {
			f.cycle = append(append(f.cycle[:0], f.path[start:]...), nb)
			return true
		}
	}
	f.recStack.Remove(n)
	f.path = f.path[:len(f.path)-1]
	return false
}

func memberLabel(m ir.Member) string {
	if k := m.Key(); k != "" {
		return k
	}
	return "(unknown)"
}
