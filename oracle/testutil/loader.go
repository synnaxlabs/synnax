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

// MockFileLoader is a file loader that serves files from an in-memory map.
// It implements analyzer.FileLoader for use in tests.
type MockFileLoader struct {
	// Files maps import paths to file contents.
	Files map[string]string
	root  string
}

// NewMockFileLoader creates a new MockFileLoader with an empty file map
// and default repo root of "/mock/repo".
func NewMockFileLoader() *MockFileLoader {
	return &MockFileLoader{Files: make(map[string]string), root: "/mock/repo"}
}

// Add adds a file to the mock loader and returns the loader for chaining.
func (m *MockFileLoader) Add(path, content string) *MockFileLoader {
	m.Files[path] = content
	return m
}

// Load implements analyzer.FileLoader.
func (m *MockFileLoader) Load(importPath string) (string, string, error) {
	if content, ok := m.Files[importPath]; ok {
		return content, importPath + ".oracle", nil
	}
	if content, ok := m.Files[importPath+".oracle"]; ok {
		return content, importPath + ".oracle", nil
	}
	return "", "", &FileNotFoundError{Path: importPath}
}

// RepoRoot implements analyzer.FileLoader.
func (m *MockFileLoader) RepoRoot() string { return m.root }

// FileNotFoundError is returned when MockFileLoader cannot find a requested file.
type FileNotFoundError struct {
	Path string
}

func (e *FileNotFoundError) Error() string {
	return "file not found: " + e.Path
}
