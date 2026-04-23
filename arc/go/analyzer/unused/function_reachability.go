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

// Function reachability
// =====================
//
// A function is "uncalled" (ARC5102) when no call site for it lives in code
// that can actually execute. A call site lives in:
//   - a top-level flow statement (always live),
//   - a stage body (live iff the stage is reachable per the sequence
//     reachability analysis),
//   - another function's body (live iff that enclosing function is itself
//     live — determined by fixpoint).
//
// This pass runs after analyzeSequenceReachability so that stage liveness
// is already known. It uses ctx.CallEdges for function-to-function calls
// (collected during expression analysis) and walks the AST afresh to find
// call sites inside flow statements and stage bodies.

// callerRef identifies the code unit containing a function call site.
//
// kind == callerTopLevel   -> caller is a module-scope flow; always live.
// kind == callerStage      -> scope is the enclosing stage.
// kind == callerFunction   -> scope is the enclosing function.
type callerRef struct {
	kind  callerKind
	scope *symbol.Scope
}

type callerKind int

const (
	callerTopLevel callerKind = iota
	callerStage
	callerFunction
)

// analyzeFunctionReachability emits ARC5102 diagnostics for top-level
// functions that have no call site inside live code. reachable is the set
// of reachable sequence and stage scopes produced by
// analyzeSequenceReachability.
func analyzeFunctionReachability(
	ctx context.Context[parser.IProgramContext],
	reachable set.Set[*symbol.Scope],
) {
	callers := collectFunctionCallers(ctx)
	live := computeLiveFunctions(callers, reachable)
	emitUncalledFunctionDiagnostics(ctx, live)
}

// collectFunctionCallers returns a map keyed by callee function scope to
// the set of caller references where it is invoked. Every function declared
// in the program appears as a key (with an empty set if never called) so
// the emitter can iterate declared functions via the map.
func collectFunctionCallers(
	ctx context.Context[parser.IProgramContext],
) map[*symbol.Scope]set.Set[callerRef] {
	callers := map[*symbol.Scope]set.Set[callerRef]{}
	addCall := func(callee *symbol.Scope, ref callerRef) {
		if callee == nil {
			return
		}
		s, ok := callers[callee]
		if !ok {
			s = set.New[callerRef]()
			callers[callee] = s
		}
		s.Add(ref)
	}

	// Seed every declared function so the emitter knows about uncalled ones.
	for _, child := range ctx.Scope.Children {
		if child.Kind == symbol.KindFunction && child.AST != nil {
			if _, ok := callers[child]; !ok {
				callers[child] = set.New[callerRef]()
			}
		}
	}

	// Function-body calls are already collected by the analyzer for channel
	// propagation; reuse them to avoid re-walking expression trees.
	for _, edge := range *ctx.CallEdges {
		addCall(edge.Callee, callerRef{kind: callerFunction, scope: edge.Caller})
	}

	for _, item := range ctx.AST.AllTopLevelItem() {
		if flowStmt := item.FlowStatement(); flowStmt != nil {
			walkFlowFunctionCalls(ctx.Context, ctx.Scope, flowStmt, func(callee *symbol.Scope) {
				addCall(callee, callerRef{kind: callerTopLevel})
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
		for _, stageDecl := range seqDecl.AllStageDeclaration() {
			stageScope, err := seqScope.GetChildByParserRule(stageDecl)
			if err != nil {
				continue
			}
			body := stageDecl.StageBody()
			if body == nil {
				continue
			}
			for _, stageItem := range body.AllStageItem() {
				if flowStmt := stageItem.FlowStatement(); flowStmt != nil {
					walkFlowFunctionCalls(ctx.Context, seqScope, flowStmt, func(callee *symbol.Scope) {
						addCall(callee, callerRef{kind: callerStage, scope: stageScope})
					})
				}
				if single := stageItem.SingleInvocation(); single != nil {
					if fn := single.Function(); fn != nil {
						addCall(
							resolveCalledFunction(ctx.Context, seqScope, fn),
							callerRef{kind: callerStage, scope: stageScope},
						)
					}
				}
			}
		}
	}
	return callers
}

// walkFlowFunctionCalls visits every function invocation node inside a flow
// statement, including those nested in routing table entries.
func walkFlowFunctionCalls(
	stdCtx stdcontext.Context,
	resolveScope *symbol.Scope,
	flowStmt parser.IFlowStatementContext,
	fn func(callee *symbol.Scope),
) {
	for _, node := range flowStmt.AllFlowNode() {
		if f := node.Function(); f != nil {
			fn(resolveCalledFunction(stdCtx, resolveScope, f))
		}
	}
	for _, rt := range flowStmt.AllRoutingTable() {
		for _, entry := range rt.AllRoutingEntry() {
			for _, node := range entry.AllFlowNode() {
				if f := node.Function(); f != nil {
					fn(resolveCalledFunction(stdCtx, resolveScope, f))
				}
			}
		}
	}
}

// resolveCalledFunction returns the scope of the function named by fn if it
// resolves in the given scope and is indeed a function; otherwise nil.
func resolveCalledFunction(
	stdCtx stdcontext.Context,
	scope *symbol.Scope,
	fn parser.IFunctionContext,
) *symbol.Scope {
	name := fn.IDENTIFIER().GetText()
	resolved, err := scope.Resolve(stdCtx, name)
	if err != nil {
		return nil
	}
	if resolved.Kind != symbol.KindFunction {
		return nil
	}
	return resolved
}

// computeLiveFunctions returns the set of function scopes that have at
// least one live call site. A call site is live when the caller is a
// top-level flow, a reachable stage, or another live function. The
// computation is a simple fixpoint; since Arc programs are small the
// quadratic worst case is not a concern.
func computeLiveFunctions(
	callers map[*symbol.Scope]set.Set[callerRef],
	reachableStages set.Set[*symbol.Scope],
) set.Set[*symbol.Scope] {
	live := set.New[*symbol.Scope]()
	for {
		changed := false
		for callee, refs := range callers {
			if live.Contains(callee) {
				continue
			}
			for ref := range refs {
				if callerIsLive(ref, live, reachableStages) {
					live.Add(callee)
					changed = true
					break
				}
			}
		}
		if !changed {
			return live
		}
	}
}

func callerIsLive(
	ref callerRef,
	live set.Set[*symbol.Scope],
	reachableStages set.Set[*symbol.Scope],
) bool {
	switch ref.kind {
	case callerTopLevel:
		return true
	case callerStage:
		return ref.scope != nil && reachableStages.Contains(ref.scope)
	case callerFunction:
		return ref.scope != nil && live.Contains(ref.scope)
	default:
		return false
	}
}

// emitUncalledFunctionDiagnostics emits ARC5102 for each top-level function
// that is not in live. Declarations with an underscore-prefixed name or an
// invalid type are skipped (matching the ARC51xx suppression rules).
func emitUncalledFunctionDiagnostics(
	ctx context.Context[parser.IProgramContext],
	live set.Set[*symbol.Scope],
) {
	for _, child := range ctx.Scope.Children {
		if child.Kind != symbol.KindFunction || child.AST == nil {
			continue
		}
		if strings.HasPrefix(child.Name, "_") {
			continue
		}
		if !child.Type.IsValid() {
			continue
		}
		if live.Contains(child) {
			continue
		}
		ctx.Diagnostics.Add(diagnostics.
			Warningf(child.AST, "uncalled function '%s'", child.Name).
			WithCode(codes.UncalledFunction).
			WithNote("prefix the name with an underscore to suppress this warning"))
	}
}
