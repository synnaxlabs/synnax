// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/framer/streamer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type (
	Frame            = frame.Frame
	Iterator         = iterator.Iterator
	IteratorRequest  = iterator.Request
	IteratorResponse = iterator.Response
	StreamIterator   = iterator.StreamIterator
	Writer           = writer.Writer
	WriterRequest    = writer.Request
	WriterResponse   = writer.Response
	StreamWriter     = writer.StreamWriter
	WriterConfig     = writer.Config
	IteratorConfig   = iterator.Config
	StreamerConfig   = streamer.Config
	StreamerRequest  = streamer.Request
	StreamerResponse = streamer.Response
	Streamer         = streamer.Streamer
)

type ServiceConfig struct {
	// DB is the underlying database used by the calculation service.
	// [REQUIRED]
	DB *gorp.DB
	// Framer is the distribution-layer framer service this service extends.
	// [REQUIRED]
	Framer *framer.Service
	// Channel is used to resolve channel metadata and to create the node's control
	// update channel.
	// [REQUIRED]
	Channel *channel.Service
	// Status is used for persisting calculation status updates.
	// [REQUIRED]
	Status *status.Service
	// HostResolver identifies the host node, used to name and lease the node's control
	// update channel.
	// [REQUIRED]
	HostResolver node.HostResolver
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("framer")
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "host_resolver", c.HostResolver)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.DB = override.Nil(c.DB, other.DB)
	c.Status = override.Nil(c.Status, other.Status)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	return c
}

type Service struct {
	closer   io.MultiCloser
	Streamer *streamer.Service
	Iterator *iterator.Service
	Writer   *writer.Service
	cfg      ServiceConfig
}

func (s *Service) OpenIterator(
	ctx context.Context, cfg IteratorConfig,
) (*Iterator, error) {
	return s.Iterator.Open(ctx, cfg)
}

func (s *Service) NewStreamIterator(
	ctx context.Context, cfg IteratorConfig,
) (StreamIterator, error) {
	return s.Iterator.NewStream(ctx, cfg)
}

func (s *Service) NewStreamWriter(
	ctx context.Context, cfg WriterConfig,
) (StreamWriter, error) {
	return s.Writer.NewStream(ctx, cfg)
}

func (s *Service) OpenWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	return s.Writer.Open(ctx, cfg)
}

func (s *Service) DeleteTimeRange(
	ctx context.Context,
	keys channel.Keys,
	tr telem.TimeRange,
) error {
	return s.cfg.Framer.DeleteTimeRange(ctx, keys, tr)
}

func (s *Service) NewStreamer(
	ctx context.Context,
	cfg StreamerConfig,
) (Streamer, error) {
	return s.Streamer.New(ctx, cfg)
}

func (s *Service) Close() error { return s.closer.Close() }

// OpenService opens a framer Service from the provided configuration. All fields are
// required. It wires up the writer, calculation-backed streaming and iteration, and
// configures the host node's control update channel.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	if s.Writer, err = writer.NewService(writer.ServiceConfig{
		Instrumentation: cfg.Child("writer"),
		Framer:          cfg.Framer,
		Channel:         cfg.Channel,
	}); err != nil {
		return nil, err
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	var calcSvc *calculation.Service
	if calcSvc, err = calculation.OpenService(ctx, calculation.ServiceConfig{
		Instrumentation: cfg.Child("calculation"),
		DB:              cfg.DB,
		Channel:         cfg.Channel,
		Framer:          cfg.Framer,
		Writer:          s.Writer,
		Status:          cfg.Status,
	}); !ok(err, calcSvc) {
		return nil, err
	}
	if s.Streamer, err = streamer.NewService(streamer.ServiceConfig{
		Instrumentation: cfg.Child("streamer"),
		DistFramer:      cfg.Framer,
		Channel:         cfg.Channel,
		Calculation:     calcSvc,
	}); !ok(err, nil) {
		return nil, err
	}
	if s.Iterator, err = iterator.NewService(iterator.ServiceConfig{
		Instrumentation: cfg.Child("iterator"),
		DistFramer:      cfg.Framer,
		Channel:         cfg.Channel,
	}); !ok(err, nil) {
		return nil, err
	}
	if err = s.configureControlUpdates(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// configureControlUpdates creates the host node's control update channel (if it does not
// already exist) and registers it with the distribution framer so control state changes
// are streamed to clients.
func (s *Service) configureControlUpdates(ctx context.Context) error {
	name := fmt.Sprintf("sy_node_%v_control", s.cfg.HostResolver.HostKey())
	controlCh := channel.Channel{
		Name:        name,
		Leaseholder: s.cfg.HostResolver.HostKey(),
		Virtual:     true,
		DataType:    telem.StringT,
		Internal:    true,
	}
	if err := s.cfg.Channel.Create(ctx, &controlCh, channel.RetrieveIfNameExists()); err != nil {
		return err
	}
	return s.cfg.Framer.ConfigureControlUpdateChannel(ctx, controlCh.Key(), name)
}
