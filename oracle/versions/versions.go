// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package versions models the explicitly managed schema version files under
// schemas/<domain>/versions/<resource>/vN.oracle. A version file enumerates a
// resource's complete persisted namespace at version N: full declarations for
// types whose shape changed at N, and alias lines (`Key = v0.Key`) pointing at
// the defining version for the rest. A name absent from the file was removed
// at N.
package versions

import (
	"fmt"
	"path"
	"strings"

	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/x/errors"
)

// Chain is one resource's discovered version history.
type Chain struct {
	// Domain is the schema domain directory ("synnax", "x", "arc").
	Domain string
	// Resource is the versioned live schema's base name ("channel").
	Resource string
	// Numbers holds the declared versions, ascending and dense from 0.
	Numbers []int
}

// Current returns the chain's current (highest) version.
func (c Chain) Current() int { return c.Numbers[len(c.Numbers)-1] }

// First returns the chain's oldest version. Chains normally start at v0;
// resources whose older history predates per-resource versioning (legacy
// payload ranges) start above it.
func (c Chain) First() int { return c.Numbers[0] }

// Predecessor returns the version preceding n in the chain, or false when n
// is the chain's first version.
func (c Chain) Predecessor(n int) (int, bool) {
	if n <= c.First() {
		return 0, false
	}
	return n - 1, true
}

// Dir returns the chain's repo-relative directory.
func (c Chain) Dir() string {
	return path.Join("schemas", c.Domain, paths.VersionsDirName, c.Resource)
}

// FilePath returns version n's repo-relative import path, without the .oracle
// extension.
func (c Chain) FilePath(n int) string {
	return path.Join(c.Dir(), fmt.Sprintf("v%d", n))
}

// LivePath returns the live schema's repo-relative import path.
func (c Chain) LivePath() string {
	return path.Join("schemas", c.Domain, c.Resource)
}

// LiveFromFilePath converts a version file's import path
// ("schemas/x/versions/telem/v0") to its resource's live import path
// ("schemas/x/telem"). It errors when the path does not have the
// schemas/<domain>/versions/<resource>/vN shape.
func LiveFromFilePath(p string) (string, error) {
	parts := strings.Split(strings.TrimSuffix(p, ".oracle"), "/")
	if len(parts) != 5 || parts[0] != "schemas" ||
		parts[2] != paths.VersionsDirName {
		return "", errors.Newf("%s is not a schema version file path", p)
	}
	if _, _, ok := paths.VersionFile(p); !ok {
		return "", errors.Newf("%s is not a schema version file path", p)
	}
	return path.Join(parts[0], parts[1], parts[3]), nil
}
