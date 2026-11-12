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

// clang is a formatter that runs `clang-format` over its input via stdin.
// `--assume-filename=<absPath>` is passed so the tool can locate the
// nearest .clang-format relative to the file's intended location.
type clang struct{}

// NewClang returns a clang formatter.
func NewClang() *clang { return &clang{} }

// Format runs clang-format with content on stdin.
func (c *clang) Format(
	ctx context.Context,
	content []byte,
	absPath string,
) ([]byte, error) {
	return stdinRun{
		Name:  "clang-format",
		Args:  []string{"--assume-filename=" + absPath},
		Stdin: content,
	}.run(ctx)
}
