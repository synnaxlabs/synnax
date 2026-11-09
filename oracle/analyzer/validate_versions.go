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

	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/x/diagnostics"
)

// validateImportPlacement enforces that a live schema never imports another
// resource's version files. A version file pins every versioned dependency it
// references; the versions gate checks that rule.
func validateImportPlacement(c *analysisCtx) {
	_, _, inVersionFile := paths.VersionFile(c.filePath)
	if inVersionFile {
		return
	}
	for _, imp := range c.ast.AllImportStmt() {
		path := strings.Trim(imp.STRING_LIT().GetText(), `"`)
		resource, _, importsVersionFile := paths.VersionFile(path)
		if importsVersionFile && resource != c.namespace {
			c.report(diagnostics.Errorf(imp,
				"%s may only import its own resource's version files, not %q",
				c.namespace, path,
			))
		}
	}
}

// validateNoVersionFileActions rejects actions and @actions in version files. Actions
// are wire mutations, not persisted content, so they are live-owned: sync carries them
// from the live schema and never projects them from the chain.
func validateNoVersionFileActions(c *analysisCtx) {
	if _, _, inVersionFile := paths.VersionFile(c.filePath); !inVersionFile {
		return
	}
	for _, def := range c.ast.AllDefinition() {
		for _, ad := range actionDefs(def) {
			c.report(diagnostics.Errorf(ad,
				"actions are live-owned; declare %q in the live schema, not a "+
					"version file",
				ad.IDENT().GetText(),
			))
		}
		for _, d := range bodyDomains(def) {
			if d.IDENT().GetText() == actionsDomain {
				c.report(diagnostics.Errorf(d,
					"actions are live-owned; mark @actions in the live schema, not "+
						"a version file",
				))
			}
		}
	}
}

func actionDefs(def parser.IDefinitionContext) []parser.IActionDefContext {
	if ud := def.UnionDef(); ud != nil && ud.UnionBody() != nil {
		return ud.UnionBody().AllActionDef()
	}
	if full, ok := def.StructDef().(*parser.StructFullContext); ok &&
		full.StructBody() != nil {
		return full.StructBody().AllActionDef()
	}
	return nil
}
