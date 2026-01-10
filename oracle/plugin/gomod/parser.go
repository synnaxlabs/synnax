// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package gomod provides utilities for parsing go.mod files and resolving Go import paths.
package gomod

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

// DefaultModulePrefix is the fallback import path when go.mod resolution fails.
const DefaultModulePrefix = "github.com/synnaxlabs/synnax/"

// ParseModuleName extracts the module name from a go.mod file.
func ParseModuleName(modPath string) (string, error) {
	file, err := os.Open(modPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", errors.Newf("no module directive found in %s", modPath)
}

// ResolveImportPath resolves a repo-relative output path to a full Go import path.
// It searches for go.mod files to determine the correct module path.
// If no go.mod is found, it falls back to using the fallbackPrefix.
func ResolveImportPath(outputPath, repoRoot, fallbackPrefix string) string {
	if repoRoot == "" {
		return fallbackPrefix + outputPath
	}

	absPath := filepath.Join(repoRoot, outputPath)
	dir := absPath
	for {
		modPath := filepath.Join(dir, "go.mod")
		if info, err := os.Stat(modPath); err == nil && !info.IsDir() {
			moduleName, err := ParseModuleName(modPath)
			if err != nil {
				break
			}
			relPath, err := filepath.Rel(dir, absPath)
			if err != nil {
				break
			}
			if relPath == "." {
				return moduleName
			}
			return moduleName + "/" + filepath.ToSlash(relPath)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return fallbackPrefix + outputPath
}

// FileExists checks if a file exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// FindRepoRoot walks up from the given path to find the git repository root.
func FindRepoRoot(path string) string {
	dir := filepath.Dir(path)
	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
