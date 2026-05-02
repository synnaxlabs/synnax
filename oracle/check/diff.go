// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package check

import (
	"fmt"
	"strings"
)

// unifiedDiff produces a tiny unified-style diff between want and got.
// It is intentionally simple - line-by-line, no LCS - because the inputs
// are generated files where every byte difference is a real change. We
// don't need git's heuristics to surface a useful preview to the user.
//
// The output is bounded to roughly maxLines diff operations to keep
// terminal output sane; substitutions count as one operation but emit
// two lines, so worst-case output is 2*maxLines diff lines. When
// truncated the trailing line records how many source lines on each
// side remain unprocessed. Empty if want == got.
func unifiedDiff(label, want, got string, maxLines int) string {
	if want == got {
		return ""
	}
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s (on disk)\n", label)
	fmt.Fprintf(&sb, "+++ %s (generated)\n", label)
	emitted := 0
	limit := maxLines
	if limit <= 0 {
		limit = 40
	}

	i, j := 0, 0
	for i < len(wantLines) || j < len(gotLines) {
		if emitted >= limit {
			fmt.Fprintf(&sb, "... up to %d source line(s) remain unprocessed\n",
				(len(wantLines)-i)+(len(gotLines)-j))
			break
		}
		switch {
		case i >= len(wantLines):
			fmt.Fprintf(&sb, "+%s\n", gotLines[j])
			j++
		case j >= len(gotLines):
			fmt.Fprintf(&sb, "-%s\n", wantLines[i])
			i++
		case wantLines[i] == gotLines[j]:
			i++
			j++
			continue
		default:
			fmt.Fprintf(&sb, "-%s\n", wantLines[i])
			fmt.Fprintf(&sb, "+%s\n", gotLines[j])
			i++
			j++
			emitted++
			continue
		}
		emitted++
	}
	return sb.String()
}
