// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unused

import (
	stdcontext "context"
	"strings"

	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// Sequence, stage, and function reachability
// ==========================================
//
// A single forward reachability analysis covers ARC5102 (uncalled function),
// ARC5201 (unreachable stage), and ARC5202 (unstarted sequence). Nodes in
// the activation graph are sequence, stage, and function scopes; the nil
// sentinel is a virtual root representing module-scope code. Edges are:
//
//   nil        -> sequence     (top-level `=> sequence`)
//   nil        -> function     (top-level flow invoking `function{}`)
//   sequence   -> entry stage  (implicit; entering a sequence activates its
//                               first stage in source order)
//   stage      -> stage        (intra-sequence `=>` and `=> next`)
//   stage      -> sequence     (cross-sequence activation from a stage)
//   stage      -> function     (stage body invoking `function{}`)
//   function   -> function     (expression-level call, from ctx.CallEdges)
//
// A BFS from the virtual root visits exactly the scopes whose bodies can
// actually execute. A declared sequence, stage, or function that is not
// visited is diagnosed under its respective code.

// analyzeReachability emits ARC5102, ARC5201, and ARC5202 by performing a
// single forward reachability pass over the combined activation graph.
func analyzeReachability(ctx context.Context[parser.IProgramContext]) {
	edges := collectReachabilityEdges(ctx)
	reached := reachableFromRoot(edges)
	emitReachabilityDiagnostics(ctx, reached)
}

// collectReachabilityEdges walks the AST (and ctx.CallEdges) and builds the
// combined activation graph. The nil key represents the virtual root.
func collectReachabilityEdges(
	ctx context.Context[parser.IProgramContext],
) map[*symbol.Scope]set.Set[*symbol.Scope] {
	edges := map[*symbol.Scope]set.Set[*symbol.Scope]{}
	addEdge := func(src, dst *symbol.Scope) {
		if dst == nil {
			return
		}
		s, ok := edges[src]
		if !ok {
			s = set.New[*symbol.Scope]()
			edges[src] = s
		}
		s.Add(dst)
	}

	// Function-to-function calls are already collected during expression
	// analysis for channel propagation; reuse them.
	for _, edge := range *ctx.CallEdges {
		addEdge(edge.Caller, edge.Callee)
	}

	for _, item := range ctx.AST.AllTopLevelItem() {
		if flowStmt := item.FlowStatement(); flowStmt != nil {
			walkFlowOutputs(ctx.Context, ctx.Scope, nil, flowStmt, func(target *symbol.Scope) {
				addEdge(nil, target)
			})
			continue
		}
		seqDecl := item.SequenceDeclaration()
		if seqDecl == nil {
			continue
		}
		seqScope, err := ctx.Scope.GetChildByParserRule(seqDecl)
		if err != nil {
			continue
		}
		stages := seqDecl.AllStageDeclaration()
		stageScopes := make([]*symbol.Scope, len(stages))
		for i, stageDecl := range stages {
			if stageScope, err := seqScope.GetChildByParserRule(stageDecl); err == nil {
				stageScopes[i] = stageScope
			}
		}
		if len(stageScopes) > 0 && stageScopes[0] != nil {
			addEdge(seqScope, stageScopes[0])
		}
		for i, stageDecl := range stages {
			stageScope := stageScopes[i]
			if stageScope == nil {
				continue
			}
			var nextStage *symbol.Scope
			if i+1 < len(stageScopes) {
				nextStage = stageScopes[i+1]
			}
			body := stageDecl.StageBody()
			if body == nil {
				continue
			}
			for _, stageItem := range body.AllStageItem() {
				if flowStmt := stageItem.FlowStatement(); flowStmt != nil {
					walkFlowOutputs(ctx.Context, seqScope, nextStage, flowStmt, func(target *symbol.Scope) {
						addEdge(stageScope, target)
					})
				}
				if single := stageItem.SingleInvocation(); single != nil {
					if fn := single.Function(); fn != nil {
						addEdge(stageScope, resolveCalledFunction(ctx.Context, seqScope, fn))
					}
				}
			}
		}
	}
	return edges
}

// walkFlowOutputs invokes addTarget for every scope that a flow statement
// causes control to flow into: every function it invokes (`foo{}` anywhere
// in the flow, including inside routing tables) and every stage or sequence
// activated by a `=>` transition.
func walkFlowOutputs(
	stdCtx stdcontext.Context,
	resolveScope *symbol.Scope,
	nextStage *symbol.Scope,
	flowStmt parser.IFlowStatementContext,
	addTarget func(*symbol.Scope),
) {
	for _, node := range flowStmt.AllFlowNode() {
		if f := node.Function(); f != nil {
			addTarget(resolveCalledFunction(stdCtx, resolveScope, f))
		}
	}
	for _, rt := range flowStmt.AllRoutingTable() {
		for _, entry := range rt.AllRoutingEntry() {
			for _, node := range entry.AllFlowNode() {
				if f := node.Function(); f != nil {
					addTarget(resolveCalledFunction(stdCtx, resolveScope, f))
				}
			}
		}
	}
	transition := false
	for _, child := range flowStmt.GetChildren() {
		switch x := child.(type) {
		case parser.IFlowOperatorContext:
			transition = x.TRANSITION() != nil
		case parser.IFlowNodeContext:
			if transition {
				addTarget(resolveTransitionTarget(stdCtx, resolveScope, nextStage, x))
			}
			transition = false
		default:
			transition = false
		}
	}
}

// resolveTransitionTarget resolves a `=>` target to the stage or sequence
// scope it activates, or nil when the target is a channel write, an
// expression, a function invocation (those invocations are collected by
// walkFlowOutputs's flow-node scan), or an identifier that does not resolve
// to a stage/sequence.
func resolveTransitionTarget(
	stdCtx stdcontext.Context,
	resolveScope *symbol.Scope,
	nextStage *symbol.Scope,
	target parser.IFlowNodeContext,
) *symbol.Scope {
	if target.NEXT() != nil {
		return nextStage
	}
	idCtx := target.Identifier()
	if idCtx == nil {
		return nil
	}
	resolved, err := resolveScope.Resolve(stdCtx, idCtx.IDENTIFIER().GetText())
	if err != nil {
		return nil
	}
	if resolved.Kind == symbol.KindStage || resolved.Kind == symbol.KindSequence {
		return resolved
	}
	return nil
}

// resolveCalledFunction resolves a function-invocation AST node to the
// scope of the function being called, or nil if it does not resolve to a
// function (e.g., the name is undefined or refers to a non-function symbol).
func resolveCalledFunction(
	stdCtx stdcontext.Context,
	scope *symbol.Scope,
	fn parser.IFunctionContext,
) *symbol.Scope {
	resolved, err := scope.Resolve(stdCtx, fn.IDENTIFIER().GetText())
	if err != nil || resolved.Kind != symbol.KindFunction {
		return nil
	}
	return resolved
}

// reachableFromRoot returns the set of scopes reachable from the virtual
// root (the nil key in edges) via BFS over the forward edge map.
func reachableFromRoot(
	edges map[*symbol.Scope]set.Set[*symbol.Scope],
) set.Set[*symbol.Scope] {
	reached := set.New[*symbol.Scope]()
	var queue []*symbol.Scope
	for root := range edges[nil] {
		if reached.Contains(root) {
			continue
		}
		reached.Add(root)
		queue = append(queue, root)
	}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for next := range edges[node] {
			if reached.Contains(next) {
				continue
			}
			reached.Add(next)
			queue = append(queue, next)
		}
	}
	return reached
}

