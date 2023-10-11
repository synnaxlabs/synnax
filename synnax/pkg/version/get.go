// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/git"
	"go.uber.org/zap"
	"io"
)

//go:embed VERSION
var fs embed.FS

const unknown = "unknown"
const errorMsg = "unexpected failure to resolve version"

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
	return string(v)
}

func Dev() (string, error) {
	return git.CurrentCommit(true)
}

func Get() string {
	d, err := Dev()
	if err == nil {
		return d
	}
	return Prod()
}
