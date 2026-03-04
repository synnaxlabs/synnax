// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/constant"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/analyzer/function"
	"github.com/synnaxlabs/arc/analyzer/sequence"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/diagnostics"
)

func AnalyzeProgram(ctx acontext.Context[parser.IProgramContext]) {
	collectDeclarations(ctx)
	analyzeDeclarations(ctx)
	propagateCallChannels(ctx.CallEdges)
	detectCallCycles(ctx.CallEdges, ctx.Diagnostics)
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			addUnificationError(ctx.Diagnostics, err, ctx.AST)
			return
		}
		applyTypeSubstitutionsToSymbols(ctx, ctx.Scope)
		substituteTypeMap(ctx)
	}
}

func substituteTypeMap(ctx acontext.Context[parser.IProgramContext]) {
	for node, typ := range ctx.TypeMap {
		ctx.TypeMap[node] = ctx.Constraints.ApplySubstitutions(typ)
	}
}

func collectDeclarations(ctx acontext.Context[parser.IProgramContext]) {
	constant.CollectDeclarations(ctx)
	function.CollectDeclarations(ctx)
	sequence.CollectDeclarations(ctx)
}

// propagateCallChannels runs a fixpoint loop over recorded call edges to ensure
// that callee channel accesses are propagated to callers even when the callee is
// declared after the caller in source order (forward references).
func propagateCallChannels(edges *[]acontext.CallEdge) {
	changed := true
	for changed {
		changed = false
		for _, edge := range *edges {
			for id, name := range edge.Callee.Channels.Read {
				if !edge.Caller.Channels.Read.Contains(id) {
					edge.Caller.Channels.Read[id] = name
					changed = true
				}
			}
			for id, name := range edge.Callee.Channels.Write {
				if !edge.Caller.Channels.Write.Contains(id) {
					edge.Caller.Channels.Write[id] = name
					changed = true
				}
			}
		}
	}
}

type callEdgeInfo struct {
	callee   string
	callSite antlr.ParserRuleContext
}

// detectCallCycles finds strongly connected components in the call graph and reports
// an error for any SCC where no node has an exit path. A node is "safe" if its
// function body has at least one execution path that calls no other SCC member.
func detectCallCycles(edges *[]acontext.CallEdge, diag *diagnostics.Diagnostics) {
	graph := make(map[string][]callEdgeInfo)
	for _, edge := range *edges {
		graph[edge.Caller.Name] = append(graph[edge.Caller.Name], callEdgeInfo{
			callee:   edge.Callee.Name,
			callSite: edge.CallSite,
		})
	}

	adj := make(map[string][]string)
	for name, edgeList := range graph {
		for _, e := range edgeList {
			adj[name] = append(adj[name], e.callee)
		}
	}

	sccs := tarjanSCC(adj)

	for _, scc := range sccs {
		sccSet := make(map[string]bool)
		for _, name := range scc {
			sccSet[name] = true
		}
		if len(scc) == 1 {
			hasSelfLoop := false
			for _, e := range graph[scc[0]] {
				if e.callee == scc[0] {
					hasSelfLoop = true
					break
				}
			}
			if !hasSelfLoop {
				continue
			}
		}

		anySafe := false
		for _, node := range scc {
			var sites []antlr.ParserRuleContext
			for _, e := range *edges {
				if e.Caller.Name == node && sccSet[e.Callee.Name] {
					sites = append(sites, e.CallSite)
				}
			}
			if len(sites) == 0 {
				anySafe = true
				break
			}
			body := findFuncBody(sites[0])
			if body == nil {
				anySafe = true
				break
			}
			if !blockAlwaysCalls(body, sites) {
				anySafe = true
				break
			}
		}

		if !anySafe {
			cycle, closingSite := findCycleInSCC(scc, sccSet, graph)
			chain := strings.Join(cycle, " -> ")
			diag.Add(diagnostics.Errorf(closingSite, "circular function call: %s", chain))
		}
	}
}

