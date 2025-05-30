// Copyright 2025 Synnax Labs, Inc.
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
