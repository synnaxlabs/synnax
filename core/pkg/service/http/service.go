// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/http/versions"
	"github.com/synnaxlabs/synnax/pkg/service/http/versions/legacy"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the HTTP task config service.
type ServiceConfig struct {
	// DB is the database config records are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Channel holds the channels that config fields read from and write to.
	// [REQUIRED]
	Channel *channel.Service
	alamos.Instrumentation
}

var _ xconfig.Config[ServiceConfig] = ServiceConfig{}

// Override implements xconfig.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("http.service")
	v.NotNil("db", c.DB)
	v.NotNil("channel", c.Channel)
	return v.Error()
}

// Service owns the stored configuration records of the HTTP task types.
type Service struct {
	// Read stores http_read task configuration records.
	Read *config.Service[ReadConfig]
	// Write stores http_write task configuration records.
	Write *config.Service[WriteConfig]
	// Scan stores http_scan task configuration records.
	Scan   *config.Service[ScanConfig]
	closer xio.MultiCloser
}

// OpenService opens the HTTP task config service with the provided configuration.
// If error is nil, the service is ready for use and must be closed by calling Close
// to prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := xconfig.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	channels := channelsOf(cfg.Channel)
	s = &Service{}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.Read, err = config.OpenService(
		ctx, config.ServiceConfig[ReadConfig]{
			DB:              cfg.DB,
			References:      readConfigReferences(),
			Instrumentation: cfg.Instrumentation,
			Type:            "http_read",
			Migrations:      versions.NewReadMigrations(channels),
			LegacyVersion:   legacy.LastVersion,
			Ladder:          versions.ReadConfigLadder,
			Legacy:          &legacy.Read,
			SetEntryKey:     (*ReadConfig).SetKey,
			PatchEntry:      (*ReadConfig).Patch,
			HashEntry:       (*ReadConfig).HashContent,
			ResolveEntry: func(
				ctx context.Context,
				snap gorp.Snapshot,
				_ string,
				c *ReadConfig,
			) (func(context.Context, gorp.Tx) error, error) {
				return nil, v5.ResolveReadConfig(ctx, snap, channels, c)
			},
			ReconcileEntry: func(ctx context.Context, tx gorp.Tx, c *ReadConfig) error {
				return v5.ReconcileReadConfig(ctx, tx, channels, c, cfg.L.Warn)
			},
			ValidateEntry: v5.ValidateReadConfig,
		},
	); !ok(err, s.Read) {
		return nil, err
	}
	if s.Write, err = config.OpenService(
		ctx, config.ServiceConfig[WriteConfig]{
			DB:              cfg.DB,
			References:      writeConfigReferences(),
			Instrumentation: cfg.Instrumentation,
			Type:            "http_write",
			LegacyVersion:   legacy.LastVersion,
			Ladder:          versions.WriteConfigLadder,
			Legacy:          &legacy.Write,
			SetEntryKey:     (*WriteConfig).SetKey,
			PatchEntry:      (*WriteConfig).Patch,
			HashEntry:       (*WriteConfig).HashContent,
			ValidateEntry:   (*WriteConfig).Validate,
		},
	); !ok(err, s.Write) {
		return nil, err
	}
	if s.Scan, err = config.OpenService(
		ctx, config.ServiceConfig[ScanConfig]{
			DB:              cfg.DB,
			Instrumentation: cfg.Instrumentation,
			Type:            "http_scan",
			LegacyVersion:   legacy.LastVersion,
			Ladder:          versions.ScanConfigLadder,
			Legacy:          &legacy.Scan,
			SetEntryKey:     (*ScanConfig).SetKey,
			PatchEntry:      (*ScanConfig).Patch,
			HashEntry:       (*ScanConfig).HashContent,
			ValidateEntry:   (*ScanConfig).Validate,
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

// channelsOf reads the channels that read fields write to.
func channelsOf(svc *channel.Service) v5.Channels {
	return func(
		ctx context.Context,
		snap gorp.Snapshot,
		keys []channel.Key,
	) (map[channel.Key]v5.Channel, error) {
		var chs []channel.Channel
		if err := svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			SkipMissingKeys().
			Entries(&chs).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
		found := make(map[channel.Key]v5.Channel, len(chs))
		for _, ch := range chs {
			found[ch.Key()] = v5.Channel{DataType: ch.DataType, IsIndex: ch.IsIndex}
		}
		return found, nil
	}
}
