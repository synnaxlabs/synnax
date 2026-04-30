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

// Go is a Formatter that runs `gofmt -s` over its input via stdin/stdout.
// Cold start of gofmt is in the single-millisecond range so a per-file
// invocation is acceptable.
type Go struct{}

// NewGo returns a Go formatter.
func NewGo() *Go { return &Go{} }

// Format runs gofmt -s with content on stdin.
func (g *Go) Format(content []byte, _ string) ([]byte, error) {
	return runStdin(context.Background(), "gofmt", []string{"-s"}, content)
}
