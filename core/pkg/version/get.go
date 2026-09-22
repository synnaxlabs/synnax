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
	"fmt"
	"time"

	"go.uber.org/zap"
)

const unknown = "unknown"

// The build injects these with -ldflags, for example
// -X github.com/synnaxlabs/synnax/pkg/version.version=1.0.0. Nothing writes them at
// runtime.
var (
	version   string
	gitCommit string
	buildDate string
)

// Get returns the version injected at build time, or 0.0.0 when none was.
func Get() string {
	if version != "" {
		return version
	}
	return "0.0.0"
}

// Commit returns the git commit injected at build time, or "unknown".
func Commit() string {
	if gitCommit != "" {
		return gitCommit
	}
	return unknown
}

// date returns the build date injected at build time, or "unknown".
func date() string {
	if buildDate != "" {
		return buildDate
	}
	return unknown
}

// Time returns the build date as a time.Time, or the zero time when it was not
// injected or does not parse.
func Time() time.Time {
	if buildDate == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, buildDate)
	if err != nil {
		zap.S().Errorw("failed to parse build date", "error", err, "date", buildDate)
		return time.Time{}
	}
	return t
}

// Full returns the version with the commit and build date when they are known.
func Full() string {
	v := Get()
	commit := Commit()
	d := date()
	if commit != unknown && d != unknown {
		return fmt.Sprintf("%s (commit: %s, built: %s)", v, commit[:7], d)
	} else if commit != unknown {
		return fmt.Sprintf("%s (commit: %s)", v, commit[:7])
	}
	return v
}
