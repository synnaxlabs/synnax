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

	"github.com/synnaxlabs/x/diagnostics"
)

// FileDiagnostics groups analyzer diagnostics by the schema file they originate from.
// The file key carries the origin that diagnostics.Diagnostic does not, mirroring how
// LSP scopes diagnostics by URI at the envelope. Files are kept in first-seen order so
// downstream output is deterministic.
type FileDiagnostics struct {
	order  []string
	byFile map[string]*diagnostics.Diagnostics
}

var _ error = (*FileDiagnostics)(nil)

// NewFileDiagnostics constructs an empty FileDiagnostics.
func NewFileDiagnostics() *FileDiagnostics {
	return &FileDiagnostics{byFile: make(map[string]*diagnostics.Diagnostics)}
}

// at returns the diagnostics for file, allocating and recording order on first use.
func (f *FileDiagnostics) at(file string) *diagnostics.Diagnostics {
	ds, ok := f.byFile[file]
	if !ok {
		ds = &diagnostics.Diagnostics{}
		f.byFile[file] = ds
		f.order = append(f.order, file)
	}
	return ds
}

// Report adds a diagnostic under file, deduplicating within that file.
func (f *FileDiagnostics) Report(file string, d diagnostics.Diagnostic) {
	f.at(file).Add(d)
}

// Add reports a diagnostic with no associated file.
func (f *FileDiagnostics) Add(d diagnostics.Diagnostic) { f.Report("", d) }

// MergeFile folds a flat set of diagnostics (e.g. parse output) under file.
func (f *FileDiagnostics) MergeFile(file string, ds diagnostics.Diagnostics) {
	f.at(file).Merge(ds)
}

// Combine folds every diagnostic from other into f, preserving file grouping.
func (f *FileDiagnostics) Combine(other *FileDiagnostics) {
	for _, file := range other.order {
		f.at(file).Merge(*other.byFile[file])
	}
}

// Ok reports whether no file holds an error-level diagnostic.
func (f *FileDiagnostics) Ok() bool {
	for _, ds := range f.byFile {
		if !ds.Ok() {
			return false
		}
	}
	return true
}

// Empty reports whether no diagnostics were recorded.
func (f *FileDiagnostics) Empty() bool {
	for _, ds := range f.byFile {
		if !ds.Empty() {
			return false
		}
	}
	return true
}

// Errors returns every error-level diagnostic across all files, in file order.
func (f *FileDiagnostics) Errors() []diagnostics.Diagnostic {
	var out []diagnostics.Diagnostic
	for _, file := range f.order {
		out = append(out, f.byFile[file].Errors()...)
	}
	return out
}

// Each calls fn for every diagnostic in file order, passing its origin file.
func (f *FileDiagnostics) Each(fn func(file string, d diagnostics.Diagnostic)) {
	for _, file := range f.order {
		for _, d := range *f.byFile[file] {
			fn(file, d)
		}
	}
}

// Flat returns all diagnostics as one ordered collection, dropping file grouping. Used
// where a per-file envelope already fixes the origin, such as LSP publish.
func (f *FileDiagnostics) Flat() diagnostics.Diagnostics {
	var out diagnostics.Diagnostics
	for _, file := range f.order {
		out = append(out, *f.byFile[file]...)
	}
	return out
}

// String formats all diagnostics grouped by file, prefixing each group with its file
// name when one is known.
func (f *FileDiagnostics) String() string {
	parts := make([]string, 0, len(f.order))
	for _, file := range f.order {
		ds := f.byFile[file]
		if ds.Empty() {
			continue
		}
		if file == "" {
			parts = append(parts, ds.String())
		} else {
			parts = append(parts, file+":\n"+ds.String())
		}
	}
	if len(parts) == 0 {
		return "analysis successful"
	}
	return strings.Join(parts, "\n")
}

// Error implements the error interface.
func (f *FileDiagnostics) Error() string { return f.String() }
