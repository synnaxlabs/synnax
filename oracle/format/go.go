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
type Go struct{}

// NewGo returns a Go formatter.
func NewGo() *Go { return &Go{} }

// Format runs gofmt -s with content on stdin.
func (g *Go) Format(ctx context.Context, content []byte, _ string) ([]byte, error) {
	return stdinRun{Name: "gofmt", Args: []string{"-s"}, Stdin: content}.run(ctx)
}
