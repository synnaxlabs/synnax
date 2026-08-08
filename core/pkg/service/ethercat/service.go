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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task/common"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the EtherCAT task config service.
type ServiceConfig struct {
	// DB is the database config records are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to register the config record resource types.
	// [REQUIRED]
	Ontology *ontology.Ontology
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("ethercat.service")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	return v.Error()
}

// Service owns the stored configuration records of the EtherCAT task types.
type Service struct {
	// Read stores ethercat_read task configuration records.
	Read *common.ConfigService[ReadConfig, *ReadConfig]
	// Write stores ethercat_write task configuration records.
	Write *common.ConfigService[WriteConfig, *WriteConfig]
	// Scan stores ethercat_scan task configuration records.
	Scan   *common.ConfigService[ScanConfig, *ScanConfig]
	closer xio.MultiCloser
}

// OpenService opens the EtherCAT task config service with the provided configuration.
// If error is nil, the service is ready for use and must be closed by calling Close
// to prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	base := common.ConfigServiceConfig{
		DB:              cfg.DB,
		Ontology:        cfg.Ontology,
		Instrumentation: cfg.Instrumentation,
	}
	if s.Read, err = common.OpenConfigService[ReadConfig](
		ctx, base, common.ConfigServiceConfig{Type: ontology.ResourceTypeEthercatRead},
	); !ok(err, s.Read) {
		return nil, err
	}
	if s.Write, err = common.OpenConfigService[WriteConfig](
		ctx, base, common.ConfigServiceConfig{Type: ontology.ResourceTypeEthercatWrite},
	); !ok(err, s.Write) {
		return nil, err
	}
	if s.Scan, err = common.OpenConfigService[ScanConfig](
		ctx, base, common.ConfigServiceConfig{Type: ontology.ResourceTypeEthercatScan},
	); !ok(err, s.Scan) {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// Stores returns the config stores the service owns, for registry assembly.
func (s *Service) Stores() []common.ConfigStore {
	return []common.ConfigStore{s.Read, s.Write, s.Scan}
}
