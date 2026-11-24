// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"cmp"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/lsp"
	"go.lsp.dev/protocol"
)

// Warnings returns the warning-level diagnostics in d, in order.
func Warnings(d *diagnostics.Diagnostics) []diagnostics.Diagnostic {
	return lo.Filter(*d, func(diag diagnostics.Diagnostic, _ int) bool {
		return diag.Severity == protocol.DiagnosticSeverityWarning
	})
}

// ApplyFix returns source with the edits of fix made.
func ApplyFix(source string, fix diagnostics.Fix) string {
	edits := slices.Clone(fix.Edits)
	// A later edit goes first, so the positions of the earlier edits stay correct.
	slices.SortFunc(edits, func(a, b protocol.TextEdit) int {
		return cmp.Or(
			cmp.Compare(b.Range.Start.Line, a.Range.Start.Line),
			cmp.Compare(b.Range.Start.Character, a.Range.Start.Character),
		)
	})
	for _, edit := range edits {
		source = lsp.ApplyIncrementalChange(
			source,
			&protocol.TextDocumentContentChangePartial{
				Range: edit.Range,
				Text:  edit.NewText,
			},
		)
	}
	return source
}
