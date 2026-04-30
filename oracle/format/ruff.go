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

// Ruff is a Formatter that runs `ruff format` followed by `ruff check --fix`
// over its input via stdin. Ruff is a single statically-linked Rust binary
// with sub-30ms cold start, so per-file invocation is cheap.
type Ruff struct {
	// Bin is the ruff binary. Defaults to "uv" with Args=["run", "ruff"] for
	// compatibility with the existing post-hook configuration.
	Bin string
	// Args are extra args inserted before the subcommand.
	Args []string
}

// NewRuff returns a Ruff formatter using `uv run ruff` for compatibility
// with the existing post-hook configuration.
func NewRuff() *Ruff {
	return &Ruff{Bin: "uv", Args: []string{"run", "ruff"}}
}

// Format runs `ruff format` then `ruff check --fix` with content on stdin.
//
// `--exit-zero` is passed to `ruff check` so that lint findings without an
// auto-fix don't fail the sync (matching the legacy behavior of the post-
// hook, which also tolerated such findings).
func (r *Ruff) Format(content []byte, absPath string) ([]byte, error) {
	formatArgs := append([]string{}, r.Args...)
	formatArgs = append(formatArgs, "format", "--stdin-filename", absPath, "-")
	formatted, err := runStdin(context.Background(), r.Bin, formatArgs, content)
	if err != nil {
		return nil, err
	}
	checkArgs := append([]string{}, r.Args...)
	checkArgs = append(checkArgs, "check", "--fix", "--exit-zero", "--stdin-filename", absPath, "-")
	return runStdin(context.Background(), r.Bin, checkArgs, formatted)
}
