// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pipeline

import (
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/paths"
)

// overlayLoader serves the freshly formatted in-memory schema bytes when the
// analyzer asks for one of the top-level inputs, and falls through to the
// underlying loader for everything else (transitive imports that aren't in
// the input set).
//
// Without this overlay, the analyzer would re-read schema files from disk
// using their pre-format bytes, which means format drift (e.g. an extra
// trailing newline) would silently slip through analysis when in fact the
// formatter rewrote the file. Analyzing the formatted bytes is the only
// behaviour consistent with what sync would write to disk on the next run.
type overlayLoader struct {
	inner   analyzer.FileLoader
	overlay map[string][]byte
}

func newOverlayLoader(inner analyzer.FileLoader, overlay map[string][]byte) *overlayLoader {
	return &overlayLoader{inner: inner, overlay: overlay}
}

func (l *overlayLoader) Load(importPath string) (string, string, error) {
	relPath := paths.EnsureOracleExtension(importPath)
	if data, ok := l.overlay[relPath]; ok {
		return string(data), relPath, nil
	}
	return l.inner.Load(importPath)
}

func (l *overlayLoader) RepoRoot() string { return l.inner.RepoRoot() }
