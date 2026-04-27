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
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	basetypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// nameNode returns the IDENTIFIER terminal in scope.AST whose text matches
// scope.Name when one exists, falling back to scope.AST otherwise.
// Diagnostics tagged TagUnnecessary use this to fade only the name rather
// than the entire declaration.
func nameNode(scope *symbol.Scope) antlr.Tree {
	if scope == nil || scope.AST == nil {
		return nil
	}
	if id := parser.FindIdentifierToken(scope.AST, scope.Name); id != nil {
		return id
	}
	return scope.AST
}

// analyzeDeclarations emits ARC5101 (unused variable / stateful variable /
// channel alias) and ARC5103 (unused global constant) for every declaration
// whose scope was never recorded in ctx.ReferencedSymbols during analysis.
//
// Declarations whose name begins with an underscore or whose type failed to
// resolve are skipped to avoid piling warnings onto unrelated errors.
func analyzeDeclarations(ctx context.Context[parser.IProgramContext]) {
	if ctx.ReferencedSymbols == nil {
		return
	}
	walkScope(ctx.Scope, *ctx.ReferencedSymbols, ctx.Diagnostics)
}

func walkScope(
	scope *symbol.Scope,
	referenced set.Set[*symbol.Scope],
	diag *diagnostics.Diagnostics,
) {
	for _, child := range scope.Children {
		if d := unusedDeclarationDiagnostic(child, referenced); d != nil {
			diag.Add(*d)
		}
		walkScope(child, referenced, diag)
	}
}

func unusedDeclarationDiagnostic(
	scope *symbol.Scope,
	referenced set.Set[*symbol.Scope],
) *diagnostics.Diagnostic {
	if scope.AST == nil ||
		strings.HasPrefix(scope.Name, "_") ||
		!scope.Type.IsValid() ||
		referenced.Contains(scope) {
		return nil
	}
	switch scope.Kind {
	case symbol.KindGlobalConstant:
		return newUnusedDiagnostic(scope, codes.UnusedGlobalConstant, "unused global constant")
	case symbol.KindVariable:
		if scope.SourceID != nil && scope.Type.Kind == basetypes.KindChan {
			return newUnusedDiagnostic(scope, codes.UnusedVariable, "unused channel alias")
		}
		return newUnusedDiagnostic(scope, codes.UnusedVariable, "unused variable")
	case symbol.KindStatefulVariable:
		return newUnusedDiagnostic(scope, codes.UnusedVariable, "unused stateful variable")
	default:
		return nil
	}
}

func newUnusedDiagnostic(
	scope *symbol.Scope,
	code diagnostics.ErrorCode,
	label string,
) *diagnostics.Diagnostic {
	d := diagnostics.
		Warningf(nameNode(scope), "%s '%s'", label, scope.Name).
		WithCode(code).
		WithTags(diagnostics.TagUnnecessary).
		WithNote("prefix the name with an underscore to suppress this warning")
	return &d
}
