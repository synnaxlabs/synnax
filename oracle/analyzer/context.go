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
	"os"

	"github.com/synnaxlabs/oracle/paths"
)

// FileLoader loads schema files for import resolution.
type FileLoader interface {
	// Load loads a schema file by its import path.
	// The importPath should be repo-relative (e.g., "schema/core/label").
	// Returns the file source content, the repo-relative file path, and any error.
	Load(importPath string) (source, filePath string, err error)

	// RepoRoot returns the absolute path to the git repository root.
	RepoRoot() string
}

// StandardFileLoader loads files from the filesystem relative to the git repo root.
type StandardFileLoader struct {
	repoRoot string
}

// NewStandardFileLoader creates a FileLoader that resolves paths from the repo root.
func NewStandardFileLoader(repoRoot string) *StandardFileLoader {
	return &StandardFileLoader{repoRoot: repoRoot}
}

// Load loads a schema file by its repo-relative import path.
func (l *StandardFileLoader) Load(importPath string) (string, string, error) {
	importPath = paths.EnsureOracleExtension(importPath)
	fullPath := paths.Resolve(importPath, l.repoRoot)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", "", err
	}
	return string(content), importPath, nil
}

// RepoRoot returns the absolute path to the git repository root.
func (l *StandardFileLoader) RepoRoot() string {
	return l.repoRoot
}

// DeriveNamespace extracts namespace from path: "schema/label.oracle" -> "label"
// This is a convenience wrapper around paths.DeriveNamespace.
func DeriveNamespace(path string) string {
	return paths.DeriveNamespace(path)
}
