// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

const (
	// ReadTaskType is the type of a task that reads topics and Sparkplug B tags.
	ReadTaskType = "mqtt_read"
	// WriteTaskType is the type of a task that publishes to topics and commands
	// Sparkplug B tags.
	WriteTaskType = "mqtt_write"
	// ScanTaskType is the type of the internal task that checks broker devices.
	ScanTaskType = "mqtt_scan"
	// EdgeTaskType is the type of a task that is one Sparkplug B edge node.
	EdgeTaskType = "mqtt_sparkplug_edge"
	// configVersion is the version of every MQTT config type. No legacy shape sits
	// below it.
	configVersion = 1
)

// ServiceConfig is the configuration for opening the MQTT task config service.
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
	v := validate.New("mqtt.service")
	v.NotNil("db", c.DB)
	return v.Error()
}

// Service owns the stored configuration records of the MQTT task types.
type Service struct {
	// Read stores mqtt_read task configuration records.
	Read *config.Service[ReadConfig]
	// Write stores mqtt_write task configuration records.
	Write *config.Service[WriteConfig]
	// Scan stores mqtt_scan task configuration records.
	Scan *config.Service[ScanConfig]
	// Edge stores mqtt_sparkplug_edge task configuration records.
	Edge   *config.Service[EdgeConfig]
	closer xio.MultiCloser
}

// OpenService opens the MQTT task config service with the provided configuration.
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
			Type:               ReadTaskType,
			Version:            configVersion,
			SetEntryKey:        (*ReadConfig).SetKey,
			ApplyEntryDefaults: (*ReadConfig).ApplyDefaults,
			ValidateEntry:      (*ReadConfig).Validate,
		},
	); !ok(err, s.Read) {
		return nil, err
	}
	if s.Write, err = config.OpenService(
		ctx, config.ServiceConfig[WriteConfig]{
			DB:                 cfg.DB,
			Instrumentation:    cfg.Instrumentation,
			Type:               WriteTaskType,
			Version:            configVersion,
			SetEntryKey:        (*WriteConfig).SetKey,
			ApplyEntryDefaults: (*WriteConfig).ApplyDefaults,
			ValidateEntry:      (*WriteConfig).Validate,
		},
	); !ok(err, s.Write) {
		return nil, err
	}
	if s.Scan, err = config.OpenService(
		ctx, config.ServiceConfig[ScanConfig]{
			DB:                 cfg.DB,
			Instrumentation:    cfg.Instrumentation,
			Type:               ScanTaskType,
			Version:            configVersion,
			SetEntryKey:        (*ScanConfig).SetKey,
			ApplyEntryDefaults: (*ScanConfig).ApplyDefaults,
		},
	); !ok(err, s.Scan) {
		return nil, err
	}
	if s.Edge, err = config.OpenService(
		ctx, config.ServiceConfig[EdgeConfig]{
			DB:                 cfg.DB,
			Instrumentation:    cfg.Instrumentation,
			Type:               EdgeTaskType,
			Version:            configVersion,
			SetEntryKey:        (*EdgeConfig).SetKey,
			ApplyEntryDefaults: (*EdgeConfig).ApplyDefaults,
			ValidateEntry:      (*EdgeConfig).Validate,
		},
	); !ok(err, s.Edge) {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// Stores returns the config stores the service owns, for registry assembly.
func (s *Service) Stores() []config.Store {
	return []config.Store{s.Read, s.Write, s.Scan, s.Edge}
}
