// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/arc/task/versions/legacy"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the Arc task config service.
type ServiceConfig struct {
	// DB is the database config records are stored in.
	// [REQUIRED]
	DB *gorp.DB
	alamos.Instrumentation
}

var _ xconfig.Config[ServiceConfig] = ServiceConfig{}

// Override implements xconfig.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("arc.task.service")
	validate.NotNil(v, "db", c.DB)
	return v.Error()
}

// Service owns the stored configuration records of the Arc task type.
type Service struct {
	// Config stores arc task configuration records.
	Config *config.Service[Config]
	closer xio.MultiCloser
}

// OpenService opens the Arc task config service with the provided configuration.
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
	if s.Config, err = config.OpenService(
		ctx, config.ServiceConfig[Config]{
			DB:                 cfg.DB,
			Instrumentation:    cfg.Instrumentation,
			Type:               Type,
			Version:            legacy.LastVersion + 1,
			Legacy:             &legacy.Config,
			SetEntryKey:        (*Config).SetKey,
			ApplyEntryDefaults: (*Config).ApplyDefaults,
			ValidateEntry:      (*Config).Validate,
		},
	); !ok(err, s.Config) {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// Stores returns the config stores the service owns, for registry assembly.
func (s *Service) Stores() []config.Store {
	return []config.Store{s.Config}
}
