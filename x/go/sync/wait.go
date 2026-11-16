// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sync

import "context"

// Wait waits for g and returns its error, or returns ctx.Err() if ctx is done first.
// When ctx wins, the goroutine that waits on g exits once g returns.
func Wait(ctx context.Context, g interface{ Wait() error }) error {
	done := make(chan error, 1)
	go func() { done <- g.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
