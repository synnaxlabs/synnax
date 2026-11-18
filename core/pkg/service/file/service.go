// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package file

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/file/versions"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the file config service.
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
	v := validate.New("file.service")
	v.NotNil("db", c.DB)
	return v.Error()
}

// Service owns the stored configuration records of the file browser task type.
type Service struct {
	// Browser stores file browser task configuration records.
	Browser *config.Service[BrowserConfig]
	closer  xio.MultiCloser
}

// OpenService opens the file config service with the provided configuration. If error
// is nil, the service is ready for use and must be closed by calling Close to prevent
// resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := xconfig.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.Browser, err = config.OpenService(
		ctx, config.ServiceConfig[BrowserConfig]{
			DB:              cfg.DB,
			Instrumentation: cfg.Instrumentation,
			Type:            browserTaskType,
			Ladder:          versions.BrowserConfigLadder,
			SetEntryKey:     (*BrowserConfig).SetKey,
			PatchEntry:      (*BrowserConfig).Patch,
			HashEntry:       (*BrowserConfig).HashContent,
		},
	); !ok(err, s.Browser) {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe to
// call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// Stores returns the config stores the service owns, for registry assembly.
func (s *Service) Stores() []config.Store { return []config.Store{s.Browser} }
