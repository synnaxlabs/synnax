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

// detectCallCycles detects circular function calls in the call graph and reports
// them as errors. Uses DFS with a recursion stack to find back edges.
func detectCallCycles(edges *[]acontext.CallEdge, diag *diagnostics.Diagnostics) {
	type edgeInfo struct {
		callee   string
		callSite antlr.ParserRuleContext
	}
	graph := make(map[string][]edgeInfo)
	for _, edge := range *edges {
		graph[edge.Caller.Name] = append(graph[edge.Caller.Name], edgeInfo{
			callee:   edge.Callee.Name,
			callSite: edge.CallSite,
		})
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []string
	var cycleSite antlr.ParserRuleContext

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)
		for _, edge := range graph[node] {
			if !visited[edge.callee] {
				if dfs(edge.callee) {
					return true
				}
			} else if recStack[edge.callee] {
				cycleSite = edge.callSite
				cycleStart := -1
				for i, n := range path {
					if n == edge.callee {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					path = append(path[cycleStart:], edge.callee)
				}
				return true
			}
		}
		recStack[node] = false
		path = path[:len(path)-1]
		return false
	}

	for caller := range graph {
		if !visited[caller] {
			path = nil
			clear(recStack)
			if dfs(caller) {
				chain := strings.Join(path, " -> ")
				diag.Add(diagnostics.Errorf(cycleSite, "circular function call: %s", chain))
			}
		}
	}
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
