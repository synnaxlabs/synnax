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
	"encoding/json"
	"path/filepath"

	"github.com/synnaxlabs/x/errors"
)

// ESLint is a Formatter that applies `eslint --fix` to TypeScript /
// JavaScript content via stdin. ESLint's `--fix` flag is incompatible
// with `--stdin`, so this implementation uses
// `--fix-dry-run --format json --stdin --stdin-filename=<absPath>` and
// parses the JSON envelope to extract the fixed source. When eslint
// reports no fixes applicable to the input, the JSON contains no
// `output` field for the file and the original content is returned
// unchanged.
//
// The command runs from the nearest package.json directory so `npx`
// resolves the package-local eslint binary and so jiti/tsx (used to
// load the package's eslint.config.ts) is available in node_modules.
type ESLint struct {
	// Bin is the eslint binary to run.
	Bin string
	// Args are extra args inserted before the stdin flags.
	Args []string
}

// NewESLint returns an ESLint formatter using `npx eslint`.
func NewESLint() *ESLint {
	return &ESLint{Bin: "npx", Args: []string{"eslint"}}
}

type eslintReport struct {
	Output   string `json:"output"`
	Messages []struct {
		Message string `json:"message"`
		Fatal   bool   `json:"fatal"`
	} `json:"messages"`
}

// Format runs eslint on stdin and returns the fixed source.
//
// Non-fatal lint findings are tolerated: oracle-generated code is
// allowed to trip lint rules that have no auto-fix available, matching
// the legacy post-hook's stance. A fatal parse error from eslint is
// surfaced as an error since it indicates the generated source is
// broken.
func (e *ESLint) Format(ctx context.Context, content []byte, absPath string) ([]byte, error) {
	args := append([]string{}, e.Args...)
	args = append(args,
		"--fix-dry-run",
		"--format", "json",
		"--stdin",
		"--stdin-filename", absPath,
	)
	dir := findProjectDir(absPath, "package.json")
	if dir == "" {
		dir = filepath.Dir(absPath)
	}
	stdout, err := stdinRun{
		Name:      e.Bin,
		Args:      args,
		Dir:       dir,
		Stdin:     content,
		AllowExit: true,
	}.run(ctx)
	if err != nil {
		return nil, err
	}
	if len(stdout) == 0 {
		return nil, errors.Newf("eslint produced empty output for %s (cwd=%s)", absPath, dir)
	}
	var reports []eslintReport
	if err := json.Unmarshal(stdout, &reports); err != nil {
		return nil, errors.Wrapf(err, "parse eslint output for %s: %s", absPath, string(stdout))
	}
	if len(reports) == 0 {
		return content, nil
	}
	r := reports[0]
	for _, m := range r.Messages {
		if m.Fatal {
			return nil, errors.Newf("eslint fatal error in %s: %s", absPath, m.Message)
		}
	}
	if r.Output == "" {
		return content, nil
	}
	return []byte(r.Output), nil
}
