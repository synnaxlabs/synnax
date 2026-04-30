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
	"os/exec"

	"github.com/synnaxlabs/x/errors"
)

// runStdin runs cmd with content piped to stdin and returns its stdout. The
// command's stderr is captured and included in any error so the caller can
// see why the formatter rejected the input.
func runStdin(ctx context.Context, name string, args []string, content []byte) ([]byte, error) {
	c := exec.CommandContext(ctx, name, args...)
	c.Stdin = bytes.NewReader(content)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return nil, errors.Wrapf(err, "%s failed: %s", name, stderr.String())
	}
	return stdout.Bytes(), nil
}

// runStdinAllowExit is like runStdin but tolerates a non-zero exit status.
// Some tools (eslint, ruff check) signal "issues found" via exit code while
// still emitting useful output on stdout that the caller wants to consume.
// Errors that prevent the process from running at all (missing binary, I/O
// error) are still propagated.
func runStdinAllowExit(ctx context.Context, name string, args []string, content []byte) ([]byte, error) {
	c := exec.CommandContext(ctx, name, args...)
	c.Stdin = bytes.NewReader(content)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, errors.Wrapf(err, "%s failed: %s", name, stderr.String())
		}
	}
	return stdout.Bytes(), nil
}
