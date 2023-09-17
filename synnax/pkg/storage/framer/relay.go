// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"io"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type relay struct {
	delta    *confluence.DynamicDeltaMultiplier[Frame]
	inlet    confluence.Inlet[Frame]
	shutdown io.Closer
}

func openRelay(cfg Config) *relay {
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	delta := confluence.NewDynamicDeltaMultiplier[Frame]()
	frames := confluence.NewStream[Frame](1)
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
	frames.SetInletAddress(address.Rand())
	r.delta.Connect(frames)
	return frames, func() {
		r.delta.Disconnect(frames)
		// We need to make sure we drain any remaining frames before exiting to
		// avoid blocking the relay.
		confluence.Drain[Frame](frames)
	}
}

func (r *relay) close() error { return r.shutdown.Close() }
