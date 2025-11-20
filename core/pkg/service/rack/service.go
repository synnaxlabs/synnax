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
	"io"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
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
	HostProvider cluster.HostProvider
	// Status is used to define and process statuses for racks.
	// [REQUIRED]
	Status *status.Service
	// Signals is used to propagate rack changes through the Synnax signals' channel
	// communication mechanism.
	// [OPTIONAL]
	Signals *signals.Provider
	// HealthCheckInterval specifies the interval at which the rack service will check
	// that it has received a status update from a rack.
	// [OPTIONAL]
	HealthCheckInterval telem.TimeSpan
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening a rack service. Note
	// that this configuration is not valid. See the Config documentation for more
	// details on which fields must be set.
	DefaultConfig = Config{HealthCheckInterval: 5 * telem.Second}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Status = override.Nil(c.Status, other.Status)
	c.HealthCheckInterval = override.Numeric(c.HealthCheckInterval, other.HealthCheckInterval)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("hardware.rack")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "host", c.HostProvider)
	validate.NotNil(v, "status", c.Status)
	validate.Positive(v, "health", c.HealthCheckInterval)
	return v.Error()
}

type Service struct {
	Config
	EmbeddedKey     Key
	keyMu           *sync.Mutex
	localKeyCounter *kv.AtomicInt64Counter
	shutdownSignals io.Closer
	group           group.Group
	monitor         *monitor
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
	s.monitor, err = openMonitor(s.Child("monitor"), s)
	if err != nil {
		return nil, err
	}
	if cfg.Signals != nil {
		s.shutdownSignals, err = signals.PublishFromGorp[Key](
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigNumeric[Key, Rack](cfg.DB, telem.Uint32T),
		)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Service) OnSuspect(handler func(ctx context.Context, status Status)) observe.Disconnect {
	return s.monitor.OnChange(handler)
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
	c := errors.NewCatcher(errors.WithAggregation())
	c.Exec(s.monitor.Close)
	if s.shutdownSignals == nil {
		return nil
	}
	c.Exec(s.shutdownSignals.Close)
	return c.Error()
}

func (s *Service) RetrieveStatus(ctx context.Context, key Key) (Status, error) {
	var stat Status
	if err := status.NewRetrieve[StatusDetails](s.Status).
		WhereKeys(OntologyID(key).String()).
		Entry(&stat).
		Exec(ctx, nil); err != nil {
		return Status{}, err
	}
	return stat, nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.DB, tx)
	return Writer{
		tx:         tx,
		otg:        s.Ontology.NewWriter(tx),
		newKey:     s.newKey,
		newTaskKey: s.newTaskKey,
		group:      s.group,
		status:     status.NewWriter[StatusDetails](s.Status, tx),
	}
}

func (s *Service) newKey() (Key, error) {
	n, err := s.localKeyCounter.Add(1)
	return NewKey(s.HostProvider.HostKey(), uint16(n)), err
}

func (s *Service) newTaskKey(ctx context.Context, rackKey Key) (next uint32, err error) {
	s.keyMu.Lock()
	defer s.keyMu.Unlock()
	return next, gorp.NewUpdate[Key, Rack]().WhereKeys(rackKey).Change(func(_ gorp.Context, r Rack) Rack {
		r.TaskCounter += 1
		next = r.TaskCounter
		return r
	}).Exec(ctx, s.DB)
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		otg:          s.Ontology,
		baseTX:       s.DB,
		gorp:         gorp.NewRetrieve[Key, Rack](),
		hostProvider: s.HostProvider,
	}
}
