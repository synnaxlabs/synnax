// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package plumber_test

import (
	"context"
	"testing"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
)

// BenchmarkPipelineFlow flows a chain of four transforms, then closes its input and
// waits for every routine to exit.
func BenchmarkPipelineFlow(b *testing.B) {
	addrs := []address.Address{"a", "b", "c", "d"}
	b.ReportAllocs()
	for b.Loop() {
		pipe := plumber.New()
		for i, addr := range addrs {
			t := &confluence.LinearTransform[int, int]{}
			t.Transform = func(_ context.Context, v int) (int, bool, error) {
				return v, true, nil
			}
			pipe.SetSegment(addr, t)
			if i > 0 {
				pipe.MustConnect[int](addrs[i-1], addr, 1)
			}
		}
		seg := &plumber.Segment[int, int]{
			Pipeline:         pipe,
			RouteInletsTo:    addrs[:1],
			RouteOutletsFrom: addrs[len(addrs)-1:],
		}
		in := confluence.NewStream[int]()
		seg.InFrom(in)
		seg.OutTo(confluence.NewStream[int]())
		sCtx, cancel := signal.Isolated()
		seg.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		in.Close()
		if err := sCtx.Wait(); err != nil {
			b.Fatal(err)
		}
		cancel()
	}
}
