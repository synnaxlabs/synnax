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

// BufFormat is a Formatter that runs `buf format` over .proto content. Buf
// has no stdin mode, so the content is written to a temp file, formatted,
// and the formatted bytes captured from stdout. Single-file `buf format`
// does not consult buf.yaml so the temp directory location does not affect
// output.
type BufFormat struct{}

// NewBufFormat returns a BufFormat formatter.
func NewBufFormat() *BufFormat { return &BufFormat{} }

// Format writes content to a temp file and returns `buf format`'s stdout.
func (b *BufFormat) Format(content []byte, absPath string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "oracle-buf-format-*")
	if err != nil {
		return nil, errors.Wrap(err, "create temp dir")
	}
	defer func() { _ = os.RemoveAll(dir) }()
	tmpFile := filepath.Join(dir, filepath.Base(absPath))
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		return nil, errors.Wrap(err, "write temp proto")
	}
	c := exec.CommandContext(context.Background(), "buf", "format", tmpFile)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return nil, errors.Wrapf(err, "buf format failed: %s", stderr.String())
	}
	return stdout.Bytes(), nil
}
