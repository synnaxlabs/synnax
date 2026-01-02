// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the kv.Service.
type ServiceConfig struct {
	alamos.Instrumentation
	// DB is the underlying database.
	DB *gorp.DB
	// Signals is used to publish signals when key-value pairs are created or deleted.
	Signals *signals.Provider
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultConfig is the default configuration for opening a kv service.
	DefaultConfig = ServiceConfig{}
)

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("service.ranger.kv")
	validate.NotNil(v, "db", c.DB)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Service is the main entry point for managing key-value pairs on ranges.
type Service struct {
	cfg             ServiceConfig
	shutdownSignals io.Closer
}

// OpenService opens a new kv.Service with the provided configuration.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if cfg.Signals == nil {
		return s, nil
	}
	signalsCfg := signals.GorpPublisherConfigString[Pair](cfg.DB)
	signalsCfg.SetName = "sy_range_kv_set"
	signalsCfg.DeleteName = "sy_range_kv_delete"
	kvSignals, err := signals.PublishFromGorp(ctx, cfg.Signals, signalsCfg)
	if err != nil {
		return nil, err
	}
	s.shutdownSignals = kvSignals
	return s, nil
}

// Close closes the service and releases any resources.
func (s *Service) Close() error {
	if s.shutdownSignals != nil {
		return s.shutdownSignals.Close()
	}
	return nil
}

// NewWriter opens a new Writer to create and delete key-value pairs.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: gorp.OverrideTx(s.cfg.DB, tx)}
}

// NewReader opens a new Reader to retrieve key-value pairs.
func (s *Service) NewReader(tx gorp.Tx) Reader {
	return Reader{tx: gorp.OverrideTx(s.cfg.DB, tx)}
}
