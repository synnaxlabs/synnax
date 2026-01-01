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
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

type relay struct {
	delta      *confluence.DynamicDeltaMultiplier[Frame]
	inlet      confluence.Inlet[Frame]
	bufferSize int
}

// DBStreamingConfig is the configuration for cesium's streaming mechanisms.
type DBStreamingConfig struct {
	// BufferSize sets the buffer size for the main streaming pipe. All written frames
	// are moved through this pipe, so the value should be relatively large.
	BufferSize int
	// SlowConsumerTimeout sets the maximum amount of time the relay will wait for a
	// consumer to receive a frame before dropping the frame.
	SlowConsumerTimeout time.Duration
}

var (
	_                        config.Config[DBStreamingConfig] = DBStreamingConfig{}
	DefaultDBStreamingConfig                                  = DBStreamingConfig{
		// 1000 * 72 bytes = 72kb
		BufferSize:          1000,
		SlowConsumerTimeout: 20 * time.Millisecond,
	}
)

// Override implements config.Config.
func (sc DBStreamingConfig) Override(other DBStreamingConfig) DBStreamingConfig {
	sc.BufferSize = override.Numeric(sc.BufferSize, other.BufferSize)
	sc.SlowConsumerTimeout = override.Numeric(sc.SlowConsumerTimeout, other.SlowConsumerTimeout)
	return sc
}

func (sc DBStreamingConfig) Validate() error {
	v := validate.New("cesium.db_streaming_config")
	validate.Positive(v, "buffer_size", sc.BufferSize)
	validate.Positive(v, "slow_consumer_timeout", sc.SlowConsumerTimeout)
	return v.Error()
}

func openRelay(
	sCtx signal.Context,
	ins alamos.Instrumentation,
	cfg DBStreamingConfig,
) *relay {
	delta := confluence.NewDynamicDeltaMultiplier[Frame](
		cfg.SlowConsumerTimeout,
		ins,
	)
	writes := confluence.NewStream[Frame](cfg.BufferSize)
	delta.InFrom(writes)
	delta.Flow(
		sCtx,
		confluence.RecoverWithErrOnPanic(),
		confluence.WithRetryOnPanic(),
		confluence.WithAddress("relay"),
	)
	return &relay{delta: delta, inlet: writes}
}

func (r *relay) connect() (confluence.Outlet[Frame], func()) {
	frames := confluence.NewStream[Frame](r.bufferSize)
	frames.SetInletAddress(address.Newf("%s_storage", address.Rand().String()))
	r.delta.Connect(frames)
	return frames, func() {
		var wg sync.WaitGroup
		// NOTE: This area is a source of concurrency bugs. BE CAREFUL. We need to make
		// sure we drain the frames in a SEPARATE goroutine. This prevents deadlocks
		// inside the relay.
		wg.Add(1)
		go func() {
			confluence.Drain(frames)
			wg.Done()
		}()
		r.delta.Disconnect(frames)
		wg.Wait()
	}
}
