// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package unused emits diagnostics for dead code: declarations that are never
// referenced (ARC51xx) and statements, stages, sequences, and functions that
// can never execute (ARC52xx, ARC5102).
//
// Three passes, all running after the main analyzer has produced the scope
// tree, call edges, and referenced-symbol set:
//
//	analyzeDeclarations  -> ARC5101 (variables), ARC5103 (globals)
//	analyzeUnreachable   -> ARC5203 (code after an always-returning statement)
//	analyzeReachability  -> ARC5102, ARC5201, ARC5202 (single BFS over the
//	                        combined activation graph)
package unused

import (
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
)

// Analyze runs every pass in the unused package, appending diagnostics to
// ctx.Diagnostics. It should be called after AnalyzeProgram's main passes
// so the scope tree, ctx.CallEdges, and ctx.ReferencedSymbols are complete.
func Analyze(ctx context.Context[parser.IProgramContext]) {
	analyzeDeclarations(ctx)
	analyzeUnreachable(ctx)
	analyzeReachability(ctx)
}
