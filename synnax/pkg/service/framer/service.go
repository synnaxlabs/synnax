// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/computron"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculated"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type Config struct {
	alamos.Instrumentation
	// Distribution layer framer service.
	Framer  *framer.Service
	Channel channel.Readable
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("framer")
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	return c
}

type Service struct {
	Config
	Calculated *calculated.Service
}

func (s *Service) OpenIterator(ctx context.Context, cfg framer.IteratorConfig) (*framer.Iterator, error) {
	return s.Framer.OpenIterator(ctx, cfg)
}

func (s *Service) NewStreamIterator(ctx context.Context, cfg framer.IteratorConfig) (framer.StreamIterator, error) {
	return s.Framer.NewStreamIterator(ctx, cfg)
}

func (s *Service) NewStreamWriter(ctx context.Context, cfg framer.WriterConfig) (framer.StreamWriter, error) {
	return s.Framer.NewStreamWriter(ctx, cfg)
}

func (s *Service) NewDeleter() framer.Deleter {
	return s.Framer.NewDeleter()
}

func (s *Service) NewStreamer(ctx context.Context, cfg framer.StreamerConfig) (framer.Streamer, error) {
	ut := &updaterTransform{
		Instrumentation: s.Instrumentation,
		c:               s.Calculated,
		readable:        s.Channel,
	}
	ut.Transform = ut.transform
	if err := ut.update(ctx, cfg.Keys); err != nil {
		return nil, err
	}
	p := plumber.New()
	streamer, err := s.Framer.NewStreamer(ctx, cfg)
	if err != nil {
		return nil, err
	}
	plumber.SetSegment(p, "streamer", streamer)
	plumber.SetSegment[framer.StreamerRequest, framer.StreamerRequest](p, "updater", ut)
	plumber.MustConnect[framer.StreamerRequest](p, "updater", "streamer", 10)
	seg := &plumber.Segment[framer.StreamerRequest, framer.StreamerResponse]{
		Pipeline:         p,
		RouteInletsTo:    []address.Address{"updater"},
		RouteOutletsFrom: []address.Address{"streamer"},
	}
	return seg, nil
}

func (s *Service) Close() error {
	return s.Calculated.Close()
}

type updaterTransform struct {
	alamos.Instrumentation
	c        *calculated.Service
	readable channel.Readable
	closer   xio.MultiCloser
	confluence.LinearTransform[framer.StreamerRequest, framer.StreamerRequest]
}

var _ confluence.Segment[framer.StreamerRequest, framer.StreamerRequest] = &updaterTransform{}

func (t *updaterTransform) update(ctx context.Context, keys []channel.Key) error {
	if err := t.closer.Close(); err != nil {
		return err
	}
	var channels []channel.Channel
	if err := t.readable.NewRetrieve().WhereKeys(keys...).Entries(&channels).Exec(ctx, nil); err != nil {
		return err
	}
	for _, ch := range channels {
		if ch.IsCalculated() {
			closer, err := t.c.Request(ctx, ch.Key())
			if err != nil {
				return err
			}
			t.closer = append(t.closer, closer)
		}
	}
	return nil
}

func (t *updaterTransform) transform(ctx context.Context, req framer.StreamerRequest) (framer.StreamerRequest, bool, error) {
	if err := t.update(ctx, req.Keys); err != nil {
		t.L.Error("failed to update calculated channels", zap.Error(err))
	}
	return req, true, nil
}

func (t *updaterTransform) Flow(ctx signal.Context, opts ...confluence.Option) {
	t.LinearTransform.Flow(ctx, append(opts, confluence.Defer(func() {
		if err := t.closer.Close(); err != nil {
			t.L.Error("failed to close calculated channels", zap.Error(err))
		}
	}))...)
}

func OpenService(cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg}
	computer, err := computron.New(computron.Config{Instrumentation: cfg.Child("computron")})
	if err != nil {
		return nil, err
	}
	calc, err := calculated.Open(calculated.Config{
		Instrumentation: cfg.Instrumentation.Child("calculated"),
		Computron:       computer,
		Channel:         cfg.Channel,
		Framer:          cfg.Framer,
	})
	s.Calculated = calc
	return s, err
}
