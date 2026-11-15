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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/node"
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
	// HostKey is the key of the host node. Rack keys embed it.
	//
	// [REQUIRED]
	HostKey node.Key
	// DB is the gorp database that racks will be stored in.
	//
	// [REQUIRED]
	DB *gorp.DB
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
	c.HostKey = override.Numeric(c.HostKey, other.HostKey)
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
	v.NonZero("host_key", c.HostKey)
	v.NotNil("status", c.Status)
	v.Ternary("driver_role", c.DriverRole == uuid.Nil(), "must be set")
	v.Positive("health", c.HealthCheckInterval)
	return v.Error()
}

type Service struct {
	closer          xio.MultiCloser
	localKeyCounter *kv.AtomicInt64Counter
	monitor         *monitor
	table           *gorp.Table[Key, Rack]
	state           *actions.State[Key, Action]
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
	s = &Service{
		ServiceConfig: cfg,
		indexes:       newIndexes(),
		state:         actions.NewState[Key, Action](cfg.DB),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Rack]{
		DB:         cfg.DB,
		Name:       tableName,
		Indexes:    s.indexes.all(),
		References: s.indexes.references(),
		Migrations: versions.NewMigrations(versions.MigrationsConfig{
			HostKey: cfg.HostKey,
			Status:  cfg.Status,
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
	counterKey := []byte(cfg.HostKey.String() + ".rack.counter")
	if s.localKeyCounter, err = kv.NewCounter(
		ctx,
		cfg.DB.KV(),
		counterKey,
	); !ok(
		err,
		nil,
	) {
		return nil, err
	}
	if err = s.loadEmbeddedRack(ctx); !ok(err, nil) {
		return nil, err
	}
	if s.monitor, err = openMonitor(ctx, s.Child("monitor"), s); !ok(err, s.monitor) {
		return nil, err
	}
	return s, nil
}

// Changes returns the changes that c made to racks.
func (s *Service) Changes(ctx context.Context, c gorp.Commit) gorp.TxReader[Key, Rack] {
	return s.table.Changes(ctx, c)
}

// Actions returns the action batches that Writer attaches to its transactions.
func (s *Service) Actions() *actions.State[Key, Action] { return s.state }

// OnSuspect calls handler with the outage of each rack that stops reporting, once for
// each outage. A handler that returns an error is called again on the next health check
// until it succeeds or the rack reports healthy.
func (s *Service) OnSuspect(handler SuspectHandler) observe.Disconnect {
	return s.monitor.onSuspect(handler)
}

func (s *Service) loadEmbeddedRack(ctx context.Context) error {
	r, err := s.RetrieveOrCreateEmbedded(
		ctx,
		fmt.Sprintf("Node %s Embedded Driver", s.HostKey),
	)
	if err != nil {
		return err
	}
	s.EmbeddedKey = r.Key
	return nil
}

// RetrieveOrCreateEmbedded returns the embedded rack of this Core that was created with
// name, creating it if it does not exist. The rack keeps its identity across renames
// and restarts. acts apply to a new rack before it is created, and to an existing rack
// through Writer.Apply.
func (s *Service) RetrieveOrCreateEmbedded(
	ctx context.Context,
	name string,
	acts ...Action,
) (r Rack, err error) {
	bindingKey := []byte(s.HostKey.String() + ".rack.embedded." + name)
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
				Where(MatchNode(s.HostKey))
		} else {
			return err
		}
		w := s.NewWriter(tx)
		if err = q.Exec(ctx, tx); errors.Is(err, query.ErrNotFound) {
			if r, err = Reduce(Rack{Name: name, Embedded: true}, acts...); err != nil {
				return err
			}
			if err = w.Create(ctx, &r); err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if len(acts) > 0 {
			if err = w.Apply(ctx, r.Key, acts...); err != nil {
				return err
			}
			if err = s.NewRetrieve().
				Where(MatchKeys(r.Key)).
				Entry(&r).
				Exec(ctx, tx); err != nil {
				return err
			}
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
	return Writer{
		tx:         tx,
		newKey:     s.newKey,
		status:     s.Status.NewWriter(tx),
		statuses:   s.Status,
		table:      s.table,
		driverRole: s.DriverRole,
		dispatcher: s.state.Dispatcher(),
	}
}

func (s *Service) newKey(ctx context.Context) (Key, error) {
	n, err := s.localKeyCounter.Add(ctx, 1)
	return NewKey(s.HostKey, uint16(n)), err
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		indexes: s.indexes,
		gorp:    s.table.NewRetrieve(),
	}
}

// newDriverRoleMigration gives the Driver role to every rack created before racks
// received it at creation.
func newDriverRoleMigration(s *Service) gorp.Migration {
	return migrate.New(
		"v0.driver_role",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			return s.table.NewUpdate().
				Where(gorp.Match(func(_ gorp.Context, r *Rack) (bool, error) {
					return r.Role == uuid.Nil(), nil
				})).
				Change(func(_ gorp.Context, r Rack) Rack {
					r.Role = s.DriverRole
					return r
				}).
				Exec(ctx, tx)
		},
	)
}
