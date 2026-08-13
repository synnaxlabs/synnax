// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ni

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the NI task config service.
type ServiceConfig struct {
	// DB is the database config records are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to register the config record resource types.
	// [REQUIRED]
	Ontology *ontology.Ontology
	alamos.Instrumentation
}

var _ xconfig.Config[ServiceConfig] = ServiceConfig{}

// Override implements xconfig.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("ni.service")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	return v.Error()
}

// Service owns the stored configuration records of the NI task types.
type Service struct {
	// AnalogRead stores ni_analog_read task configuration records.
	AnalogRead *config.Service[AnalogReadConfig, *AnalogReadConfig]
	// AnalogWrite stores ni_analog_write task configuration records.
	AnalogWrite *config.Service[AnalogWriteConfig, *AnalogWriteConfig]
	// CounterRead stores ni_counter_read task configuration records.
	CounterRead *config.Service[CounterReadConfig, *CounterReadConfig]
	// DigitalRead stores ni_digital_read task configuration records.
	DigitalRead *config.Service[DigitalReadConfig, *DigitalReadConfig]
	// DigitalWrite stores ni_digital_write task configuration records.
	DigitalWrite *config.Service[DigitalWriteConfig, *DigitalWriteConfig]
	// Scanner stores ni_scanner task configuration records.
	Scanner *config.Service[ScanConfig, *ScanConfig]
	closer  xio.MultiCloser
}

// OpenService opens the NI task config service with the provided configuration. If
// error is nil, the service is ready for use and must be closed by calling Close to
// prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := xconfig.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.AnalogRead, err = config.OpenService(
		ctx, config.ServiceConfig[AnalogReadConfig]{
			DB:                 cfg.DB,
			Ontology:           cfg.Ontology,
			Instrumentation:    cfg.Instrumentation,
			Type:               ontology.ResourceTypeNiAnalogRead,
			ApplyEntryDefaults: (*AnalogReadConfig).ApplyDefaults,
			ValidateEntry:      (*AnalogReadConfig).Validate,
		},
	); !ok(err, s.AnalogRead) {
		return nil, err
	}
	if s.AnalogWrite, err = config.OpenService(
		ctx, config.ServiceConfig[AnalogWriteConfig]{
			DB:                 cfg.DB,
			Ontology:           cfg.Ontology,
			Instrumentation:    cfg.Instrumentation,
			Type:               ontology.ResourceTypeNiAnalogWrite,
			ApplyEntryDefaults: (*AnalogWriteConfig).ApplyDefaults,
			ValidateEntry:      (*AnalogWriteConfig).Validate,
		},
	); !ok(err, s.AnalogWrite) {
		return nil, err
	}
	if s.CounterRead, err = config.OpenService(
		ctx, config.ServiceConfig[CounterReadConfig]{
			DB:                 cfg.DB,
			Ontology:           cfg.Ontology,
			Instrumentation:    cfg.Instrumentation,
			Type:               ontology.ResourceTypeNiCounterRead,
			ApplyEntryDefaults: (*CounterReadConfig).ApplyDefaults,
			ValidateEntry:      (*CounterReadConfig).Validate,
		},
	); !ok(err, s.CounterRead) {
		return nil, err
	}
	if s.DigitalRead, err = config.OpenService(
		ctx, config.ServiceConfig[DigitalReadConfig]{
			DB:                 cfg.DB,
			Ontology:           cfg.Ontology,
			Instrumentation:    cfg.Instrumentation,
			Type:               ontology.ResourceTypeNiDigitalRead,
			ApplyEntryDefaults: (*DigitalReadConfig).ApplyDefaults,
		},
	); !ok(err, s.DigitalRead) {
		return nil, err
	}
	if s.DigitalWrite, err = config.OpenService(
		ctx, config.ServiceConfig[DigitalWriteConfig]{
			DB:                 cfg.DB,
			Ontology:           cfg.Ontology,
			Instrumentation:    cfg.Instrumentation,
			Type:               ontology.ResourceTypeNiDigitalWrite,
			ApplyEntryDefaults: (*DigitalWriteConfig).ApplyDefaults,
		},
	); !ok(err, s.DigitalWrite) {
		return nil, err
	}
	if s.Scanner, err = config.OpenService(
		ctx, config.ServiceConfig[ScanConfig]{
			DB:              cfg.DB,
			Ontology:        cfg.Ontology,
			Instrumentation: cfg.Instrumentation,
			Type:            ontology.ResourceTypeNiScanner,
		},
	); !ok(err, s.Scanner) {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// Stores returns the config stores the service owns, for registry assembly.
func (s *Service) Stores() []config.Store {
	return []config.Store{
		s.AnalogRead,
		s.AnalogWrite,
		s.CounterRead,
		s.DigitalRead,
		s.DigitalWrite,
		s.Scanner,
	}
}
