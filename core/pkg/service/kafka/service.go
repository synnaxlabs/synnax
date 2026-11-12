// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/kafka/versions"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

const (
	// ScanTaskType is the type of the internal task that tests and watches cluster
	// connections.
	ScanTaskType = "kafka_scan"
	// ReadTaskType is the type of the task that consumes a topic into channels.
	ReadTaskType = "kafka_read"
	// WriteTaskType is the type of the task that produces channel samples to a topic.
	WriteTaskType = "kafka_write"
)

// ServiceConfig is the configuration for opening the Kafka task config service.
type ServiceConfig struct {
	// DB is the database config records are stored in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Channel holds the channels that config fields read from and write to.
	//
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
	v := validate.New("kafka.service")
	v.NotNil("db", c.DB)
	v.NotNil("channel", c.Channel)
	return v.Error()
}

// Service owns the stored configuration records of the Kafka task types.
type Service struct {
	// Read stores kafka_read task configuration records.
	Read *config.Service[ReadConfig]
	// Write stores kafka_write task configuration records.
	Write *config.Service[WriteConfig]
	// Scan stores kafka_scan task configuration records.
	Scan   *config.Service[ScanConfig]
	closer xio.MultiCloser
}

// OpenService opens the Kafka task config service with the provided configuration.
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
			DB:              cfg.DB,
			References:      readConfigReferences(),
			Instrumentation: cfg.Instrumentation,
			Type:            ReadTaskType,
			Ladder:          versions.ReadConfigLadder,
			SetEntryKey:     (*ReadConfig).SetKey,
			PatchEntry:      (*ReadConfig).Patch,
			HashEntry:       (*ReadConfig).HashContent,
			ValidateEntry:   (*ReadConfig).Validate,
		},
	); !ok(err, s.Read) {
		return nil, err
	}
	if s.Write, err = config.OpenService(
		ctx, config.ServiceConfig[WriteConfig]{
			DB:              cfg.DB,
			References:      writeConfigReferences(),
			Instrumentation: cfg.Instrumentation,
			Type:            WriteTaskType,
			Ladder:          versions.WriteConfigLadder,
			Migrations:      []gorp.Migration{newWriteMigration(cfg.Channel)},
			SetEntryKey:     (*WriteConfig).SetKey,
			PatchEntry:      (*WriteConfig).Patch,
			HashEntry:       (*WriteConfig).HashContent,
			ResolveEntry: func(
				ctx context.Context,
				snap gorp.Snapshot,
				_ string,
				e *WriteConfig,
			) (func(context.Context, gorp.Tx) error, error) {
				return nil, resolveWriteConfig(ctx, snap, cfg.Channel, e)
			},
			ReconcileEntry: func(ctx context.Context, tx gorp.Tx, e *WriteConfig) error {
				return reconcileWriteConfig(
					ctx,
					tx,
					cfg.Channel,
					cfg.Instrumentation,
					e,
				)
			},
			ValidateEntry: validateWriteConfig,
		},
	); !ok(err, s.Write) {
		return nil, err
	}
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
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// Stores returns the config stores the service owns, for registry assembly.
func (s *Service) Stores() []config.Store {
	return []config.Store{s.Read, s.Write, s.Scan}
}
