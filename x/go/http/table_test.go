// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"fmt"
	"os"
	"testing"
	"text/tabwriter"

	"github.com/synnaxlabs/x/errors"
)

// table prints a column-aligned report to stdout. It holds the first write error and
// surfaces it at Flush, so a benchmark report can be built one row at a time without
// an error check around every row.
type table struct {
	w   *tabwriter.Writer
	err error
}

// newTable starts a table with the given header. Columns are separated by tabs in both
// the header and every row.
func newTable(header string) *table {
	t := &table{w: tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)}
	t.row(header)
	return t
}

// row appends a formatted row. The trailing newline is added.
func (t *table) row(format string, args ...any) {
	if t.err != nil {
		return
	}
	_, t.err = fmt.Fprintf(t.w, format+"\n", args...)
}

// flush writes the table out, failing tb if any row or the flush itself errored.
func (t *table) flush(tb testing.TB) {
	tb.Helper()
	if err := errors.Combine(t.err, t.w.Flush()); err != nil {
		tb.Fatal(err)
	}
}
