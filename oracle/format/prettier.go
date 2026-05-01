// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package format

import (
	"context"
	"path/filepath"
)

// Prettier is a Formatter that runs prettier over its input via stdin.
// `--stdin-filepath=<absPath>` is passed so prettier can resolve the
// surrounding .prettierrc / prettier.config.* and pick the right parser
// from the file extension. The command runs from the nearest
// package.json directory so `npx` resolves the package-local prettier
// binary instead of the system PATH copy.
type Prettier struct {
	// Bin is the prettier binary to run.
	Bin string
	// Args are extra args inserted before --stdin-filepath.
	Args []string
}

// NewPrettier returns a Prettier formatter using `npx prettier`.
func NewPrettier() *Prettier {
	return &Prettier{Bin: "npx", Args: []string{"prettier"}}
}

// Format runs prettier with content on stdin.
func (p *Prettier) Format(ctx context.Context, content []byte, absPath string) ([]byte, error) {
	args := append([]string{}, p.Args...)
	args = append(args, "--stdin-filepath", absPath)
	dir := findProjectDir(absPath, "package.json")
	if dir == "" {
		dir = filepath.Dir(absPath)
	}
	return stdinRun{
		Name:  p.Bin,
		Args:  args,
		Dir:   dir,
		Stdin: content,
	}.run(ctx)
}
