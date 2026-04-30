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

// Prettier is a Formatter that runs prettier over its input via stdin.
// `--stdin-filepath=<absPath>` is passed so prettier can resolve the
// surrounding .prettierrc / prettier.config.* and pick the right parser
// from the file extension.
//
// Prettier's Node-based cold start is the dominant cost. The format pass
// runs files in parallel to amortize it.
type Prettier struct {
	// Bin is the prettier binary to run. Defaults to "prettier" on PATH.
	// Set to "npx" with Args=["prettier", ...] to mirror the legacy
	// post-hook invocation if `prettier` is not on PATH directly.
	Bin string
	// Args are extra args inserted before --stdin-filepath. Empty by default.
	Args []string
}

// NewPrettier returns a Prettier formatter using `npx prettier` for
// compatibility with the existing post-hook configuration.
func NewPrettier() *Prettier {
	return &Prettier{Bin: "npx", Args: []string{"prettier"}}
}

// Format runs prettier with content on stdin.
func (p *Prettier) Format(content []byte, absPath string) ([]byte, error) {
	args := append([]string{}, p.Args...)
	args = append(args, "--stdin-filepath", absPath)
	return runStdin(context.Background(), p.Bin, args, content)
}
