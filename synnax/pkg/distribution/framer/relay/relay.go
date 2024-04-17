// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay

import (
	"fmt"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"time"
)

type Config struct {
	alamos.Instrumentation
	Transport    Transport
	HostResolver core.HostResolver
	TS           *ts.DB
	FreeWrites   confluence.Outlet[Response]
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Properties.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.TS = override.Nil(c.TS, other.TS)
	c.FreeWrites = override.Nil(c.FreeWrites, other.FreeWrites)
	return c
}

// Validate implements config.Properties.
func (c Config) Validate() error {
	v := validate.New("relay")
	validate.NotNil(v, "Transport", c.Transport)
	validate.NotNil(v, "HostProvider", c.HostResolver)
	validate.NotNil(v, "TS", c.TS)
	validate.NotNil(v, "FreeWrites", c.FreeWrites)
	return v.Error()
}

type Relay struct {
	ins     alamos.Instrumentation
	delta   *confluence.DynamicDeltaMultiplier[Response]
	demands confluence.Inlet[demand]
	wg      signal.WaitGroup
}

// defaultBuffer is the default buffer size for channels in the relay.
// TODO: Figure out what the optimal buffer size is.
const defaultBuffer = 25

func Open(configs ...Config) (*Relay, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}

	r := &Relay{ins: cfg.Instrumentation}

	tpr := newTapper(cfg)
	demands := confluence.NewStream[demand](defaultBuffer)
	demands.SetOutletAddress("peer-demands")
	demands.Acquire(1)
	tpr.InFrom(demands)
	r.demands = demands

	r.delta = confluence.NewDynamicDeltaMultiplier[Response](20 * time.Millisecond)
	writes := confluence.NewStream[Response](defaultBuffer)
	writes.SetInletAddress("delta")
	writes.SetOutletAddress("taps")
	r.delta.InFrom(writes)
	tpr.OutTo(writes)

	sCtx, _ := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))

	r.delta.Flow(sCtx, confluence.WithAddress("delta"))
	tpr.Flow(sCtx, confluence.WithAddress("tapper"), confluence.CloseInletsOnExit())
	r.wg = sCtx

	startServer(cfg, r.NewStreamer)

	return r, nil
}

func (r *Relay) Close() error {
	r.demands.Close()
	err := r.wg.Wait()
	return err
}

func (r *Relay) connectToDelta(buf int) (confluence.Outlet[Response], observe.Disconnect) {
	var (
		data = confluence.NewStream[Response](buf)
		addr = address.Newf(fmt.Sprintf("%s-%s", r.ins.Meta.Path, address.Rand().String()))
	)
	data.SetInletAddress(addr)
	r.delta.Connect(data)
	return data, func() {
		// NOTE: This area is a source of concurrency bugs. BE CAREFUL. We need to make
		// sure we drain the frames in a SEPARATE goroutine. This prevents deadlocks
		// inside the relay.
		c := make(chan struct{})
		go func() {
			confluence.Drain[Response](data)
			close(c)
		}()
		r.delta.Disconnect(data)
		<-c
	}
}
