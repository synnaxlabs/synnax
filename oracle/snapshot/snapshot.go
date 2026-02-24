// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

// LatestVersion returns the highest snapshot version in the snapshots
// directory, or 0 if no snapshots exist or the directory does not exist.
func LatestVersion(snapshotsDir string) (int, error) {
	entries, err := os.ReadDir(snapshotsDir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read snapshots directory %q", snapshotsDir)
	}
	max := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		v, ok := parseVersion(e.Name())
		if !ok {
			continue
		}
		if v > max {
			max = v
		}
	}
	return max, nil
}

// Create copies all .oracle files from schemasDir into
// snapshotsDir/v<version>/. Creates directories as needed.
func Create(schemasDir, snapshotsDir string, version int) error {
	versionDir := filepath.Join(snapshotsDir, fmt.Sprintf("v%d", version))
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create snapshot directory %q", versionDir)
	}
	files, err := globOracle(schemasDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return errors.Wrapf(err, "failed to read schema file %q", f)
		}
		dst := filepath.Join(versionDir, filepath.Base(f))
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return errors.Wrapf(err, "failed to write snapshot file %q", dst)
		}
	}
	return nil
}

// Check compares current .oracle files in schemasDir against the latest
// snapshot. Returns nil if they match, or an error describing the
// differences if they differ. Returns an error if no snapshots exist.
func Check(schemasDir, snapshotsDir string) error {
	latest, err := LatestVersion(snapshotsDir)
	if err != nil {
		return err
	}
	if latest == 0 {
		return errors.New("no snapshots found; run 'oracle migrate generate' first")
	}
	snapshotDir := filepath.Join(snapshotsDir, fmt.Sprintf("v%d", latest))
	currentFiles, err := oracleFileMap(schemasDir)
	if err != nil {
		return errors.Wrap(err, "failed to read current schemas")
	}
	snapshotFiles, err := oracleFileMap(snapshotDir)
	if err != nil {
		return errors.Wrap(err, "failed to read snapshot schemas")
	}

	var diffs []string
	for name := range currentFiles {
		if _, ok := snapshotFiles[name]; !ok {
			diffs = append(diffs, fmt.Sprintf("added: %s", name))
		}
	}
	for name := range snapshotFiles {
		if _, ok := currentFiles[name]; !ok {
			diffs = append(diffs, fmt.Sprintf("removed: %s", name))
		}
	}
	for name, current := range currentFiles {
		snapshot, ok := snapshotFiles[name]
		if !ok {
			continue
		}
		if current != snapshot {
			diffs = append(diffs, fmt.Sprintf("modified: %s", name))
		}
	}
	if len(diffs) == 0 {
		return nil
	}
	sort.Strings(diffs)
	return errors.Newf(
		"schema changed but no migration generated. Run 'oracle migrate generate' "+
			"and commit the result.\n  %s",
		strings.Join(diffs, "\n  "),
	)
}

func parseVersion(name string) (int, bool) {
	if !strings.HasPrefix(name, "v") {
		return 0, false
	}
	n, err := strconv.Atoi(name[1:])
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func globOracle(dir string) ([]string, error) {
	pattern := filepath.Join(dir, "*.oracle")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to glob %q", pattern)
	}
	sort.Strings(matches)
	return matches, nil
}

func oracleFileMap(dir string) (map[string]string, error) {
	files, err := globOracle(dir)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read %q", f)
		}
		m[filepath.Base(f)] = string(data)
	}
	return m, nil
}
