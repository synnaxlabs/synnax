// Copyright 2025 Synnax Labs, Inc.
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
	"path/filepath"
	"strings"
)

// FileLoader loads schema files for import resolution.
type FileLoader interface {
	Load(importPath string) (source, filePath string, err error)
}

// StandardFileLoader loads files from the filesystem.
type StandardFileLoader struct{ BaseDir string }

func NewStandardFileLoader(baseDir string) *StandardFileLoader {
	return &StandardFileLoader{baseDir}
}

func (l *StandardFileLoader) Load(importPath string) (string, string, error) {
	if !strings.HasSuffix(importPath, ".oracle") {
		importPath += ".oracle"
	}
	fullPath := importPath
	if !filepath.IsAbs(importPath) {
		fullPath = filepath.Join(l.BaseDir, importPath)
	}
	content, err := os.ReadFile(fullPath)
	return string(content), fullPath, err
}

// DeriveNamespace extracts namespace from path: "schema/label.oracle" -> "label"
func DeriveNamespace(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".oracle")
}
