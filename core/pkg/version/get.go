// Copyright 2025 Synnax Labs, Inc.
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
	"go.uber.org/zap"
	"io"
	"strings"
)

//go:embed VERSION
var fs embed.FS

const unknown = "unknown"
const errorMsg = "unexpected failure to resolve version"

// Prod returns the production version of Synnax.
func Prod() string {
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
