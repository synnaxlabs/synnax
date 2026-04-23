// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package unused emits diagnostics for declarations that are never referenced.
//
// This pass runs after the main analyzer passes have populated the scope tree and
// set Referenced on every scope that was looked up from a use-site. It walks the
// scope tree, and for each declaration in a category covered by an ARC51xx rule
// whose scope was never referenced, it emits a warning.
package unused

import (
	"strings"

	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	basetypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// Analyze walks the program's scope tree and emits warnings for unreferenced
// variable declarations (ARC5101). It should run after analyzeDeclarations so
// that every use-site has had a chance to mark its target scope Referenced.
func Analyze(ctx context.Context[parser.IProgramContext]) {
	walk(ctx.Scope, ctx.Diagnostics)
}

func walk(scope *symbol.Scope, diag *diagnostics.Diagnostics) {
	for _, child := range scope.Children {
		if d := unusedDiagnostic(child); d != nil {
			diag.Add(*d)
		}
		walk(child, diag)
	}
}

// unusedDiagnostic returns a pointer to a diagnostic for the scope if it is an
// unreferenced declaration subject to an ARC51xx rule, otherwise nil. Names
// beginning with an underscore are treated as intentionally unused, and
// declarations whose type failed to resolve are skipped to avoid piling a
// warning on top of the unrelated error the user is already seeing.
func unusedDiagnostic(scope *symbol.Scope) *diagnostics.Diagnostic {
	if scope.AST == nil {
		return nil
	}
	if scope.Referenced {
		return nil
	}
	if strings.HasPrefix(scope.Name, "_") {
		return nil
	}
	if !scope.Type.IsValid() {
		return nil
	}
	switch scope.Kind {
	case symbol.KindFunction:
		d := diagnostics.
			Warningf(scope.AST, "uncalled function '%s'", scope.Name).
			WithCode(codes.UncalledFunction).
			WithNote("prefix the name with an underscore to suppress this warning")
		return &d
	case symbol.KindVariable, symbol.KindStatefulVariable:
		label, ok := variableLabel(scope)
		if !ok {
			return nil
		}
		d := diagnostics.
			Warningf(scope.AST, "unused %s '%s'", label, scope.Name).
			WithCode(codes.UnusedVariable).
			WithNote("prefix the name with an underscore to suppress this warning")
		return &d
	default:
		return nil
	}
}

// variableLabel returns the human-readable label for a variable-kind scope for
// use in the unused-variable diagnostic message. The second return value is
// false if this scope's kind does not have a variable-flavored label.
func variableLabel(scope *symbol.Scope) (string, bool) {
	switch scope.Kind {
	case symbol.KindStatefulVariable:
		return "stateful variable", true
	case symbol.KindVariable:
		if scope.SourceID != nil && scope.Type.Kind == basetypes.KindChan {
			return "channel alias", true
		}
		return "variable", true
	default:
		return "", false
	}
}