// tarjanSCC returns all strongly connected components of the directed graph.
func tarjanSCC(adj map[string][]string) [][]string {
	var (
		idx      int
		stack    []string
		onStack  = make(map[string]bool)
		indices  = make(map[string]int)
		lowlinks = make(map[string]int)
		defined  = make(map[string]bool)
		sccs     [][]string
	)
	var strongconnect func(v string)
	strongconnect = func(v string) {
		indices[v] = idx
		lowlinks[v] = idx
		idx++
		defined[v] = true
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range adj[v] {
			if !defined[w] {
				strongconnect(w)
				if lowlinks[w] < lowlinks[v] {
					lowlinks[v] = lowlinks[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlinks[v] {
					lowlinks[v] = indices[w]
				}
			}
		}
		if lowlinks[v] == indices[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}
	for v := range adj {
		if !defined[v] {
			strongconnect(v)
		}
	}
	return sccs
}

// findCycleInSCC finds a simple cycle within the SCC for error reporting and returns
// the cycle as a list of names (ending with the start node) and the closing call site.
func findCycleInSCC(scc []string, sccSet map[string]bool, graph map[string][]callEdgeInfo) ([]string, antlr.ParserRuleContext) {
	start := scc[0]

	if len(scc) == 1 {
		for _, e := range graph[start] {
			if e.callee == start {
				return []string{start, start}, e.callSite
			}
		}
		return []string{start, start}, nil
	}

	visited := make(map[string]bool)
	var path []string
	var closingSite antlr.ParserRuleContext

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		path = append(path, node)
		for _, e := range graph[node] {
			if !sccSet[e.callee] {
				continue
			}
			if e.callee == start && len(path) > 1 {
				path = append(path, start)
				closingSite = e.callSite
				return true
			}
			if !visited[e.callee] {
				if dfs(e.callee) {
					return true
				}
			}
		}
		path = path[:len(path)-1]
		visited[node] = false
		return false
	}

	dfs(start)
	return path, closingSite
}

// blockAlwaysCalls returns true if every execution path through the block reaches at
// least one of the given call sites. Mirrors blockAlwaysReturns in function.go.
func blockAlwaysCalls(block parser.IBlockContext, sites []antlr.ParserRuleContext) bool {
	if block == nil {
		return false
	}
	statements := block.AllStatement()
	for i := len(statements) - 1; i >= 0; i-- {
		stmt := statements[i]
		if stmt.IfStatement() == nil && stmtContainsSite(stmt, sites) {
			return true
		}
		if ifStmt := stmt.IfStatement(); ifStmt != nil {
			if ifAlwaysCalls(ifStmt, sites) {
				return true
			}
		}
	}
	return false
}

// ifAlwaysCalls returns true if all branches of the if-statement always reach a call
// site. Mirrors ifStmtAlwaysReturns in function.go.
func ifAlwaysCalls(ifStmt parser.IIfStatementContext, sites []antlr.ParserRuleContext) bool {
	if ifStmt.ElseClause() == nil || !blockAlwaysCalls(ifStmt.Block(), sites) {
		return false
	}
	for _, elseIf := range ifStmt.AllElseIfClause() {
		if !blockAlwaysCalls(elseIf.Block(), sites) {
			return false
		}
	}
	return blockAlwaysCalls(ifStmt.ElseClause().Block(), sites)
}

// stmtContainsSite checks if any call site is a descendant of the given statement.
func stmtContainsSite(stmt parser.IStatementContext, sites []antlr.ParserRuleContext) bool {
	for _, site := range sites {
		if isDescendant(site, stmt) {
			return true
		}
	}
	return false
}

// isDescendant returns true if child is a descendant of ancestor in the AST.
func isDescendant(child, ancestor antlr.Tree) bool {
	node := child.GetParent()
	for node != nil {
		if node == ancestor {
			return true
		}
		node = node.GetParent()
	}
	return false
}

// findFuncBody walks up from a call site to the enclosing function declaration and
// returns its body block.
func findFuncBody(callSite antlr.ParserRuleContext) parser.IBlockContext {
	node := callSite.GetParent()
	for node != nil {
		if funcDecl, ok := node.(parser.IFunctionDeclarationContext); ok {
			return funcDecl.Block()
		}
		node = node.GetParent()
	}
	return nil
}

func analyzeDeclarations(ctx acontext.Context[parser.IProgramContext]) {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
			function.Analyze(acontext.Child(ctx, funcDecl))
		} else if flowStmt := item.FlowStatement(); flowStmt != nil {
			flow.Analyze(acontext.Child(ctx, flowStmt))
		} else if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			sequence.Analyze(acontext.Child(ctx, seqDecl))
		}
	}
}

func AnalyzeStatement(ctx acontext.Context[parser.IStatementContext]) {
	statement.Analyze(ctx)
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			addUnificationError(ctx.Diagnostics, err, ctx.AST)
			return
		}
		applyTypeSubstitutionsToSymbols(ctx, ctx.Scope)
	}
}

func AnalyzeBlock(ctx acontext.Context[parser.IBlockContext]) {
	statement.AnalyzeBlock(ctx)
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			addUnificationError(ctx.Diagnostics, err, ctx.AST)
			return
		}
		applyTypeSubstitutionsToSymbols(ctx, ctx.Scope)
	}
}

func applyTypeSubstitutionsToSymbols[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	scope *symbol.Scope,
) {
	if scope.Type.IsValid() {
		scope.Type = ctx.Constraints.ApplySubstitutions(scope.Type)
	}
	for _, child := range scope.Children {
		applyTypeSubstitutionsToSymbols[T](ctx, child)
	}
}

func addUnificationError(
	diag *diagnostics.Diagnostics,
	err error,
	fallbackCtx antlr.ParserRuleContext,
) {
	if ue, ok := err.(*constraints.UnificationError); ok && ue.Constraint != nil && ue.Constraint.Source != nil {
		diag.Add(diagnostics.Error(err, ue.Constraint.Source))
	} else {
		diag.Add(diagnostics.Error(err, fallbackCtx))
	}
}
