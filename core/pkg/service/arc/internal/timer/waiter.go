// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build linux || windows

package timer

// waiter is the state of a goroutine that waits on an OS timer.
type waiter struct {
	// done closes when the goroutine exits.
	done chan struct{}
	// err is the goroutine's failure. It is safe to read after done closes.
	err error
}

// failure returns the goroutine's failure, or nil while it runs.
func (w *waiter) failure() error {
	select {
	case <-w.done:
		return w.err
	default:
		return nil
	}
}
