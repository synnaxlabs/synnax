// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"sync"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type relayResponse struct {
	frame Frame
	group uint32
}

type relay struct {
	delta      *confluence.DynamicDeltaMultiplier[relayResponse]
	inlet      confluence.Inlet[relayResponse]
	bufferSize int
}

const (
	// defaultRelayBufferSize is the default buffer size for the relay's main streaming
	// pipe and for each streamer's connection to it. All written frames are moved
	// through the main pipe, so the value is relatively large.
	// 1000 * 72 bytes = 72kb
	defaultRelayBufferSize = 1000
	// slowConsumerTimeout is the maximum amount of time the relay will wait for a
	// consumer to receive a frame before dropping the frame.
	slowConsumerTimeout = 20 * time.Millisecond
)

func openRelay(sCtx signal.Context, ins alamos.Instrumentation, bufferSize int) *relay {
	delta := confluence.NewDynamicDeltaMultiplier[relayResponse](
		slowConsumerTimeout,
		ins,
	)
	writes := confluence.NewStream[relayResponse](bufferSize)
	delta.InFrom(writes)
	delta.Flow(
		sCtx,
		confluence.RecoverWithErrOnPanic(),
		confluence.WithRetryOnPanic(),
		confluence.WithAddress("relay"),
	)
	return &relay{delta: delta, inlet: writes, bufferSize: bufferSize}
}

func (r *relay) connect() (confluence.Outlet[relayResponse], func()) {
	frames := confluence.NewStream[relayResponse](r.bufferSize)
	frames.SetInletAddress(address.Newf("%s_storage", address.Rand().String()))
	r.delta.Connect(frames)
	return frames, func() {
		var wg sync.WaitGroup
		// NOTE: This area is a source of concurrency bugs. BE CAREFUL. We need to make
		// sure we drain the frames in a SEPARATE goroutine. This prevents deadlocks
		// inside the relay.
		wg.Go(func() { confluence.Drain(frames) })
		r.delta.Disconnect(frames)
		wg.Wait()
	}
}
