// Copyright 2025 Synnax Labs, Inc.
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
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/framer/streamer"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type (
	Frame            = core.Frame
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
	Deleter          = deleter.Deleter
)

type Config struct {
	alamos.Instrumentation
	//  Distribution layer framer service.
	Framer                   *framer.Service
	Channel                  channel.Service
	Arc                      *arc.Service
	EnableLegacyCalculations *bool
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{
		EnableLegacyCalculations: config.False(),
	}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("framer")
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "arc", c.Arc)
	validate.NotNil(v, "enable_legacy_calculations", c.EnableLegacyCalculations)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Arc = override.Nil(c.Arc, other.Arc)
	c.EnableLegacyCalculations = override.Nil(c.EnableLegacyCalculations, other.EnableLegacyCalculations)
	return c
}

type Service struct {
	Config
	Streamer *streamer.Service
	Iterator *iterator.Service
	closer   io.Closer
}

func (s *Service) OpenIterator(ctx context.Context, cfg framer.IteratorConfig) (*Iterator, error) {
	return s.Iterator.Open(ctx, cfg)
}

func (s *Service) NewStreamIterator(ctx context.Context, cfg framer.IteratorConfig) (StreamIterator, error) {
	return s.Iterator.NewStream(ctx, cfg)
}

func (s *Service) NewStreamWriter(ctx context.Context, cfg framer.WriterConfig) (StreamWriter, error) {
	return s.Framer.NewStreamWriter(ctx, cfg)
}

func (s *Service) OpenWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	return s.Framer.OpenWriter(ctx, cfg)
}

func (s *Service) NewDeleter() framer.Deleter {
	return s.Framer.NewDeleter()
}

func (s *Service) NewStreamer(ctx context.Context, cfg StreamerConfig) (Streamer, error) {
	return s.Streamer.New(ctx, cfg)
}

func (s *Service) Close() error {
	return s.closer.Close()
}

func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	calcSvc, err := calculation.OpenService(ctx, calculation.ServiceConfig{
		Instrumentation:          cfg.Child("calculated"),
		Channel:                  cfg.Channel,
		Framer:                   cfg.Framer,
		Arc:                      cfg.Arc,
		ChannelObservable:        cfg.Channel.NewObservable(),
		EnableLegacyCalculations: cfg.EnableLegacyCalculations,
	})
	if err != nil {
		return nil, err
	}
	streamerSvc, err := streamer.NewService(streamer.ServiceConfig{
		Instrumentation: cfg.Child("streamer"),
		DistFramer:      cfg.Framer,
		Channel:         cfg.Channel,
		Calculation:     calcSvc,
	})
	if err != nil {
		return nil, err
	}
	iteratorSvc, err := iterator.NewService(iterator.ServiceConfig{
		DistFramer: cfg.Framer,
		Channel:    cfg.Channel,
		Arc:        cfg.Arc,
	})
	if err != nil {
		return nil, err
	}
	return &Service{
		Config:   cfg,
		Streamer: streamerSvc,
		Iterator: iteratorSvc,
		closer:   xio.MultiCloser{calcSvc},
	}, nil
}
