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
	"io"

	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
)

// NewHardShutdown extends a signal WaitGroup and its corresponding cancellation
// function to implement an io.Closer that (1) cancels the context and (2) waits for all
// routines to exit. This order is reversed when compared to NewGracefulShutdown, as it
// forces routines to exit before completing.
//
// The context.Canceled error returned by any routine is ignored, as it's assumed that a
// context cancellation is a nominal shutdown.
func NewHardShutdown(
	wg WaitGroup,
	cancel context.CancelFunc,
) io.Closer {
	return xio.CloserFunc(func() error {
		cancel()
		err := wg.Wait()
		return errors.Skip(err, context.Canceled)
	})
}

// NewGracefulShutdown extends a signal WaitGroup and its corresponding cancellation
// function to implement an io.Closer that (1) waits for all routines to exit and (2)
// cancels the context. This order is reversed when compared to NewHardShutdown, as it
// waits for all routines to gracefully exit before cancelling the context.
func NewGracefulShutdown(
	wg WaitGroup,
	cancel context.CancelFunc,
) io.Closer {
	return xio.CloserFunc(func() error {
		err := wg.Wait()
		cancel()
		return err
	})
}
