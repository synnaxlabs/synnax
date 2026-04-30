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
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/synnaxlabs/x/errors"
)

// stdinRun configures one stdin/stdout invocation of an external tool.
type stdinRun struct {
	// Name is the binary to invoke (e.g. "gofmt", "npx").
	Name string
	// Args is the rest of argv.
	Args []string
	// Dir is the working directory to run the command in. Empty means
	// inherit the caller's CWD.
	Dir string
	// Stdin is the bytes piped to the process.
	Stdin []byte
	// AllowExit tolerates a non-zero exit code provided the process
	// actually started. Tools like eslint and ruff check signal "issues
	// found" via exit code but still produce useful stdout.
	AllowExit bool
}

// run executes the configured invocation and returns stdout. On failure
// the error includes both the configured Name and the captured stderr,
// which is essential for debugging missing-binary or wrong-cwd issues.
func (r stdinRun) run(ctx context.Context) ([]byte, error) {
	c := exec.CommandContext(ctx, r.Name, r.Args...)
	c.Stdin = bytes.NewReader(r.Stdin)
	if r.Dir != "" {
		c.Dir = r.Dir
	}
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	if err != nil {
		if _, isExit := err.(*exec.ExitError); !isExit || !r.AllowExit {
			return nil, errors.Wrapf(err, "%s failed (cwd=%q, stderr=%q)", r.Name, r.Dir, stderr.String())
		}
	}
	return stdout.Bytes(), nil
}

// findPackageJSONDir walks up from the directory containing absPath and
// returns the first directory that holds a package.json. Returns "" if
// no package.json is found before reaching the filesystem root, in which
// case the caller can fall back to the file's containing directory.
func findPackageJSONDir(absPath string) string {
	dir := filepath.Dir(absPath)
	for {
		if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// findPyProjectDir walks up from the directory containing absPath and
// returns the first directory that holds a pyproject.toml. Returns ""
// if none is found before reaching the filesystem root.
func findPyProjectDir(absPath string) string {
	dir := filepath.Dir(absPath)
	for {
		if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
