// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

// resolveGoImportPath resolves a repo-relative output path to a full Go import path
// by walking up the directory tree to find the nearest go.mod file.
//
// Example: "core/pkg/service/rack" with module "github.com/synnaxlabs/synnax"
// returns "github.com/synnaxlabs/synnax/pkg/service/rack"
func resolveGoImportPath(outputPath, repoRoot string) (string, error) {
	absPath := filepath.Join(repoRoot, outputPath)

	// Walk up from the output path to find go.mod
	dir := absPath
	for {
		modPath := filepath.Join(dir, "go.mod")
		if fileExists(modPath) {
			moduleName, err := parseModuleName(modPath)
			if err != nil {
				return "", errors.Wrapf(err, "failed to parse go.mod at %s", modPath)
			}

			// Compute relative path from module root to output
			relPath, err := filepath.Rel(dir, absPath)
			if err != nil {
				return "", errors.Wrapf(err, "failed to compute relative path")
			}

			// Combine module name with relative path
			if relPath == "." {
				return moduleName, nil
			}
			return moduleName + "/" + filepath.ToSlash(relPath), nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			break
		}
		dir = parent
	}

	return "", errors.Newf("no go.mod found for path %s", outputPath)
}

// parseModuleName extracts the module name from a go.mod file.
func parseModuleName(modPath string) (string, error) {
	file, err := os.Open(modPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			// Extract module name (handles both "module foo" and "module foo // comment")
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

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
