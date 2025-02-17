// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"context"
	"fmt"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"io"
	"sync"
)

// Config is the configuration for creating a Service.
type Config struct {
	alamos.Instrumentation
	// DB is the gorp database that racks will be stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between racks and other resources
	// in the Synnax cluster.
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create rack related groups of ontology resources.
	// [REQUIRED]
	Group *group.Service
	// HostProvider is used to assign keys to racks.
	// [REQUIRED]
	HostProvider core.HostProvider
	// Signals is used to propagate rack changes through the Synnax signals' channel
	// communication mechanism.
	// [OPTIONAL]
	Signals *signals.Provider
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening a rack service. Note
	// that this configuration is not valid. See the Config documentation for more
	// details on which fields must be set.
	DefaultConfig = Config{}
)

// Override implements config.Properties.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Properties.
func (c Config) Validate() error {
	v := validate.New("hardware.rack")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "host", c.HostProvider)
	return v.Error()
}

type Service struct {
	Config
	EmbeddedKey     Key
	localKeyCounter *kv.AtomicInt64Counter
	shutdownSignals io.Closer
	group           group.Group
	keyMu           *sync.Mutex
}

const localKeyCounterSuffix = ".rack.counter"

const groupName = "Devices"

func OpenService(ctx context.Context, configs ...Config) (s *Service, err error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, groupName, ontology.RootID)
	if err != nil {
		return
	}
	counterKey := []byte(cfg.HostProvider.HostKey().String() + localKeyCounterSuffix)
	c, err := kv.OpenCounter(ctx, cfg.DB, counterKey)
	if err != nil {
		return nil, err
	}
	s = &Service{Config: cfg, localKeyCounter: c, group: g, keyMu: &sync.Mutex{}}
	if err = s.loadEmbeddedRack(ctx); err != nil {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)

	if cfg.Signals != nil {
		cdcS, err := signals.PublishFromGorp[Key](
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigPureNumeric[Key, Rack](cfg.DB, telem.Uint32T),
		)
		s.shutdownSignals = cdcS
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Service) loadEmbeddedRack(ctx context.Context) error {
	var embeddedRack Rack

	// Check if a v1 rack exists.
	v1RackName := fmt.Sprintf("sy_node_%s_rack", s.HostProvider.HostKey())
	err := s.NewRetrieve().WhereNames(v1RackName).Entry(&embeddedRack).Exec(ctx, s.DB)
	isNotFound := errors.Is(err, query.NotFound)
	if err != nil && !isNotFound {
		return err
	}

	// If a v1 rack does not exist, check if a v2 rack exists.
	if isNotFound {
		err = s.NewRetrieve().
			WhereEmbedded(true, gorp.Required()).
			WhereNode(s.HostProvider.HostKey(), gorp.Required()).
			Entry(&embeddedRack).Exec(ctx, s.DB)
		if err != nil && !errors.Is(err, query.NotFound) {
			return err
		}
	}

	// The key will get populated if a rack exists, in which case this will be an update.
	embeddedRack.Name = fmt.Sprintf("Node %s Embedded Driver", s.HostProvider.HostKey())
	embeddedRack.Embedded = true
	err = s.NewWriter(nil).Create(ctx, &embeddedRack)
	s.EmbeddedKey = embeddedRack.Key
	return err
}

func (s *Service) Close() error {
	if s.shutdownSignals == nil {
		return nil
	}
	return s.shutdownSignals.Close()
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:  gorp.OverrideTx(s.DB, tx),
		otg: s.Ontology.NewWriter(tx),
		newKey: func() (Key, error) {
			n, err := s.localKeyCounter.Add(1)
			return NewKey(s.HostProvider.HostKey(), uint16(n)), err
		},
		group: s.group,
		keyMu: s.keyMu,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		otg:          s.Ontology,
		baseTX:       s.DB,
		gorp:         gorp.NewRetrieve[Key, Rack](),
		hostProvider: s.HostProvider,
	}
}
