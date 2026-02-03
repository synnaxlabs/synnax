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
	"embed"
	"fmt"
	"io"
	"strings"
	"time"

	"go.uber.org/zap"
)

//go:embed VERSION
var fs embed.FS

const unknown = "unknown"
const errorMsg = "unexpected failure to resolve version"

// These variables can be set at build time using -ldflags:
// -X github.com/synnaxlabs/synnax/pkg/version.Version=1.0.0
// -X github.com/synnaxlabs/synnax/pkg/version.GitCommit=abc123
// -X github.com/synnaxlabs/synnax/pkg/version.BuildDate=2025-01-01T00:00:00Z
var (
	Version   string
	GitCommit string
	BuildDate string
)

// Prod returns the production version of Synnax.
func Prod() string {
	// If version was injected at build time, use it
	if Version != "" {
		return Version
	}

	// Otherwise fall back to embedded VERSION file
	f, err := fs.Open("VERSION")
	if err != nil {
		zap.S().Errorw(errorMsg, "error", err)
		return unknown
	}
	v, err := io.ReadAll(f)
	if err != nil {
		zap.S().Errorw(errorMsg, "error", err)
		return unknown
	}
	vString := string(v)
	vString = strings.TrimSpace(vString)
	vString = strings.ReplaceAll(vString, "\n", "")
	return vString
}

// Get returns the production version of Synnax.
func Get() string { return Prod() }

// Commit returns the git commit hash.
func Commit() string {
	if GitCommit != "" {
		return GitCommit
	}
	return unknown
}

// Date returns the build date.
func Date() string {
	if BuildDate != "" {
		return BuildDate
	}
	return unknown
}

// Time returns the build date as a time.Time.
// Returns zero time if BuildDate is not set or cannot be parsed.
func Time() time.Time {
	if BuildDate == "" || BuildDate == unknown {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, BuildDate)
	if err != nil {
		zap.S().Errorw("failed to parse build date", "error", err, "date", BuildDate)
		return time.Time{}
	}
	return t
}

// Full returns the full version string with commit and build date.
func Full() string {
	v := Get()
	commit := Commit()
	date := Date()

	if commit != unknown && date != unknown {
		return fmt.Sprintf("%s (commit: %s, built: %s)", v, commit[:7], date)
	} else if commit != unknown {
		return fmt.Sprintf("%s (commit: %s)", v, commit[:7])
	}
	return v
}
