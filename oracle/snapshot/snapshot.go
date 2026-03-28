// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package snapshot manages schema snapshots for the Oracle migration system. A snapshot
// is a copy of all .oracle schema files at a migration point, used to diff against the
// current schema when generating the next migration.
package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

// Ensure the import path has .oracle extension.
func ensureOracleExt(p string) string {
	if strings.HasSuffix(p, ".oracle") {
		return p
	}
	return p + ".oracle"
}

// Create copies all .oracle files from schemasDir into snapshotsDir/v{version}/,
// preserving subdirectory structure. Skips the .snapshots directory itself.
func Create(schemasDir, snapshotsDir string, version int) error {
	destDir := filepath.Join(snapshotsDir, fmt.Sprintf("v%d", version))
	return filepath.Walk(schemasDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip the snapshots directory itself.
		if info.IsDir() && filepath.Base(path) == ".snapshots" {
			return filepath.SkipDir
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".oracle") {
			return nil
		}
		rel, err := filepath.Rel(schemasDir, path)
		if err != nil {
			return errors.Wrapf(err, "failed to compute relative path for %s", path)
		}
		dst := filepath.Join(destDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return errors.Wrapf(err, "failed to create directory for %s", dst)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to read %s", path)
		}
		return os.WriteFile(dst, data, 0644)
	})
}

// LatestVersion returns the highest snapshot version number in snapshotsDir,
// or 0 if no snapshots exist.
func LatestVersion(snapshotsDir string) (int, error) {
	entries, err := os.ReadDir(snapshotsDir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read snapshots directory %s", snapshotsDir)
	}
	max := 0
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "v") {
			continue
		}
		v, err := strconv.Atoi(entry.Name()[1:])
		if err != nil {
			continue
		}
		if v > max {
			max = v
		}
	}
	return max, nil
}

// FileLoader implements analyzer.FileLoader and redirects all imports to the snapshot
// directory. When a schema imports "schemas/telem", this loader reads from
// snapshotDir/telem.oracle instead of schemas/telem.oracle.
type FileLoader struct {
	snapshotDir string
	repoRoot    string
}

// NewFileLoader creates a FileLoader that reads from the given snapshot directory.
func NewFileLoader(snapshotDir, repoRoot string) *FileLoader {
	return &FileLoader{snapshotDir: snapshotDir, repoRoot: repoRoot}
}

func (l *FileLoader) Load(importPath string) (string, string, error) {
	// importPath is like "schemas/workspace" or "schemas/arc/graph".
	// Strip the "schemas/" prefix and resolve within the snapshot directory.
	rel := strings.TrimPrefix(importPath, "schemas/")
	rel = strings.TrimSuffix(rel, ".oracle") + ".oracle"
	fullPath := filepath.Join(l.snapshotDir, rel)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", "", err
	}
	// Return the original import path so namespace derivation works the same.
	filePath := importPath
	if !strings.HasSuffix(filePath, ".oracle") {
		filePath += ".oracle"
	}
	return string(content), filePath, nil
}

func (l *FileLoader) RepoRoot() string { return l.repoRoot }

// Files returns the .oracle file paths in a snapshot directory, sorted. Walks
// subdirectories to find all schema files.
func Files(snapshotDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(snapshotDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".oracle") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to walk snapshot directory %s", snapshotDir)
	}
	sort.Strings(files)
	return files, nil
}
