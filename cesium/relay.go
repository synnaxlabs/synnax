// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"io"
	"sync"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type relay struct {
	delta    *confluence.DynamicDeltaMultiplier[Frame]
	inlet    confluence.Inlet[Frame]
	shutdown io.Closer
}

func newRelay(o *options) *relay {
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(o.Instrumentation))
	delta := confluence.NewDynamicDeltaMultiplier[Frame]()
	frames := confluence.NewStream[Frame](10)
	delta.InFrom(frames)
	delta.Flow(sCtx)
	return &relay{
		delta:    delta,
		inlet:    frames,
		shutdown: signal.NewShutdown(sCtx, cancel),
	}
}

func (r *relay) connect(buffer int) (confluence.Outlet[Frame], func()) {
	frames := confluence.NewStream[Frame](buffer)
	frames.SetInletAddress(address.Newf("%s-storage", address.Rand().String()))
	r.delta.Connect(frames)
	return frames, func() {
		var wg sync.WaitGroup
		// NOTE: This area is a source of concurrency bugs. BE CAREFUL. We need to make
		// sure we drain the frames in a SEPARATE goroutine. This prevents deadlocks
		// inside the relay.
		wg.Add(1)
		go func() {
			confluence.Drain[Frame](frames)
			wg.Done()
		}()
		r.delta.Disconnect(frames)
		wg.Wait()
	}
}

func (r *relay) close() error { return r.shutdown.Close() }
