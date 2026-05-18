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
	"path/filepath"
	"sort"

	"github.com/synnaxlabs/x/errors"
)

// SchemaGlob is the canonical pattern for discovering top-level schema files.
const SchemaGlob = "schemas/*.oracle"

func globOracleSchemas(repoRoot string) ([]string, error) {
	pattern := filepath.Join(repoRoot, SchemaGlob)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid glob pattern %q", pattern)
	}
	sort.Strings(matches)
	return matches, nil
}
