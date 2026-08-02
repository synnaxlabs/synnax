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

// validateVersionFileFields errors on non-persisted fields declared in a
// version file: version files record persisted shape only.
func validateVersionFileFields(c *analysisCtx, types []resolution.Type) {
	if _, _, ok := paths.VersionFile(c.filePath); !ok {
		return
	}
	for _, t := range types {
		form, ok := t.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		for _, f := range form.Fields {
			if isOmitField(f) {
				c.report(diagnostics.Errorf(nil,
					"%s.%s is @go marshal omit; version files record persisted shape only",
					t.QualifiedName, f.Name,
				))
			}
		}
	}
}

// isOmitField reports whether the field carries @go marshal omit.
func isOmitField(f resolution.Field) bool {
	dom, ok := f.Domains["go"]
	if !ok {
		return false
	}
	expr, ok := dom.Expressions.Find("marshal")
	if !ok {
		return false
	}
	for _, v := range expr.Values {
		if v.IdentValue == "omit" || v.StringValue == "omit" {
			return true
		}
	}
	return false
}
