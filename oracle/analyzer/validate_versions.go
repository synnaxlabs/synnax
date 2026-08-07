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

	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
)

// validateImportPlacement enforces the version-file import rules: a version
// file imports only other version files, and a live schema imports only its
// own resource's version files.
func validateImportPlacement(c *analysisCtx) {
	_, _, inVersionFile := paths.VersionFile(c.filePath)
	for _, imp := range c.ast.AllImportStmt() {
		path := strings.Trim(imp.STRING_LIT().GetText(), `"`)
		resource, _, importsVersionFile := paths.VersionFile(path)
		if inVersionFile && !importsVersionFile {
			c.report(diagnostics.Errorf(imp,
				"version files may only import other version files; %q is a live schema",
				path,
			))
		}
		if !inVersionFile && importsVersionFile && resource != c.namespace {
			c.report(diagnostics.Errorf(imp,
				"%s may only import its own resource's version files, not %q",
				c.namespace, path,
			))
		}
	}
}

// validateLiveMarshal errors on @go marshal declared in a live schema. The
// codec it configures is generated into versions/vN, so the version file owns
// every marshal tag: type-level roots and field-level omissions alike.
func validateLiveMarshal(c *analysisCtx, types []resolution.Type) {
	// The rule is about the repository's schema layout: only a live resource
	// schema has a versions directory to move the tag into.
	if _, _, live := paths.LiveSchema(c.filePath); !live {
		return
	}
	report := func(name string) {
		c.report(diagnostics.Errorf(nil,
			"%s declares @go marshal; declare it in the resource's version file",
			name,
		))
	}
	for _, t := range types {
		if dom, ok := t.Domains["go"]; ok {
			if _, has := dom.Expressions.Find("marshal"); has {
				report(t.QualifiedName)
			}
		}
		form, ok := t.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		for _, f := range form.Fields {
			if dom, ok := f.Domains["go"]; ok {
				if _, has := dom.Expressions.Find("marshal"); has {
					report(t.QualifiedName + "." + f.Name)
				}
			}
		}
	}
}
