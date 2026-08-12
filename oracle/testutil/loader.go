// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides testing utilities for the oracle code generator.
package testutil

import (
	"strings"

	"github.com/synnaxlabs/oracle/paths"
)

// MockFileLoader is a file loader that serves files from an in-memory map. It
// implements analyzer.FileLoader for use in tests.
type MockFileLoader struct {
	// Files maps import paths to file contents.
	Files map[string]string
	root  string
}

// NewMockFileLoader creates a new MockFileLoader with an empty file map and default
// repo root of "/mock/repo".
func NewMockFileLoader() *MockFileLoader {
	return &MockFileLoader{Files: make(map[string]string), root: "/mock/repo"}
}

// NewMockFileLoaderWithRoot creates a new MockFileLoader with a custom repo root.
func NewMockFileLoaderWithRoot(root string) *MockFileLoader {
	return &MockFileLoader{Files: make(map[string]string), root: root}
}

// Add adds a file to the mock loader and returns the loader for chaining.
func (m *MockFileLoader) Add(path, content string) *MockFileLoader {
	m.Files[path] = content
	return m
}

// Load implements analyzer.FileLoader. The returned filePath always ends in ".oracle";
// importPath may be passed with or without the suffix.
func (m *MockFileLoader) Load(importPath string) (string, string, error) {
	stem := strings.TrimSuffix(importPath, ".oracle")
	if content, ok := m.Files[stem]; ok {
		return content, stem + ".oracle", nil
	}
	if content, ok := m.Files[stem+".oracle"]; ok {
		return content, stem + ".oracle", nil
	}
	return "", "", &FileNotFoundError{Path: importPath}
}

// RepoRoot implements analyzer.FileLoader.
func (m *MockFileLoader) RepoRoot() string { return m.root }

// Versioned implements analyzer.FileLoader, reporting whether the mock holds any
// version file for the resource at importPath.
func (m *MockFileLoader) Versioned(importPath string) bool {
	dir, ok := paths.VersionsDir(paths.EnsureOracleExtension(importPath))
	if !ok {
		return false
	}
	for path := range m.Files {
		if strings.HasPrefix(path, dir+"/") {
			return true
		}
	}
	return false
}

// FileNotFoundError is returned when MockFileLoader cannot find a requested file.
type FileNotFoundError struct {
	Path string
}

func (e *FileNotFoundError) Error() string { return "file not found: " + e.Path }
