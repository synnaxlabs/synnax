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
	"github.com/antlr4-go/antlr/v4"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/diagnostics"
)

// checkModuleRules reports a misplaced export, authority block, or top-level flow.
func checkModuleRules(ctx acontext.Context[parser.IProgramContext]) {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if ctx.Module {
			if auth := item.AuthorityBlock(); auth != nil {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					auth, "a module cannot have an authority block",
				))
			}
			if flow := item.FlowStatement(); flow != nil {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					flow, "a module cannot have a top-level flow",
				))
			}
		}
		if item.EXPORT() == nil {
			continue
		}
		if !ctx.Module {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				item, "export is only valid in a module",
			))
			continue
		}
		if decl, name := exportedDeclaration(item); name == nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				decl, "an exported declaration needs a name",
			))
		}
	}
}

// markExports flags the root symbol of every exported declaration.
func markExports(ctx acontext.Context[parser.IProgramContext]) {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if item.EXPORT() == nil {
			continue
		}
		_, name := exportedDeclaration(item)
		if name == nil {
			continue
		}
		if sym := ctx.Scope.FindChild(name.GetText()); sym != nil {
			sym.Exported = true
		}
	}
}

// exportedDeclaration returns the exported declaration and its name, nil if anonymous.
func exportedDeclaration(
	item parser.ITopLevelItemContext,
) (antlr.ParserRuleContext, antlr.TerminalNode) {
	if fn := item.FunctionDeclaration(); fn != nil {
		return fn, fn.IDENTIFIER()
	}
	if seq := item.SequenceDeclaration(); seq != nil {
		return seq, seq.IDENTIFIER()
	}
	if stage := item.StageDeclaration(); stage != nil {
		return stage, stage.IDENTIFIER()
	}
	v := item.VariableDeclaration()
	if local := v.LocalVariable(); local != nil {
		return v, local.IDENTIFIER()
	}
	return v, v.StatefulVariable().IDENTIFIER()
}
