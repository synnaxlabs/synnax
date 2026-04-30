// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package format

import "context"

// Clang is a Formatter that runs `clang-format` over its input via stdin.
// `--assume-filename=<absPath>` is passed so the tool can locate the
// nearest .clang-format relative to the file's intended location.
type Clang struct{}

// NewClang returns a Clang formatter.
func NewClang() *Clang { return &Clang{} }

// Format runs clang-format with content on stdin.
func (c *Clang) Format(content []byte, absPath string) ([]byte, error) {
	args := []string{"--assume-filename=" + absPath}
	return runStdin(context.Background(), "clang-format", args, content)
}
