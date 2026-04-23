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

// Sequence and stage reachability
// ===============================
//
// Arc sequences and stages form an activation graph. The virtual root is
// connected to every sequence that appears as the target of a module-scope
// `=>` flow. Entering a sequence activates its first stage (in source order),
// which we model as an implicit sequence -> entry-stage edge. Stages
// transition to each other via `=>` inside their body; `=> next` targets the
// lexically next stage in the same sequence, and `=> seq_name` activates
// another sequence.
//
// A sequence that is unreachable from the virtual root is ARC5202
// (unstarted). A stage inside a reachable sequence that is not itself
// reachable is ARC5201 (unreachable stage); we skip ARC5201 when the
// containing sequence is unstarted so the user gets one diagnostic per
// declaration rather than a cascade.

// analyzeSequenceReachability emits ARC5201 and ARC5202 diagnostics by
// computing forward reachability over the sequence/stage activation graph.
func analyzeSequenceReachability(ctx context.Context[parser.IProgramContext]) {
	edges := collectActivationEdges(ctx)
	reachable := reachableFromRoot(edges)
	emitReachabilityDiagnostics(ctx, reachable)
}

// collectActivationEdges walks the program AST and builds the activation
// graph. The nil key in the returned map represents the virtual root.
func collectActivationEdges(
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

	for _, item := range ctx.AST.AllTopLevelItem() {
		if flowStmt := item.FlowStatement(); flowStmt != nil {
			walkFlowTransitions(flowStmt, func(target parser.IFlowNodeContext) {
				addEdge(nil, resolveFlowTarget(ctx.Context, ctx.Scope, nil, target))
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
			stageScope, err := seqScope.GetChildByParserRule(stageDecl)
			if err != nil {
				continue
			}
			stageScopes[i] = stageScope
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
				flowStmt := stageItem.FlowStatement()
				if flowStmt == nil {
					continue
				}
				walkFlowTransitions(flowStmt, func(target parser.IFlowNodeContext) {
					addEdge(stageScope, resolveFlowTarget(ctx.Context, seqScope, nextStage, target))
				})
			}
		}
	}
	return edges
}

// walkFlowTransitions invokes fn for each flow node that appears as the
// target of a `=>` operator in flowStmt, walking in source order.
func walkFlowTransitions(
	flowStmt parser.IFlowStatementContext,
	fn func(parser.IFlowNodeContext),
) {
	transition := false
	for _, child := range flowStmt.GetChildren() {
		switch x := child.(type) {
		case parser.IFlowOperatorContext:
			transition = x.TRANSITION() != nil
		case parser.IFlowNodeContext:
			if transition {
				fn(x)
			}
			transition = false
		default:
			transition = false
		}
	}
}

// resolveFlowTarget resolves a `=>` target flow node to a stage or sequence
// scope if one applies, and nil otherwise (e.g., transitions to channels,
// expressions, or function invocations are not activation edges). The
// resolveScope argument is the scope in which names are looked up: the root
// scope for module-level flows or the containing sequence scope for
// stage-body flows. nextStage is the scope used to resolve the `next` keyword.
func resolveFlowTarget(
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
	name := idCtx.IDENTIFIER().GetText()
	resolved, err := resolveScope.Resolve(stdCtx, name)
	if err != nil {
		return nil
	}
	if resolved.Kind == symbol.KindStage || resolved.Kind == symbol.KindSequence {
		return resolved
	}
	return nil
}

// reachableFromRoot returns the set of sequence and stage scopes reachable
// from the virtual root via a BFS over edges. The virtual root is modeled
// with the nil key in edges.
func reachableFromRoot(edges map[*symbol.Scope]set.Set[*symbol.Scope]) set.Set[*symbol.Scope] {
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

// emitReachabilityDiagnostics emits ARC5202 for every declared sequence not
// in reached, and ARC5201 for every stage declared inside a reached sequence
// that is not itself reached. Declarations whose name begins with an
// underscore are treated as intentionally inert and skipped.
func emitReachabilityDiagnostics(
	ctx context.Context[parser.IProgramContext],
	reached set.Set[*symbol.Scope],
) {
	for _, child := range ctx.Scope.Children {
		if child.Kind != symbol.KindSequence || child.AST == nil {
			continue
		}
		if strings.HasPrefix(child.Name, "_") {
			continue
		}
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
			if strings.HasPrefix(stage.Name, "_") {
				continue
			}
			if !reached.Contains(stage) {
				ctx.Diagnostics.Add(diagnostics.
					Warningf(stage.AST, "unreachable stage '%s'", stage.Name).
					WithCode(codes.UnreachableStage).
					WithNote("prefix the name with an underscore to suppress this warning"))
			}
		}
	}
}
