// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ethercat

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/legacy"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the EtherCAT task config service.
type ServiceConfig struct {
	// DB is the database config records are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Device reads the slave devices whose discovered PDO entries address channels.
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
	v := validate.New("ethercat.service")
	v.NotNil("db", c.DB)
	v.NotNil("device", c.Device)
	return v.Error()
}

// Service owns the stored configuration records of the EtherCAT task types.
type Service struct {
	// Read stores ethercat_read task configuration records.
	Read *config.Service[ReadConfig]
	// Write stores ethercat_write task configuration records.
	Write *config.Service[WriteConfig]
	// Scan stores ethercat_scan task configuration records.
	Scan   *config.Service[ScanConfig]
	closer xio.MultiCloser
}

// OpenService opens the EtherCAT task config service with the provided configuration.
// If error is nil, the service is ready for use and must be closed by calling Close
// to prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := xconfig.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	devices := slaves(cfg.Device)
	s = &Service{}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.Read, err = config.OpenService(
		ctx, config.ServiceConfig[ReadConfig]{
			DB:                 cfg.DB,
			Instrumentation:    cfg.Instrumentation,
			Type:               "ethercat_read",
			LegacyVersion:      legacy.LastVersion,
			Ladder:             versions.ReadConfigLadder,
			Legacy:             &legacy.Read,
			Migrations:         []migrate.Migration{v2.NewReadMigration(devices)},
			SetEntryKey:        (*ReadConfig).SetKey,
			ApplyEntryDefaults: (*ReadConfig).ApplyDefaults,
			ResolveEntry: func(ctx context.Context, tx gorp.Tx, c *ReadConfig) error {
				return v2.ResolveReadConfig(ctx, tx, devices, c)
			},
		},
	); !ok(err, s.Read) {
		return nil, err
	}
	if s.Write, err = config.OpenService(
		ctx, config.ServiceConfig[WriteConfig]{
			DB:                 cfg.DB,
			Instrumentation:    cfg.Instrumentation,
			Type:               "ethercat_write",
			LegacyVersion:      legacy.LastVersion,
			Ladder:             versions.WriteConfigLadder,
			Legacy:             &legacy.Write,
			Migrations:         []migrate.Migration{v2.NewWriteMigration(devices)},
			SetEntryKey:        (*WriteConfig).SetKey,
			ApplyEntryDefaults: (*WriteConfig).ApplyDefaults,
			ResolveEntry: func(ctx context.Context, tx gorp.Tx, c *WriteConfig) error {
				return v2.ResolveWriteConfig(ctx, tx, devices, c)
			},
		},
	); !ok(err, s.Write) {
		return nil, err
	}
	if s.Scan, err = config.OpenService(
		ctx, config.ServiceConfig[ScanConfig]{
			DB:                 cfg.DB,
			Instrumentation:    cfg.Instrumentation,
			Type:               "ethercat_scan",
			LegacyVersion:      legacy.LastVersion,
			Ladder:             versions.ScanConfigLadder,
			Legacy:             &legacy.Scan,
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

func slaves(svc *device.Service) v2.Devices {
	return func(
		ctx context.Context,
		tx gorp.Tx,
		keys []string,
	) (map[string]msgpack.EncodedJSON, error) {
		var devices []device.Device
		if err := svc.NewRetrieve().
			Where(device.And(device.MatchKeys(keys...), device.MatchMakes("ethercat"))).
			Entries(&devices).
			Exec(ctx, tx); err != nil {
			return nil, err
		}
		return lo.SliceToMap(
			devices,
			func(d device.Device) (string, msgpack.EncodedJSON) {
				return d.Key, d.Properties
			},
		), nil
	}
}
