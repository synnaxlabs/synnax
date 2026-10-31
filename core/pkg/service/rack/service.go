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
	"encoding/binary"
	"fmt"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack/versions"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for creating a Service.
type ServiceConfig struct {
	// HostProvider is used to assign keys to racks.
	//
	// [REQUIRED]
	HostProvider cluster.HostProvider
	// DB is the gorp database that racks will be stored in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between racks and other resources
	// in the Synnax cluster.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create rack related groups of ontology resources.
	//
	// [REQUIRED]
	Group *group.Service
	// Status is used to define and process statuses for racks.
	//
	// [REQUIRED]
	Status *status.Service
	// DriverRole is the role each new rack receives.
	//
	// [REQUIRED]
	DriverRole role.Key
	alamos.Instrumentation
	// HealthCheckInterval specifies the interval at which the rack service will check
	// that it has received a status update from a rack.
	//
	// [OPTIONAL]
	HealthCheckInterval telem.TimeSpan
	// Now returns the current time. It is used by the rack health monitor to decide
	// whether a rack has gone stale.
	//
	// [OPTIONAL - defaults to telem.Now]
	Now func() telem.TimeStamp
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening a rack service.
	// Note that this configuration is not valid. See the Config documentation for more
	// details on which fields must be set.
	DefaultServiceConfig = ServiceConfig{
		HealthCheckInterval: 5 * telem.Second,
		Now:                 telem.Now,
	}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	c.Status = override.Nil(c.Status, other.Status)
	c.DriverRole = override.UUID(c.DriverRole, other.DriverRole)
	c.HealthCheckInterval = override.Numeric(
		c.HealthCheckInterval,
		other.HealthCheckInterval,
	)
	c.Now = override.Nil(c.Now, other.Now)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("rack")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("group", c.Group)
	v.NotNil("host", c.HostProvider)
	v.NotNil("status", c.Status)
	v.Ternary("driver_role", c.DriverRole == uuid.Nil(), "must be set")
	v.Positive("health", c.HealthCheckInterval)
	return v.Error()
}

type Service struct {
	closer          xio.MultiCloser
	localKeyCounter *kv.AtomicInt64Counter
	group           group.Group
	monitor         *monitor
	table           *gorp.Table[Key, Rack]
	indexes         indexes
	ServiceConfig
	EmbeddedKey Key
}

func OpenService(
	ctx context.Context,
	configs ...ServiceConfig,
) (s *Service, err error) {
	cfg, err := config.New(DefaultServiceConfig, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{ServiceConfig: cfg, indexes: newIndexes()}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Rack]{
		DB:      cfg.DB,
		Indexes: s.indexes.all(),
		Migrations: versions.NewMigrations(versions.MigrationsConfig{
			HostProvider: cfg.HostProvider,
			Status:       cfg.Status,
		}),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if err = gorp.Migrate(ctx, gorp.MigrateConfig{
		Instrumentation: cfg.Instrumentation,
		DB:              cfg.DB,
		Namespace:       "rack.roles",
		Migrations:      []gorp.Migration{newDriverRoleMigration(s)},
	}); !ok(err, nil) {
		return nil, err
	}
	if s.group, err = cfg.Group.CreateOrRetrieve(
		ctx,
		"Devices",
		ontology.RootID,
	); !ok(
		err,
		nil,
	) {
		return nil, err
	}
	counterKey := []byte(cfg.HostProvider.HostKey().String() + ".rack.counter")
	if s.localKeyCounter, err = kv.NewCounter(ctx, cfg.DB, counterKey); !ok(err, nil) {
		return nil, err
	}
	if err = s.loadEmbeddedRack(ctx); !ok(err, nil) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	if s.monitor, err = openMonitor(ctx, s.Child("monitor"), s); !ok(err, s.monitor) {
		return nil, err
	}
	return s, nil
}

// Observe returns an observable that notifies callers of changes to rack entries.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Rack]] {
	return s.table.Observe()
}

// OnSuspect calls handler with the outage of each rack that stops reporting, once for
// each outage. A handler that returns an error is called again on the next health check
// until it succeeds or the rack reports healthy.
func (s *Service) OnSuspect(handler SuspectHandler) observe.Disconnect {
	return s.monitor.onSuspect(handler)
}

func (s *Service) loadEmbeddedRack(ctx context.Context) error {
	r, err := s.RetrieveOrCreateEmbedded(
		ctx,
		fmt.Sprintf("Node %s Embedded Driver", s.HostProvider.HostKey()),
		nil,
	)
	if err != nil {
		return err
	}
	s.EmbeddedKey = r.Key
	return nil
}

// RetrieveOrCreateEmbedded returns the embedded rack of this Core that was created with
// name, creating it if it does not exist. The rack keeps its identity across renames
// and restarts. edit, if non-nil, changes the rack before it is saved.
func (s *Service) RetrieveOrCreateEmbedded(
	ctx context.Context,
	name string,
	edit func(*Rack),
) (r Rack, err error) {
	bindingKey := []byte(s.HostProvider.HostKey().String() + ".rack.embedded." + name)
	err = s.DB.WithTx(ctx, func(tx gorp.Tx) error {
		q := s.NewRetrieve().Entry(&r)
		b, closer, err := tx.Get(ctx, bindingKey)
		if err == nil {
			q = q.Where(MatchKeys(Key(binary.LittleEndian.Uint32(b))))
			if err = closer.Close(); err != nil {
				return err
			}
		} else if errors.Is(err, query.ErrNotFound) {
			q = q.
				Where(MatchNames(name)).
				Where(MatchEmbedded(true)).
				Where(MatchNode(s.HostProvider.HostKey()))
		} else {
			return err
		}
		if err = q.Exec(ctx, tx); errors.Is(err, query.ErrNotFound) {
			r = Rack{Name: name}
		} else if err != nil {
			return err
		}
		r.Embedded = true
		if edit != nil {
			edit(&r)
		}
		if err = s.NewWriter(tx).Create(ctx, &r); err != nil {
			return err
		}
		return tx.Set(
			ctx,
			bindingKey,
			binary.LittleEndian.AppendUint32(nil, uint32(r.Key)),
		)
	})
	if err != nil {
		return Rack{}, err
	}
	return r, nil
}

func (s *Service) Close() error { return s.closer.Close() }

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.DB, tx)
	return Writer{
		tx:         tx,
		otg:        s.Ontology.NewWriter(tx),
		newKey:     s.newKey,
		group:      s.group,
		status:     s.Status.NewWriter(tx),
		table:      s.table,
		driverRole: s.DriverRole,
	}
}

func (s *Service) newKey(ctx context.Context) (Key, error) {
	n, err := s.localKeyCounter.Add(ctx, 1)
	return NewKey(s.HostProvider.HostKey(), uint16(n)), err
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		indexes:      s.indexes,
		baseTX:       s.DB,
		gorp:         s.table.NewRetrieve(),
		hostProvider: s.HostProvider,
	}
}

// newDriverRoleMigration gives the Driver role to every rack created before racks
// received it at creation.
func newDriverRoleMigration(s *Service) gorp.Migration {
	return migrate.New(
		"v0.driver_role",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var racks []Rack
			if err := s.NewRetrieve().Entries(&racks).Exec(ctx, tx); err != nil {
				return err
			}
			w := s.Ontology.NewWriter(tx)
			for _, r := range racks {
				if err := w.DefineRelationships(
					ctx,
					role.OntologyID(s.DriverRole),
					ontology.RelationshipTypeParentOf,
					r.OntologyID(),
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
