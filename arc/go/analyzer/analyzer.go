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
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer/constant"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/analyzer/function"
	"github.com/synnaxlabs/arc/analyzer/sequence"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/graph"
	"github.com/synnaxlabs/x/set"
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
// declared after the caller in source order (forward references). When a callee
// accesses channels through chan-typed parameters, the ArgChannels on the edge
// remaps those accesses to the actual channels passed at the call site.
//
// ArgChannels is keyed by param index (position), which is resolved to param symbol
// IDs here since all functions are guaranteed to be fully analyzed at this point.
// O(N * C * E) where N = functions, C = channels, E = call edges.
func propagateCallChannels(edges *[]acontext.CallEdge) {
	type resolvedEdge struct {
		acontext.CallEdge
		paramMap map[int]acontext.ChannelMapping
	}
	resolved := make([]resolvedEdge, len(*edges))
	for i, edge := range *edges {
		resolved[i] = resolvedEdge{
			CallEdge: edge,
			paramMap: expression.ResolveArgChannels(edge.Callee, edge.ArgChannels),
		}
	}
	changed := true
	for changed {
		changed = false
		for _, edge := range resolved {
			for id, name := range edge.Callee.Channels.Read {
				resolvedID, resolvedName := id, name
				if mapping, ok := edge.paramMap[int(id)]; ok {
					resolvedID = mapping.ChannelID
					resolvedName = mapping.ChannelName
				}
				if !lo.HasKey(edge.Caller.Channels.Read, resolvedID) {
					edge.Caller.Channels.Read[resolvedID] = resolvedName
					changed = true
				}
			}
			for id, name := range edge.Callee.Channels.Write {
				resolvedID, resolvedName := id, name
				if mapping, ok := edge.paramMap[int(id)]; ok {
					resolvedID = mapping.ChannelID
					resolvedName = mapping.ChannelName
				}
				if !lo.HasKey(edge.Caller.Channels.Write, resolvedID) {
					edge.Caller.Channels.Write[resolvedID] = resolvedName
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
// O(V + E) for SCC detection, then O(|SCC| * S * K * D) per SCC for exit-path
// analysis where S = statements, K = call sites, D = AST depth.
func detectCallCycles(edges *[]acontext.CallEdge, diag *diagnostics.Diagnostics) {
	callGraph := make(map[string][]callEdgeInfo)
	for _, edge := range *edges {
		callGraph[edge.Caller.Name] = append(callGraph[edge.Caller.Name], callEdgeInfo{
			callee:   edge.Callee.Name,
			callSite: edge.CallSite,
		})
	}

	for _, edges := range callGraph {
		slices.SortFunc(edges, func(a, b callEdgeInfo) int {
			return strings.Compare(a.callee, b.callee)
		})
	}

	adj := make(map[string][]string)
	for name, edgeList := range callGraph {
		for _, e := range edgeList {
			adj[name] = append(adj[name], e.callee)
		}
	}

	sccs := graph.TarjanSCC(adj)

	for _, scc := range sccs {
		sccSet := set.New(scc...)
		if len(scc) == 1 {
			hasSelfLoop := false
			for _, e := range callGraph[scc[0]] {
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
				if e.Caller.Name == node && sccSet.Contains(e.Callee.Name) {
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
			cycle, closingSite := findCycleInSCC(scc, sccSet, callGraph)
			chain := strings.Join(cycle, " -> ")
			diag.Add(diagnostics.Errorf(closingSite, "circular function call: %s", chain))
		}
	}
}

// findCycleInSCC finds a simple cycle within the SCC for error reporting and returns
// the cycle as a list of names (ending with the start node) and the closing call site.
// O(|SCC| + edges within SCC). Only called for error SCCs.
func findCycleInSCC(scc []string, sccSet set.Set[string], graph map[string][]callEdgeInfo) ([]string, antlr.ParserRuleContext) {
	start := scc[0]

	if len(scc) == 1 {
		for _, e := range graph[start] {
			if e.callee == start {
				return []string{start, start}, e.callSite
			}
		}
		return []string{start, start}, nil
	}

	visited := make(set.Set[string])
	var path []string
	var closingSite antlr.ParserRuleContext

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited.Add(node)
		path = append(path, node)
		for _, e := range graph[node] {
			if !sccSet.Contains(e.callee) {
				continue
			}
			if e.callee == start && len(path) > 1 {
				path = append(path, start)
				closingSite = e.callSite
				return true
			}
			if !visited.Contains(e.callee) {
				if dfs(e.callee) {
					return true
				}
			}
		}
		path = path[:len(path)-1]
		visited.Remove(node)
		return false
	}

	dfs(start)
	return path, closingSite
}

// blockAlwaysCalls returns true if every execution path through the block reaches at
// least one of the given call sites. Mirrors blockAlwaysReturns in function.go.
// O(S * K * D) where S = statements, K = call sites, D = AST depth.
func blockAlwaysCalls(block parser.IBlockContext, sites []antlr.ParserRuleContext) bool {
	if block == nil {
		return false
	}
	for _, stmt := range block.AllStatement() {
		if stmt.ReturnStatement() != nil {
			return false
		}
		if ifStmt := stmt.IfStatement(); ifStmt != nil {
			if function.IfStmtAlwaysReturns(ifStmt) {
				return false
			}
			if ifAlwaysCalls(ifStmt, sites) {
				return true
			}
			if ifSometimesReturns(ifStmt) {
				return false
			}
			continue
		}
		if stmtContainsSite(stmt, sites) {
			return true
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

// ifSometimesReturns returns true if any branch of the if-statement contains a return,
// meaning some execution paths exit early and subsequent statements are not on all paths.
func ifSometimesReturns(ifStmt parser.IIfStatementContext) bool {
	if function.BlockAlwaysReturns(ifStmt.Block()) {
		return true
	}
	for _, elseIf := range ifStmt.AllElseIfClause() {
		if function.BlockAlwaysReturns(elseIf.Block()) {
			return true
		}
	}
	if ifStmt.ElseClause() != nil {
		return function.BlockAlwaysReturns(ifStmt.ElseClause().Block())
	}
	return false
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

// isDescendant returns true if child is a descendant of ancestor in the AST. O(D).
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
// returns its body block. O(D).
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
