// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package framer provides the service-level types for writing, reading, and streaming
// telemetry data through Synnax. This extends the distribution-layer framer service
// with calculated channel functionality, throttling, and other features.
package framer

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/framer/streamer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type (
	Frame            = framer.Frame
	Iterator         = iterator.Iterator
	IteratorCommand  = iterator.Command
	IteratorRequest  = iterator.Request
	IteratorResponse = iterator.Response
	StreamIterator   = iterator.StreamIterator
	Writer           = framer.Writer
	WriterCommand    = framer.WriterCommand
	WriterConfig     = framer.WriterConfig
	WriterMode       = framer.WriterMode
	WriterRequest    = framer.WriterRequest
	WriterResponse   = framer.WriterResponse
	StreamWriter     = framer.StreamWriter
	IteratorConfig   = iterator.Config
	StreamerConfig   = streamer.Config
	StreamerRequest  = streamer.Request
	StreamerResponse = streamer.Response
	Streamer         = streamer.Streamer
)

const (
	IteratorResponseVariantAck  = iterator.ResponseVariantAck
	IteratorResponseVariantData = iterator.ResponseVariantData
	WriterCommandOpen           = framer.WriterCommandOpen
	WriterCommandWrite          = framer.WriterCommandWrite
	WriterCommandCommit         = framer.WriterCommandCommit
	WriterCommandSetAuthority   = framer.WriterCommandSetAuthority
)

// Frame constructors re-exported from the distribution-layer frame package so callers
// can build frames without importing it directly.
var (
	NewUnary       = frame.NewUnary
	NewMulti       = frame.NewMulti
	NewFromStorage = frame.NewFromStorage
)

type ServiceConfig struct {
	//  Distribution layer framer service.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Channel is used to retrieve channel information.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Status is used for persisting calculation status updates.
	//
	// [REQUIRED]
	Status *status.Service
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("framer")
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "status", c.Status)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

type Service struct {
	closer   io.MultiCloser
	streamer *streamer.Service
	iterator *iterator.Service
	cfg      ServiceConfig
}

func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	var calcSvc *calculation.Service
	if calcSvc, err = calculation.OpenService(ctx, calculation.ServiceConfig{
		Instrumentation: cfg.Child("calculation"),
		Channel:         cfg.Channel,
		Framer:          cfg.Framer,
		Status:          cfg.Status,
	}); !ok(err, calcSvc) {
		return nil, err
	}
	if s.streamer, err = streamer.NewService(streamer.ServiceConfig{
		Instrumentation: cfg.Child("streamer"),
		Framer:          cfg.Framer,
		Channel:         cfg.Channel,
		Calculation:     calcSvc,
	}); !ok(err, nil) {
		return nil, err
	}
	if s.iterator, err = iterator.NewService(iterator.ServiceConfig{
		Instrumentation: cfg.Child("iterator"),
		Framer:          cfg.Framer,
		Channel:         cfg.Channel,
	}); !ok(err, nil) {
		return nil, err
	}
	return s, nil
}

func (s *Service) OpenWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	return s.cfg.Framer.OpenWriter(ctx, cfg)
}

func (s *Service) NewStreamWriter(
	ctx context.Context, cfg WriterConfig,
) (StreamWriter, error) {
	return s.cfg.Framer.NewStreamWriter(ctx, cfg)
}

func (s *Service) OpenIterator(
	ctx context.Context, cfg IteratorConfig,
) (*Iterator, error) {
	return s.iterator.Open(ctx, cfg)
}

func (s *Service) NewStreamIterator(
	ctx context.Context, cfg IteratorConfig,
) (StreamIterator, error) {
	return s.iterator.NewStream(ctx, cfg)
}

func (s *Service) NewStreamer(
	ctx context.Context,
	cfg StreamerConfig,
) (Streamer, error) {
	return s.streamer.New(ctx, cfg)
}

func (s *Service) DeleteTimeRange(
	ctx context.Context,
	keys channel.Keys,
	tr telem.TimeRange,
) error {
	return s.cfg.Framer.DeleteTimeRange(ctx, keys, tr)
}

func (s *Service) Close() error { return s.closer.Close() }
