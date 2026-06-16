// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signal

import (
	"context"

	"github.com/synnaxlabs/x/errors"
)

// WaitGroup provides methods for detecting and waiting for the exit of goroutines
// managed by a signal.Conductor.
type WaitGroup interface {
	// Wait waits for all running goroutines to exit and returns the error that caused
	// the context to stop. If one or more routines failed with a genuine (non-context)
	// error, the first such failure in routine start order is returned, even when a
	// sibling routine returned context.Canceled first in response to the cancellation.
	// If no routine failed, Wait returns the first context cancellation error observed,
	// or nil if every routine exited cleanly. Returns nil if no goroutines are running.
	// Wait is NOT safe to call concurrently with any other wait methods.
	Wait() error
	// Stopped returns a channel that is closed when the context is canceled AND all
	// running goroutines have exited.
	Stopped() <-chan struct{}
}

// Wait implements the WaitGroup interface.
func (c *core) Wait() error {
	err := c.internal.Wait()
	// errgroup records only the first error returned by any routine. When a routine
	// fails with a genuine error and cancels the context via CancelOnFail, sibling
	// routines frequently return context.Canceled and can win the race to be recorded
	// as that first error, masking the real failure. Recover the genuine failure from
	// the routine states, which record non-context errors explicitly, so callers (and
	// NewHardShutdown's context.Canceled filtering) observe the true cause.
	if errors.IsAny(err, context.Canceled, context.DeadlineExceeded) {
		c.mu.RLock()
		defer c.mu.RUnlock()
		for _, r := range c.mu.routines {
			if r.state.err != nil &&
				!errors.IsAny(r.state.err, context.Canceled, context.DeadlineExceeded) {
				return r.state.err
			}
		}
	}
	return err
}
