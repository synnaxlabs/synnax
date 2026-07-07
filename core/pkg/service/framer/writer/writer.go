// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package writer is the service-layer entry point for writing telemetry to a Synnax
// cluster. It is a thin shim over the distribution-layer writer: before delegating, it
// ensures any free channels in the Config's Keys are registered in this node's local
// storage engine, where free writes are serviced.
package writer

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type (
	Config       = writer.Config
	Writer       = writer.Writer
	StreamWriter = writer.StreamWriter
	Request      = writer.Request
	Response     = writer.Response
	Command      = writer.Command
	Mode         = writer.Mode
)

const (
	CommandOpen         = writer.CommandOpen
	CommandWrite        = writer.CommandWrite
	CommandCommit       = writer.CommandCommit
	CommandSetAuthority = writer.CommandSetAuthority
)

// ServiceConfig is the configuration for opening the service-layer writer Service.
type ServiceConfig struct {
	// Framer is the distribution-layer frame service the writer delegates to.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Channel provisions free channels in local storage before writers open against
	// them.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.Framer = override.Nil(cfg.Framer, other.Framer)
	cfg.Channel = override.Nil(cfg.Channel, other.Channel)
	return cfg
}

// Validate implements config.Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("writer")
	validate.NotNil(v, "framer", cfg.Framer)
	validate.NotNil(v, "channel", cfg.Channel)
	return v.Error()
}

// Service is the service-layer entry point for opening writers against a Synnax
// cluster.
type Service struct{ cfg ServiceConfig }

// NewService creates a new service using the provided configuration(s). Each subsequent
// configuration overrides the previous one. If the resulting configuration is invalid,
// NewService returns a nil service and a non-nil error.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// NewStream opens a StreamWriter for the channels in cfg. It returns validate.Error if
// Keys is empty and query.ErrNotFound if any key does not reference an existing
// channel.
func (s *Service) NewStream(ctx context.Context, cfg Config) (StreamWriter, error) {
	if err := s.cfg.Channel.EnsureFreeStorage(ctx, cfg.Keys); err != nil {
		return nil, err
	}
	return s.cfg.Framer.NewStreamWriter(ctx, cfg)
}

// Open opens a Writer for the channels in cfg. It returns validate.Error if Keys is
// empty and query.ErrNotFound if any key does not reference an existing channel.
func (s *Service) Open(ctx context.Context, cfg Config) (*Writer, error) {
	if err := s.cfg.Channel.EnsureFreeStorage(ctx, cfg.Keys); err != nil {
		return nil, err
	}
	return s.cfg.Framer.OpenWriter(ctx, cfg)
}
