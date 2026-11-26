// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package serial stores the configs of the tasks that read and send ICD messages on a
// serial port.
package serial

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	"github.com/synnaxlabs/synnax/pkg/service/serial/versions"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/content"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

const (
	// ReadTaskType is the type of the task that reads ICD messages from a serial port.
	ReadTaskType = "serial_read"
	// WriteTaskType is the type of the task that sends ICD messages on a serial port.
	WriteTaskType = "serial_write"
)

// ServiceConfig is the configuration for opening the serial task config service.
type ServiceConfig struct {
	// DB is the database config records are stored in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Channel holds the channels that config bindings read from and write to.
	//
	// [REQUIRED]
	Channel *channel.Service
	// ICD holds the ICDs whose messages configs read and send.
	//
	// [REQUIRED]
	ICD *icd.Service
	alamos.Instrumentation
}

var _ xconfig.Config[ServiceConfig] = ServiceConfig{}

// Override implements xconfig.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.ICD = override.Nil(c.ICD, other.ICD)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("serial.service")
	v.NotNil("db", c.DB)
	v.NotNil("channel", c.Channel)
	v.NotNil("icd", c.ICD)
	return v.Error()
}

// Service owns the stored configuration records of the serial task types.
type Service struct {
	// Read stores serial_read task configuration records.
	Read *config.Service[ReadConfig]
	// Write stores serial_write task configuration records.
	Write  *config.Service[WriteConfig]
	reads  read.Resolver
	writes write.Resolver
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
	channels := icd.ReadChannels(cfg.Channel)
	s = &Service{
		reads:  read.NewResolver(cfg.ICD, channels),
		writes: write.NewResolver(cfg.ICD, channels),
	}
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
			HashReferences:  s.hashRead,
			HashedTables:    []string{"icd"},
			ResolveEntry:    s.resolveRead,
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
			SetEntryKey:     (*WriteConfig).SetKey,
			PatchEntry:      (*WriteConfig).Patch,
			HashEntry:       (*WriteConfig).HashContent,
			HashReferences:  s.hashWrite,
			HashedTables:    []string{"icd"},
			ResolveEntry:    s.resolveWrite,
			ValidateEntry:   (*WriteConfig).Validate,
		},
	); !ok(err, s.Write) {
		return nil, err
	}
	return s, nil
}

// resolveRead checks each entry of cfg against its ICD and its channels, read through
// snap.
func (s *Service) resolveRead(
	ctx context.Context,
	snap gorp.Snapshot,
	_ string,
	cfg *ReadConfig,
) (func(context.Context, gorp.Tx) error, error) {
	return nil, s.reads.Check(ctx, snap, read.Entries(cfg.Entries), nil)
}

// hashRead writes to h the framing and the payload of each ICD message that an entry
// of cfg reads, so an edit to either changes the config hash of the task.
func (s *Service) hashRead(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *ReadConfig,
	h *content.Hash,
) error {
	return s.reads.Hash(ctx, snap, h, read.Entries(cfg.Entries))
}

// resolveWrite checks each target of cfg against its ICD and its channels, read
// through snap.
func (s *Service) resolveWrite(
	ctx context.Context,
	snap gorp.Snapshot,
	_ string,
	cfg *WriteConfig,
) (func(context.Context, gorp.Tx) error, error) {
	return nil, s.writes.Check(ctx, snap, write.Targets(cfg.Targets), nil)
}

// hashWrite writes to h the framing and the payload of the ICD message that each
// target of cfg sends, so an edit to either changes the config hash of the task.
func (s *Service) hashWrite(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *WriteConfig,
	h *content.Hash,
) error {
	return s.writes.Hash(ctx, snap, h, write.Targets(cfg.Targets))
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// Stores returns the config stores the service owns, for registry assembly.
func (s *Service) Stores() []config.Store {
	return []config.Store{s.Read, s.Write}
}
