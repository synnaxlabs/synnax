// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package device implements types and services for managing physical pieces of hardware
// in Synnax. This includes creating, retrieving, and updating devices.
package device

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device/versions"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for creating a device service.
type ServiceConfig struct {
	// DB is the Gorp database that devices will be stored in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Status is used to define and process statuses for devices.
	//
	// [REQUIRED]
	Status *status.Service
	// Signals is used to propagate device changes through the Synnax signals' channel
	// communication mechanism.
	//
	// [OPTIONAL]
	Signals *signals.Provider
	// Rack is used to retrieve and manage racks.
	//
	// [REQUIRED]
	Rack *rack.Service
	// References are the references that devices of one make hold in their properties.
	//
	// [OPTIONAL]
	References []gorp.Reference[Key, Device]
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override overrides parts of c with the valid parts of other.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Status = override.Nil(c.Status, other.Status)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.References = override.Slice(c.References, other.References)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate determines whether the configuration can be used for creating a device
// service.
func (c ServiceConfig) Validate() error {
	v := validate.New("device")
	v.NotNil("db", c.DB)
	v.NotNil("status", c.Status)
	v.NotNil("rack", c.Rack)
	return v.Error()
}

// Service is the main entrypoint for managing devices within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting devices. It also
// provides mechanisms for listening to changes in devices.
type Service struct {
	cfg     ServiceConfig
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Device]
	indexes indexes
	state   *actions.State[Key, Action]
}

// OpenService opens a new device service using the provided configuration. If error
// is nil, the service is ready for use and must be closed by calling Close to
// prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:     cfg,
		indexes: newIndexes(),
		state:   actions.NewState[Key, Action](cfg.DB),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Device]{
		DB:         cfg.DB,
		Name:       tableName,
		Indexes:    s.indexes.all(),
		References: append(s.indexes.references(), cfg.References...),
		Migrations: versions.NewMigrations(
			versions.MigrationsConfig{Status: cfg.Status},
		),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	disconnect := cfg.Rack.OnSuspect(s.onSuspectRack)
	ok(nil, xio.NoFailCloserFunc(disconnect))
	if cfg.Signals != nil {
		if err = cfg.Signals.PublishTable(
			ctx,
			signals.TableConfig[Key, Device]{
				Changes:  s.table.Changes,
				Resource: ResourceID,
			},
		); !ok(err, nil) {
			return nil, err
		}
		// The Driver reads sy_device_set as devices, so actions take their own channel.
		if err = cfg.Signals.PublishActions(ctx, signals.ActionConfig[Key, Action]{
			Name:     "sy_device_action_set",
			Actions:  s.state,
			Resource: ResourceID,
		}); !ok(err, nil) {
			return nil, err
		}
	}
	return s, nil
}

// OnAction calls handler with each action batch that a transaction attaches, after the
// transaction commits. handler runs as a reader of gorp.DB.Observe does, so it must not
// commit on its own goroutine. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnChange(handler)
}

// Close closes the device service and releases any resources that it may have acquired.
func (s *Service) Close() error { return s.closer.Close() }

// NewWriter opens a new Writer to create, update, and delete devices in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		status:     s.cfg.Status.NewWriter(tx),
		statuses:   s.cfg.Status,
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a new Retrieve query to fetch devices from the database.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		indexes: s.indexes,
		gorp:    s.table.NewRetrieve(),
	}
}

// Observe returns an observable that notifies callers of changes to devices.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Device]] {
	return s.table.Observe()
}

func (s *Service) onSuspectRack(ctx context.Context, rackStat rack.Status) error {
	rackKey, err := rack.KeyFromResourceID(*rackStat.Owner)
	if err != nil {
		return err
	}
	var devices []Device
	if err := s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.NewRetrieve().
			Where(MatchRacks(rackKey)).
			Entries(&devices).
			Exec(ctx, snap)
	}); err != nil {
		return errors.Wrap(err, "failed to retrieve devices on suspect rack")
	}
	statuses := make([]Status, len(devices))
	for i, device := range devices {
		statuses[i] = Status{
			Owner:       new(device.ResourceID()),
			Name:        device.Name,
			Time:        telem.Now(),
			Variant:     rackStat.Variant,
			Message:     rackStat.Message,
			Description: rackStat.Description,
			Details:     StatusDetails{Rack: rackKey},
		}
	}
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return s.cfg.Status.NewWriter(tx).SetMany(ctx, &statuses)
	})
}
