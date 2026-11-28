// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package labjack

import (
	"context"
	"strconv"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/labjack/versions"
	"github.com/synnaxlabs/synnax/pkg/service/labjack/versions/legacy"
	v3 "github.com/synnaxlabs/synnax/pkg/service/labjack/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the LabJack task config service.
type ServiceConfig struct {
	// DB is the database config records are stored in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Device reads the device of a read task, whose model sets the ranges its analog
	// channels can take.
	//
	// [REQUIRED]
	Device *device.Service
	alamos.Instrumentation
}

var _ xconfig.Config[ServiceConfig] = ServiceConfig{}

// Override implements xconfig.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Device = override.Nil(c.Device, other.Device)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("labjack.service")
	v.NotNil("db", c.DB)
	v.NotNil("device", c.Device)
	return v.Error()
}

// Service owns the stored configuration records of the LabJack task types.
type Service struct {
	// Read stores labjack_read task configuration records.
	Read *config.Service[ReadConfig]
	// Write stores labjack_write task configuration records.
	Write *config.Service[WriteConfig]
	// Scan stores labjack_scan task configuration records.
	Scan   *config.Service[ScanConfig]
	closer xio.MultiCloser
}

// OpenService opens the LabJack task config service with the provided configuration.
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
			Type:            "labjack_read",
			LegacyVersion:   legacy.LastVersion,
			Ladder:          versions.ReadConfigLadder,
			Legacy:          &legacy.Read,
			Migrations: []gorp.Migration{
				v3.ReadMigration,
				v3.ReadReconcileMigration,
				readModelRangeMigration(cfg.Device),
			},
			SetEntryKey: (*ReadConfig).SetKey,
			PatchEntry:  (*ReadConfig).Patch,
			HashEntry:   (*ReadConfig).HashContent,
			ResolveEntry: func(
				ctx context.Context,
				snap gorp.Snapshot,
				_ string,
				c *ReadConfig,
			) (func(context.Context, gorp.Tx) error, error) {
				return nil, validateRanges(ctx, snap, cfg.Device, *c)
			},
			ReconcileEntry: func(ctx context.Context, tx gorp.Tx, c *ReadConfig) error {
				v3.ReconcileReadChannels(c.Channels)
				_, err := reconcileRanges(ctx, tx, cfg.Device, c)
				return err
			},
			ValidateEntry: func(c *ReadConfig) error {
				v := validate.New("labjack.ReadConfig")
				v.Exec(c.Validate)
				v.GreaterThan("stream_rate", c.StreamRate, 0)
				for i, ch := range c.Channels {
					v.Exec(func() error {
						return validate.PathedError(
							validateNegChan(ch),
							"channels", strconv.Itoa(i), "neg_chan",
						)
					})
				}
				return v.Error()
			},
		},
	); !ok(err, s.Read) {
		return nil, err
	}
	if s.Write, err = config.OpenService(
		ctx, config.ServiceConfig[WriteConfig]{
			DB:              cfg.DB,
			References:      writeConfigReferences(),
			Instrumentation: cfg.Instrumentation,
			Type:            "labjack_write",
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
			Type:            "labjack_scan",
			LegacyVersion:   legacy.LastVersion,
			Ladder:          versions.ScanConfigLadder,
			Legacy:          &legacy.Scan,
			Migrations:      []gorp.Migration{v3.ScanMigration},
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

// validateNegChan rejects a negative channel that is not the pair of its channel's
// port.
func validateNegChan(ch ReadChannel) error {
	var (
		port string
		neg  *uint16
	)
	switch c := ch.Variant.(type) {
	case AnalogReadChannel:
		port, neg = c.Port, c.NegChan
	case ThermocoupleReadChannel:
		port, neg = c.Port, c.NegChan
	}
	if neg == nil {
		return nil
	}
	pair, ok := v3.PairOf(port)
	if !ok {
		return validate.Errorf("%s has no differential pair", port)
	}
	if *neg != pair {
		return validate.Errorf("%s pairs only with AIN%d", port, pair)
	}
	return nil
}
