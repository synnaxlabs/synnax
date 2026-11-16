// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version

import (
	"strings"

	"golang.org/x/mod/semver"
)

// IsDev reports whether v is the version of a dev build: 0.0.0, or a 0.0 version that
// CI stamps with a pre-release. A dev version need not be valid semver, since its
// pre-release can hold a commit hash with a leading zero.
func IsDev(v string) bool { return strings.HasPrefix(v, "0.0.") }

// Valid reports whether v is a full semantic version without a "v" prefix or build
// metadata, such as 0.61.0 or 0.61.0-rc.1.
func Valid(v string) bool { return semver.Canonical(toSemver(v)) == toSemver(v) }

// Compare returns -1, 0, or +1 when a is older than, equal to, or newer than b. a and b
// are semantic versions without a "v" prefix. An invalid version is older than every
// valid one.
func Compare(a, b string) int { return semver.Compare(toSemver(a), toSemver(b)) }

func toSemver(v string) string { return "v" + v }
