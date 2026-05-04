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

// Ruff is a Formatter that runs `ruff format` followed by
// `ruff check --fix` over its input via stdin. The command runs from
// the nearest pyproject.toml directory so `uv run` resolves the
// package-local ruff binary and the package's configuration applies.
type Ruff struct {
	// Bin is the ruff binary.
	Bin string
	// Args are extra args inserted before the subcommand.
	Args []string
}

// NewRuff returns a Ruff formatter using `uv run ruff`.
func NewRuff() *Ruff {
	return &Ruff{Bin: "uv", Args: []string{"run", "ruff"}}
}

// Format runs `ruff format` then `ruff check --fix` with content on stdin.
func (r *Ruff) Format(ctx context.Context, content []byte, absPath string) ([]byte, error) {
	dir := findProjectDir(absPath, "pyproject.toml")
	if dir == "" {
		dir = filepath.Dir(absPath)
	}
	formatArgs := append([]string{}, r.Args...)
	formatArgs = append(formatArgs, "format", "--stdin-filename", absPath, "-")
	formatted, err := stdinRun{
		Name:  r.Bin,
		Args:  formatArgs,
		Dir:   dir,
		Stdin: content,
	}.run(ctx)
	if err != nil {
		return nil, err
	}
	checkArgs := append([]string{}, r.Args...)
	checkArgs = append(checkArgs, "check", "--fix", "--exit-zero", "--stdin-filename", absPath, "-")
	return stdinRun{
		Name:  r.Bin,
		Args:  checkArgs,
		Dir:   dir,
		Stdin: formatted,
	}.run(ctx)
}
