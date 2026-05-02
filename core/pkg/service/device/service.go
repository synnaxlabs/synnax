// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/device/migrations/v0"
	v54 "github.com/synnaxlabs/synnax/pkg/service/device/migrations/v54"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ServiceConfig is the configuration for creating a device service.
type ServiceConfig struct {
	// DB is the gorp database that devices will be stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between devices and other resources in
	// the Synnax cluster.
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create device related groups of ontology resources.
	// [REQUIRED]
	Group *group.Service
	// Status is used to define and process statuses for devices.
	// [REQUIRED]
	Status *status.Service
	// Search is the search index for fuzzy searching devices.
	// [REQUIRED]
	Search *search.Index
	// Signals is used to propagate device changes through the Synnax signals' channel
	// communication mechanism.
	// [OPTIONAL]
	Signals *signals.Provider
	// Rack is used to retrieve and manage racks.
	// [REQUIRED]
	Rack *rack.Service
	// Instrumentation is used for logging, tracing, and metrics.
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override overrides parts of c with the valid parts of other.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Status = override.Nil(c.Status, other.Status)
	c.Search = override.Nil(c.Search, other.Search)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate determines whether the configuration can be used for creating a device
// service.
func (c ServiceConfig) Validate() error {
	v := validate.New("hardware.device")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "rack", c.Rack)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

var DefaultServiceConfig = ServiceConfig{}

// Service is the main entrypoint for managing devices within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting devices. It also
// provides mechanisms for listening to changes in devices.
type Service struct {
	cfg    ServiceConfig
	closer xio.MultiCloser
	table  *gorp.Table[string, Device]
	group  group.Group
}

// OpenService opens a new device service using the provided configuration. If error
// is nil, the service is ready for use and must be closed by calling Close to
// prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	v0Mig := v0.Migration(v0.MigrationConfig{Status: cfg.Status})
	if s.table, err = gorp.OpenTable[string, Device](ctx, gorp.TableConfig[string, Device]{
		DB: cfg.DB,
		Migrations: []migrate.Migration{
			v0Mig,
			gorp.CodecMigration[string, v54.Device]("msgpack_to_orc", v0Mig.Key()),
			migrate.WithAddedDeps(
				gorp.NewEntryMigration[string, string, v54.Device, Device](
					"v54_drop_status_parent",
					MigrateDevice,
				),
				"msgpack_to_orc",
			),
		},
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if s.group, err = cfg.Group.CreateOrRetrieve(ctx, "Devices", ontology.RootID); !ok(err, nil) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	disconnect := cfg.Rack.OnSuspect(s.onSuspectRack)
	ok(nil, xio.NoFailCloserFunc(disconnect))
	if cfg.Signals != nil {
		var sig io.Closer
		if sig, err = signals.PublishFromGorp(
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigString[Device](s.table.Observe()),
		); !ok(err, sig) {
			return nil, err
		}
	}
	return s, nil
}

// Close closes the device service and releases any resources that it may have acquired.
func (s *Service) Close() error { return s.closer.Close() }

// RootGroup returns the permanent group for devices. Note that racks will be children
// of this group, and devices will be children of racks (or children of groups that are
// children of racks).
func (s *Service) RootGroup() group.Group { return s.group }

// NewWriter opens a new Writer to create, update, and delete devices. If tx is not nil,
// the writer will use that transaction. If tx is nil, the writer will execute
// operations directly against the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:     tx,
		otg:    s.cfg.Ontology.NewWriter(tx),
		group:  s.group,
		status: status.NewWriter[StatusDetails](s.cfg.Status, tx),
		table:  s.table,
	}
}

// NewRetrieve opens a new Retrieve query to fetch devices from the database.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		search: s.cfg.Search,
		baseTX: s.cfg.DB,
		gorp:   s.table.NewRetrieve(),
	}
}

func (s *Service) onSuspectRack(ctx context.Context, rackStat rack.Status) {
	var devices []Device
	if err := s.NewRetrieve().Where(MatchRacks(rackStat.Details.Rack)).
		Entries(&devices).
		Exec(ctx, nil); err != nil {
		s.cfg.L.Error("failed to retrieve devices on suspect rack", zap.Error(err))
	}
	statuses := make([]status.Status[StatusDetails], len(devices))
	for i, device := range devices {
		statuses[i] = status.Status[StatusDetails]{
			Key:         OntologyID(device.Key).String(),
			Name:        device.Name,
			Time:        telem.Now(),
			Variant:     rackStat.Variant,
			Message:     rackStat.Message,
			Description: rackStat.Description,
			Details:     StatusDetails{Rack: rackStat.Details.Rack, Device: device.Key},
		}
	}
	if err := status.NewWriter[StatusDetails](s.cfg.Status, nil).
		SetMany(ctx, &statuses); err != nil {
		s.cfg.L.Error("failed to set statuses on suspect rack", zap.Error(err))
	}
}
