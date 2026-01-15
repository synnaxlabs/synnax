// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package paths provides centralized path utilities for Oracle that ensure all paths
// are resolved relative to the git repository root, regardless of working directory.
package paths

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

// RepoRoot finds the git repository root from the current working directory.
// It first tries `git rev-parse --show-toplevel`, then falls back to walking
// up the directory tree looking for a .git directory.
// Returns an error if not within a git repository.
func RepoRoot() (string, error) {
	// Try git rev-parse first (most reliable)
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}

	// Fallback: walk up looking for .git directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "failed to get working directory")
	}

	return findGitRoot(cwd)
}

// findGitRoot walks up the directory tree looking for a .git directory.
func findGitRoot(startPath string) (string, error) {
	current := startPath

	for {
		gitPath := filepath.Join(current, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			return "", errors.Newf("oracle must be run within a git repository: no .git directory found in %s or any parent", startPath)
		}
		current = parent
	}
}

// Normalize converts any path to a repo-relative form.
// - Absolute paths are made relative to repoRoot
// - Relative paths are resolved from cwd then made relative to repoRoot
// - Already repo-relative paths are returned as-is (after validation)
// Returns an error if the path would escape the repository.
func Normalize(path, repoRoot string) (string, error) {
	if path == "" {
		return "", errors.New("path cannot be empty")
	}

	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		// Could be repo-relative or cwd-relative
		// First try as repo-relative
		candidatePath := filepath.Join(repoRoot, path)
		if _, err := os.Stat(candidatePath); err == nil {
			// Path exists as repo-relative
			absPath = candidatePath
		} else {
			// Try resolving from cwd
			cwd, err := os.Getwd()
			if err != nil {
				return "", errors.Wrap(err, "failed to get working directory")
			}
			absPath = filepath.Clean(filepath.Join(cwd, path))
		}
	}

	// Make relative to repo root
	relPath, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to compute relative path")
	}

	// Check for path traversal (escaping repo)
	if strings.HasPrefix(relPath, "..") {
		return "", errors.Newf("path %q escapes repository root", path)
	}

	return relPath, nil
}

// Resolve converts a repo-relative path to an absolute path.
func Resolve(repoRelative, repoRoot string) string {
	if filepath.IsAbs(repoRelative) {
		return filepath.Clean(repoRelative)
	}
	return filepath.Join(repoRoot, repoRelative)
}

// ValidateOutput ensures an output path is valid and within repo bounds.
// It checks for path traversal attempts and invalid characters.
func ValidateOutput(path, repoRoot string) error {
	if path == "" {
		return errors.New("output path cannot be empty")
	}

	// Check for path traversal patterns
	if strings.Contains(path, "..") {
		return errors.Newf("output path %q contains path traversal (..) which is not allowed", path)
	}

	// Check for absolute paths (should be repo-relative)
	if filepath.IsAbs(path) {
		return errors.Newf("output path %q must be repo-relative, not absolute", path)
	}

	// Check for invalid prefixes
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return errors.Newf("output path %q must be repo-relative", path)
	}

	// Verify the resolved path stays within repo
	resolved := Resolve(path, repoRoot)
	relPath, err := filepath.Rel(repoRoot, resolved)
	if err != nil {
		return errors.Wrap(err, "failed to validate output path")
	}

	if strings.HasPrefix(relPath, "..") {
		return errors.Newf("output path %q would escape repository root", path)
	}

	return nil
}

// RelativeImport calculates the relative import path from one repo-relative path to another.
// Both paths should be repo-relative directory paths (not file paths).
// Returns a path suitable for use in import statements (e.g., "./sibling" or "../parent/other").
func RelativeImport(from, to string) (string, error) {
	if from == to {
		return ".", nil
	}

	rel, err := filepath.Rel(from, to)
	if err != nil {
		return "", errors.Wrap(err, "failed to compute relative import")
	}

	// Ensure forward slashes for import paths
	rel = filepath.ToSlash(rel)

	// If it doesn't start with . or .., prefix with ./
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}

	return rel, nil
}

// EnsureOracleExtension adds the .oracle extension if not present.
func EnsureOracleExtension(path string) string {
	if !strings.HasSuffix(path, ".oracle") {
		return path + ".oracle"
	}
	return path
}

// DeriveNamespace extracts the namespace from a file path.
// For "schema/core/label.oracle" returns "label".
func DeriveNamespace(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".oracle")
}
