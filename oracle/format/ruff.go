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

// Ruff is a Formatter that runs `ruff check --fix` followed by `ruff format` over its
// input via stdin. The command runs from the nearest pyproject.toml directory so `uv
// run` resolves the package-local ruff binary and the package's configuration applies.
type Ruff struct {
	// Bin is the ruff binary.
	Bin string
	// Args are extra args inserted before the subcommand.
	Args []string
}

func newRuff() *Ruff { return &Ruff{Bin: "uv", Args: []string{"run", "ruff"}} }

// Format runs `ruff check --fix` then `ruff format` with content on stdin, so the
// output of a lint fix is also formatted.
func (r *Ruff) Format(
	ctx context.Context,
	content []byte,
	absPath string,
) ([]byte, error) {
	dir := findProjectDir(absPath, "pyproject.toml")
	if dir == "" {
		dir = filepath.Dir(absPath)
	}
	checkArgs := append([]string{}, r.Args...)
	checkArgs = append(
		checkArgs,
		"check",
		"--fix",
		"--exit-zero",
		"--stdin-filename",
		absPath,
		"-",
	)
	fixed, err := stdinRun{
		Name:  r.Bin,
		Args:  checkArgs,
		Dir:   dir,
		Stdin: content,
	}.run(ctx)
	if err != nil {
		return nil, err
	}
	formatArgs := append([]string{}, r.Args...)
	formatArgs = append(formatArgs, "format", "--stdin-filename", absPath, "-")
	return stdinRun{
		Name:  r.Bin,
		Args:  formatArgs,
		Dir:   dir,
		Stdin: fixed,
	}.run(ctx)
}
