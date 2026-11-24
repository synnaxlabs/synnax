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
	"github.com/synnaxlabs/arc/analyzer/codes"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// checkModuleRules reports a misplaced export, authority block, or top-level flow.
func checkModuleRules(ctx acontext.Context[parser.IProgramContext]) {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if ctx.Module {
			if auth := item.AuthorityBlock(); auth != nil {
				ctx.Diagnostics.Add(codes.ModuleAuthority.Atf(
					auth, "a module cannot have an authority block",
				))
			}
			if flow := item.FlowStatement(); flow != nil {
				ctx.Diagnostics.Add(codes.ModuleFlow.Atf(
					flow, "a module cannot have a top-level flow",
				))
			}
		}
		if item.EXPORT() == nil {
			continue
		}
		if !ctx.Module {
			ctx.Diagnostics.Add(codes.ExportOutsideModule.Atf(
				item, "export is only valid in a module",
			).WithEdits("Remove export", removeExport(item)))
			continue
		}
		if decl, name := exportedDeclaration(item); name == nil {
			ctx.Diagnostics.Add(codes.UnnamedExport.Atf(
				decl, "an exported declaration needs a name",
			))
		}
	}
}

// removeExport returns the edit that deletes the export of item and the space after it.
func removeExport(item parser.ITopLevelItemContext) protocol.TextEdit {
	var (
		export  = item.EXPORT().GetSymbol()
		decl, _ = exportedDeclaration(item)
		after   = export.GetInputStream().GetText(
			export.GetStop()+1,
			decl.GetStart().GetStart()-1,
		)
		start = protocol.Position{
			Line:      uint32(export.GetLine() - 1),
			Character: uint32(export.GetColumn()),
		}
		end = protocol.Position{
			Line:      start.Line,
			Character: start.Character + uint32(len(export.GetText())),
		}
		space = len(after) - len(strings.TrimLeft(after, " \t\r\n"))
	)
	return protocol.TextEdit{Range: protocol.Range{
		Start: start,
		End:   diagnostics.Advance(end, after, space),
	}}
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
