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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device/versions"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Service.
type ServiceConfig struct {
	// DB is the database devices are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Status stores each device's status.
	// [REQUIRED]
	Status *status.Service
	// Signals publishes sy_ni_device_set and sy_ni_device_delete.
	// [OPTIONAL]
	Signals *signals.Provider
	// Instrumentation is used for logging, tracing, and metrics.
	// [OPTIONAL]
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Status = override.Nil(c.Status, other.Status)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("ni.device")
	v.NotNil("db", c.DB)
	v.NotNil("status", c.Status)
	return v.Error()
}

// Service stores NI devices.
type Service struct {
	cfg     ServiceConfig
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Device]
	indexes indexes
}

// OpenService opens a Service. The service must be closed by calling Close.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg, indexes: newIndexes()}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Device]{
		DB:              cfg.DB,
		Name:            tableName,
		Migrations:      versions.Migrations,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
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
	}
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe to
// call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// NewWriter opens a writer that runs its operations in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:       tx,
		status:   s.cfg.Status.NewWriter(tx),
		statuses: s.cfg.Status,
		table:    s.table,
	}
}

// NewRetrieve opens a query over the stored devices.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		indexes: s.indexes,
	}
}
