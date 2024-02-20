/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package rack

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"io"
)

// Config is the configuration for creating a Service.
type Config struct {
	DB           *gorp.DB
	Ontology     *ontology.Ontology
	Group        *group.Service
	HostProvider core.HostProvider
	Signals      *signals.Provider
	Channel      channel.Writeable
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Properties.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Channel = override.Nil(c.Channel, other.Channel)
	return c
}

// Validate implements config.Properties.
func (c Config) Validate() error {
	v := validate.New("rack")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "host", c.HostProvider)
	validate.NotNil(v, "signals", c.Signals)
	validate.NotNil(v, "channel", c.Channel)
	return v.Error()
}

type Service struct {
	Config
	group           group.Group
	localKeyCounter *kv.AtomicInt64Counter
	shutdownSignals io.Closer
}

const groupName = "Racks"
const localKeyCounterSuffix = ".rack.counter"

func OpenService(ctx context.Context, configs ...Config) (s *Service, err error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}

	g, err := cfg.Group.CreateOrRetrieve(ctx, groupName, ontology.RootID)
	if err != nil {
		return nil, err
	}

	counterKey := []byte(cfg.HostProvider.HostKey().String() + localKeyCounterSuffix)
	c, err := kv.OpenCounter(ctx, cfg.DB, counterKey)
	if err != nil {
		return nil, err
	}

	s = &Service{Config: cfg, group: g, localKeyCounter: c}
	cfg.Ontology.RegisterService(s)

	if cfg.Signals == nil {
		return
	}

	cdcS, err := signals.SubscribeToGorp(ctx, cfg.Signals, signals.GorpConfigPureNumeric[Key, Rack](cfg.DB, telem.Uint32T))
	s.shutdownSignals = cdcS

	return s, nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:    gorp.OverrideTx(s.DB, tx),
		otg:   s.Ontology.NewWriter(tx),
		group: s.group,
		newKey: func() (Key, error) {
			n, err := s.localKeyCounter.Add(1)
			return NewKey(s.HostProvider.HostKey(), uint16(n)), err
		},
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		otg:    s.Ontology,
		baseTX: s.DB,
		gorp:   gorp.NewRetrieve[Key, Rack](),
	}
}
