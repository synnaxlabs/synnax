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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	alamos.Instrumentation
	Transport    Transport
	HostResolver core.HostResolver
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
	delta       *confluence.DynamicDeltaMultiplier[Data]
	peerDemands confluence.Inlet[demand]
	writes      confluence.Inlet[Data]
}

func Open(configs ...Config) (*Relay, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}

	s := &Relay{}
	startServer(cfg, s.NewReader)

	peerReceiver := newPeerReceiver(cfg)
	peerDemands := confluence.NewStream[demand](1)
	peerReceiver.InFrom(peerDemands)
	s.peerDemands = peerDemands

	s.delta = confluence.NewDynamicDeltaMultiplier[Data]()
	writes := confluence.NewStream[Data](10)
	s.delta.InFrom(writes)
	peerReceiver.OutTo(writes)
	s.writes = writes

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	s.Closer = signal.NewShutdown(sCtx, cancel)

	s.delta.Flow(sCtx, confluence.WithAddress("delta"))
	peerReceiver.Flow(sCtx, confluence.WithAddress("peerReceiver"))

	return s, nil
}

type Reader = confluence.Segment[ReadRequest, Data]

func (s *Relay) NewReader(keys ...channel.Key) Reader {
	data := confluence.NewStream[Data](10)
	s.delta.Connect(data)
	r := &reader{
		keys: keys,
		addr: address.Address(uuid.NewString()),
	}
	data.SetInletAddress(r.addr)
	r.Source = &r.responses
	r.Sink = &r.requests
	r.requests.OutTo(s.peerDemands)
	r.responses.InFrom(data)
	return r
}

func (s *Relay) Writes() confluence.Inlet[Data] { return s.writes }
