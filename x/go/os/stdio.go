// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package os

import (
	"io"
	"os"
)

// StdIO is an io.ReadWriteCloser that reads from os.Stdin and writes to os.Stdout.
var StdIO io.ReadWriteCloser = stdIO{}

type stdIO struct{}

func (stdIO) Read(p []byte) (int, error)  { return os.Stdin.Read(p) }
func (stdIO) Write(p []byte) (int, error) { return os.Stdout.Write(p) }
func (stdIO) Close() error                { return nil }
