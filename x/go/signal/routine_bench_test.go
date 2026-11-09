// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signal_test

import (
	"context"
	"testing"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

// BenchmarkGo starts one routine on a new context and waits for it, on a live context
// and on one canceled before the start.
func BenchmarkGo(b *testing.B) {
	for _, c := range []struct {
		name     string
		canceled bool
	}{{name: "live"}, {name: "canceled", canceled: true}} {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sCtx, cancel := signal.Isolated()
				if c.canceled {
					cancel()
				}
				sCtx.Go(func(context.Context) error { return nil })
				if err := sCtx.Wait(); err != nil && !errors.Is(err, context.Canceled) {
					b.Fatal(err)
				}
				cancel()
			}
		})
	}
}
