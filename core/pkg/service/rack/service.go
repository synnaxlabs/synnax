// Copyright 2026 Synnax Labs, Inc.
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
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ServiceConfig is the configuration for creating a Service.
type ServiceConfig struct {
	// HostProvider is used to assign keys to racks.
	// [REQUIRED]
	HostProvider cluster.HostProvider
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
	// Status is used to define and process statuses for racks.
	// [REQUIRED]
	Status *status.Service
	// Signals is used to propagate rack changes through the Synnax signals' channel
	// communication mechanism.
	// [OPTIONAL]
	Signals *signals.Provider
	alamos.Instrumentation
	// HealthCheckInterval specifies the interval at which the rack service will check
	// that it has received a status update from a rack.
	// [OPTIONAL]
	HealthCheckInterval telem.TimeSpan
	// AlertEveryNChecks controls dampening for dead rack alerts. After the initial
	// alert when a rack goes down, subsequent alerts are only fired every N consecutive
	// dead checks. Set to 1 to alert on every check (no dampening).
	// [OPTIONAL - defaults to 12]
	AlertEveryNChecks int
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening a rack service.
	// Note that this configuration is not valid. See the Config documentation for more
	// details on which fields must be set.
	DefaultServiceConfig = ServiceConfig{
		HealthCheckInterval: 5 * telem.Second,
		AlertEveryNChecks:   12,
	}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Status = override.Nil(c.Status, other.Status)
	c.HealthCheckInterval = override.Numeric(c.HealthCheckInterval, other.HealthCheckInterval)
	c.AlertEveryNChecks = override.Numeric(c.AlertEveryNChecks, other.AlertEveryNChecks)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
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
	shutdownSignals io.Closer
	keyMu           *sync.Mutex
	localKeyCounter *kv.AtomicInt64Counter
	monitor         *monitor
	group           group.Group
	ServiceConfig
	EmbeddedKey Key
}

func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, configs...)
	if err != nil {
		return nil, err
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, "Devices", ontology.RootID)
	if err != nil {
		return nil, err
	}
	counterKey := []byte(cfg.HostProvider.HostKey().String() + ".rack.counter")
	c, err := kv.OpenCounter(ctx, cfg.DB, counterKey)
	if err != nil {
		return nil, err
	}
	s := &Service{ServiceConfig: cfg, localKeyCounter: c, group: g, keyMu: &sync.Mutex{}}
	if err = s.loadEmbeddedRack(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateStatusesForExistingRacks(ctx); err != nil {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	if s.monitor, err = openMonitor(s.Child("monitor"), s); err != nil {
		return nil, err
	}
	if cfg.Signals != nil {
		if s.shutdownSignals, err = signals.PublishFromGorp(
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigNumeric[Key, Rack](cfg.DB, telem.Uint32T),
		); err != nil {
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
	isNotFound := errors.Is(err, query.ErrNotFound)
	if err != nil && !isNotFound {
		return err
	}

	// If a v1 rack does not exist, check if a v2 rack exists.
	if isNotFound {
		err = s.NewRetrieve().
			WhereEmbedded(true, gorp.Required()).
			WhereNode(s.HostProvider.HostKey(), gorp.Required()).
			Entry(&embeddedRack).Exec(ctx, s.DB)
		if err != nil && !errors.Is(err, query.ErrNotFound) {
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

func (s *Service) migrateStatusesForExistingRacks(ctx context.Context) error {
	var racks []Rack
	if err := s.NewRetrieve().Entries(&racks).Exec(ctx, s.DB); err != nil {
		return err
	}
	if len(racks) == 0 {
		return nil
	}
	statusKeys := make([]string, len(racks))
	for i, r := range racks {
		statusKeys[i] = OntologyID(r.Key).String()
	}
	var existingStatuses []status.Status[StatusDetails]
	if err := status.NewRetrieve[StatusDetails](s.Status).
		WhereKeys(statusKeys...).
		Entries(&existingStatuses).
		Exec(ctx, nil); err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	existingKeys := make(map[string]bool)
	for _, stat := range existingStatuses {
		existingKeys[stat.Key] = true
	}
	var missingStatuses []status.Status[StatusDetails]
	for _, r := range racks {
		key := OntologyID(r.Key).String()
		if !existingKeys[key] {
			missingStatuses = append(missingStatuses, status.Status[StatusDetails]{
				Key:     key,
				Name:    r.Name,
				Time:    telem.Now(),
				Variant: xstatus.VariantWarning,
				Message: "Status unknown",
				Details: StatusDetails{Rack: r.Key},
			})
		}
	}
	if len(missingStatuses) == 0 {
		return nil
	}
	s.L.Info("creating unknown statuses for existing racks", zap.Int("count", len(missingStatuses)))
	return status.NewWriter[StatusDetails](s.Status, nil).SetMany(ctx, &missingStatuses)
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

func (s *Service) RetrieveStatus(ctx context.Context, key Key) (status.Status[StatusDetails], error) {
	var stat status.Status[StatusDetails]
	if err := status.NewRetrieve[StatusDetails](s.Status).
		WhereKeys(OntologyID(key).String()).
		Entry(&stat).
		Exec(ctx, nil); err != nil {
		return status.Status[StatusDetails]{}, err
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
