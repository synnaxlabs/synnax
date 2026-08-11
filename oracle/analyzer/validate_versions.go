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
	"github.com/synnaxlabs/x/diagnostics"
)

// validateImportPlacement enforces the version-file import rules: a version
// file imports other version files (pinned, for stored references) or live
// schemas (floating, for resolved references and unversioned resources); a
// live schema never imports another resource's version files.
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
