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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"io"
)

type Config struct {
	alamos.Instrumentation
	Transport    Transport
	HostResolver core.HostResolver
	TS           *ts.DB
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.TS = override.Nil(c.TS, other.TS)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("relay")
	validate.NotNil(v, "Transport", c.Transport)
	validate.NotNil(v, "HostResolver", c.HostResolver)
	return v.Error()
}

type Relay struct {
	io.Closer
	delta       *confluence.DynamicDeltaMultiplier[Response]
	peerDemands confluence.Inlet[demand]
	writes      confluence.Inlet[Response]
}

func Open(configs ...Config) (*Relay, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}

	s := &Relay{}
	startServer(cfg, s.NewReader)

	coord := newReceiveCoordinator(cfg)
	peerDemands := confluence.NewStream[demand](1)
	coord.InFrom(peerDemands)
	s.peerDemands = peerDemands

	s.delta = confluence.NewDynamicDeltaMultiplier[Response]()
	writes := confluence.NewStream[Response](1)
	s.delta.InFrom(writes)
	coord.OutTo(writes)
	s.writes = writes

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	s.Closer = signal.NewShutdown(sCtx, cancel)

	s.delta.Flow(sCtx, confluence.WithAddress("delta"))
	coord.Flow(sCtx, confluence.WithAddress("receive-coordinator"))

	return s, nil
}

func (r *Relay) connect(buf int) (confluence.Outlet[Response], func()) {
	data := confluence.NewStream[Response](buf)
	data.SetInletAddress(address.Rand())
	r.delta.Connect(data)
	return data, func() {
		r.delta.Disconnect(data)
		confluence.Drain[Response](data)
	}
}
