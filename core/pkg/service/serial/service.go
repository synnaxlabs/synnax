// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package serial

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/bus"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the serial task config service.
type ServiceConfig struct {
	// DB is the database config records are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Resolver checks read and write configs against their library.
	// [REQUIRED]
	Resolver bus.Resolver
	alamos.Instrumentation
}

var _ xconfig.Config[ServiceConfig] = ServiceConfig{}

// Override implements xconfig.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Resolver.Stamper.DB = override.Nil(
		c.Resolver.Stamper.DB,
		other.Resolver.Stamper.DB,
	)
	c.Resolver.Stamper.Ontology = override.Nil(
		c.Resolver.Stamper.Ontology,
		other.Resolver.Stamper.Ontology,
	)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("serial.service")
	v.NotNil("db", c.DB)
	v.NotNil("resolver.stamper.db", c.Resolver.Stamper.DB)
	v.NotNil("resolver.stamper.ontology", c.Resolver.Stamper.Ontology)
	return v.Error()
}

// Service owns the stored configuration records of the serial task types.
type Service struct {
	// Read stores serial_read task configuration records.
	Read *config.Service[ReadConfig]
	// Write stores serial_write task configuration records.
	Write *config.Service[WriteConfig]
	// Scan stores serial_scan task configuration records.
	Scan   *config.Service[ScanConfig]
	closer xio.MultiCloser
}

// OpenService opens the serial task config service with the provided configuration.
// If error is nil, the service is ready for use and must be closed by calling Close
// to prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := xconfig.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.Read, err = config.OpenService(
		ctx, config.ServiceConfig[ReadConfig]{
			DB:                 cfg.DB,
			Instrumentation:    cfg.Instrumentation,
			Type:               "serial_read",
			Version:            1,
			SetEntryKey:        (*ReadConfig).SetKey,
			ApplyEntryDefaults: (*ReadConfig).ApplyDefaults,
			ValidateEntry:      (*ReadConfig).Validate,
			ResolveEntry: func(
				ctx context.Context,
				tx gorp.Tx,
				key uuid.UUID,
				c *ReadConfig,
			) error {
				return cfg.Resolver.Read(ctx, tx, key, &c.ReadConfig)
			},
		},
	); !ok(err, s.Read) {
		return nil, err
	}
	if s.Write, err = config.OpenService(
		ctx, config.ServiceConfig[WriteConfig]{
			DB:                 cfg.DB,
			Instrumentation:    cfg.Instrumentation,
			Type:               "serial_write",
			Version:            1,
			SetEntryKey:        (*WriteConfig).SetKey,
			ApplyEntryDefaults: (*WriteConfig).ApplyDefaults,
			ValidateEntry:      (*WriteConfig).Validate,
			ResolveEntry: func(
				ctx context.Context,
				tx gorp.Tx,
				key uuid.UUID,
				c *WriteConfig,
			) error {
				return cfg.Resolver.Write(ctx, tx, key, &c.WriteConfig)
			},
		},
	); !ok(err, s.Write) {
		return nil, err
	}
	if s.Scan, err = config.OpenService(
		ctx, config.ServiceConfig[ScanConfig]{
			DB:                 cfg.DB,
			Instrumentation:    cfg.Instrumentation,
			Type:               "serial_scan",
			Version:            1,
			SetEntryKey:        (*ScanConfig).SetKey,
			ApplyEntryDefaults: (*ScanConfig).ApplyDefaults,
		},
	); !ok(err, s.Scan) {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// Stores returns the config stores the service owns, for registry assembly.
func (s *Service) Stores() []config.Store {
	return []config.Store{s.Read, s.Write, s.Scan}
}