// emitReachabilityDiagnostics emits ARC5102, ARC5201, and ARC5202 for every
// declared function, stage, or sequence that is not in reached. Declarations
// whose name begins with an underscore are skipped, and unreachable stages
// inside an unstarted sequence are suppressed so the user sees one
// diagnostic per intent rather than a cascade.
func emitReachabilityDiagnostics(
	ctx context.Context[parser.IProgramContext],
	reached set.Set[*symbol.Scope],
) {
	for _, child := range ctx.Scope.Children {
		if child.AST == nil || strings.HasPrefix(child.Name, "_") {
			continue
		}
		switch child.Kind {
		case symbol.KindFunction:
			if !child.Type.IsValid() || reached.Contains(child) {
				continue
			}
			ctx.Diagnostics.Add(diagnostics.
				Warningf(child.AST, "uncalled function '%s'", child.Name).
				WithCode(codes.UncalledFunction).
				WithNote("prefix the name with an underscore to suppress this warning"))
		case symbol.KindSequence:
			if !reached.Contains(child) {
				ctx.Diagnostics.Add(diagnostics.
					Warningf(child.AST, "unstarted sequence '%s'", child.Name).
					WithCode(codes.UnstartedSequence).
					WithNote("prefix the name with an underscore to suppress this warning"))
				continue
			}
			for _, stage := range child.Children {
				if stage.Kind != symbol.KindStage || stage.AST == nil {
					continue
				}
				if strings.HasPrefix(stage.Name, "_") || reached.Contains(stage) {
					continue
				}
				ctx.Diagnostics.Add(diagnostics.
					Warningf(stage.AST, "unreachable stage '%s'", stage.Name).
					WithCode(codes.UnreachableStage).
					WithNote("prefix the name with an underscore to suppress this warning"))
			}
		}
	}
}
