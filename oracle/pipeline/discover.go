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
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/synnaxlabs/x/errors"
)

func globOracleSchemas(repoRoot string) ([]string, error) {
	root := filepath.Join(repoRoot, "schemas")
	var matches []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && path == root {
				return filepath.SkipAll
			}
			return err
		}
		if d.IsDir() {
			if d.Name() == "snapshots" && filepath.Dir(path) == root {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".oracle" {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "walk schema directory %q", root)
	}
	sort.Strings(matches)
	return matches, nil
}
