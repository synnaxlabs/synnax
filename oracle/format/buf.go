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

// bufFormat is a formatter that runs `buf format` over .proto content. Buf
// has no stdin mode, so the content is written to a temp file, formatted,
// and the formatted bytes captured from stdout. Single-file `buf format`
// does not consult buf.yaml so the temp directory location does not affect
// output.
type bufFormat struct{}

// NewBufFormat returns a bufFormat formatter.
func NewBufFormat() *bufFormat { return &bufFormat{} }

// Format writes content to a temp file and returns `buf format`'s stdout.
func (b *bufFormat) Format(
	ctx context.Context,
	content []byte,
	absPath string,
) (out []byte, err error) {
	dir, err := os.MkdirTemp("", "oracle-buf-format-*")
	if err != nil {
		return nil, errors.Wrap(err, "create temp dir")
	}
	defer func() {
		if err = errors.Combine(err, os.RemoveAll(dir)); err != nil {
			out = nil
		}
	}()
	tmpFile := filepath.Join(dir, filepath.Base(absPath))
	if err := os.WriteFile(tmpFile, content, 0o644); err != nil {
		return nil, errors.Wrap(err, "write temp proto")
	}
	c := exec.CommandContext(ctx, "buf", "format", tmpFile)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return nil, errors.Wrapf(err, "buf format failed: %s", stderr.String())
	}
	return stdout.Bytes(), nil
}
