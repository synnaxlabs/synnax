// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stdin watches the standard input of a Core for a request to stop.
package stdin

import (
	"bufio"
	"io"
)

// StopKeyword is the line that requests a stop.
const StopKeyword = "stop"

// Watch reads lines from r until r is exhausted, and calls stop for each line equal to
// StopKeyword. When stopOnClose is set, Watch also calls stop once r is exhausted, so a
// Core whose parent process holds the other end of the pipe stops when that parent
// exits. Watch blocks, so callers run it in its own goroutine.
func Watch(r io.Reader, stopOnClose bool, stop func()) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if scanner.Text() == StopKeyword {
			stop()
		}
	}
	if stopOnClose {
		stop()
	}
}
