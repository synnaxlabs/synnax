// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/sift/versions"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Service.
type ServiceConfig struct {
	alamos.Instrumentation
	// DB is the database config records are stored in.
	//
	// [REQUIRED]
	DB *gorp.DB
}

var _ xconfig.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("sift.service")
	v.NotNil("db", c.DB)
	return v.Error()
}

// Service stores the configuration records of Sift tasks.
type Service struct {
	// Scan stores sift_scan task configuration records.
	Scan *config.Service[ScanConfig]
	// Stream stores sift_stream task configuration records.
	Stream *config.Service[StreamConfig]
	closer xio.MultiCloser
}

// OpenService opens a Service. The caller must close it.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := xconfig.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.Scan, err = config.OpenService(
		ctx, config.ServiceConfig[ScanConfig]{
			DB:              cfg.DB,
			Instrumentation: cfg.Instrumentation,
			Type:            ScanTaskType,
			Ladder:          versions.ScanConfigLadder,
			SetEntryKey:     (*ScanConfig).SetKey,
			PatchEntry:      (*ScanConfig).Patch,
			HashEntry:       (*ScanConfig).HashContent,
			ValidateEntry:   validateScanConfig,
		},
	); !ok(err, s.Scan) {
		return nil, err
	}
	if s.Stream, err = config.OpenService(
		ctx, config.ServiceConfig[StreamConfig]{
			DB:              cfg.DB,
			Instrumentation: cfg.Instrumentation,
			Type:            StreamTaskType,
			Ladder:          versions.StreamConfigLadder,
			References:      streamConfigReferences(),
			SetEntryKey:     (*StreamConfig).SetKey,
			PatchEntry:      (*StreamConfig).Patch,
			HashEntry:       (*StreamConfig).HashContent,
		},
	); !ok(err, s.Stream) {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe to
// call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// Stores returns the config stores the service owns, for registry assembly.
func (s *Service) Stores() []config.Store { return []config.Store{s.Scan, s.Stream} }
