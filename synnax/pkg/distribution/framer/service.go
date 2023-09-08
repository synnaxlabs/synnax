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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Service struct {
	writer   *writer.Service
	iterator *iterator.Service
	relay    *relay.Relay
}

type Config struct {
	alamos.Instrumentation
	ChannelReader channel.Readable
	TS            *ts.DB
	Transport     Transport
	HostResolver  core.HostResolver
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

func (c Config) Validate() error {
	v := validate.New("distribution.framer")
	validate.NotNil(v, "ChannelReader", c.ChannelReader)
	validate.NotNil(v, "TS", c.TS)
	validate.NotNil(v, "Transport", c.Transport)
	validate.NotNil(v, "HostResolver", c.HostResolver)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.ChannelReader = override.Nil(c.ChannelReader, other.ChannelReader)
	c.TS = override.Nil(c.TS, other.TS)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	return c
}

func Open(configs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{}
	s.iterator, err = iterator.OpenService(iterator.ServiceConfig{
		TS:              cfg.TS,
		HostResolver:    cfg.HostResolver,
		Transport:       cfg.Transport.Iterator(),
		ChannelReader:   cfg.ChannelReader,
		Instrumentation: cfg.Instrumentation,
	})
	if err != nil {
		return nil, err
	}
	s.writer, err = writer.OpenService(writer.ServiceConfig{
		TS:              cfg.TS,
		HostResolver:    cfg.HostResolver,
		Transport:       cfg.Transport.Writer(),
		ChannelReader:   cfg.ChannelReader,
		Instrumentation: cfg.Instrumentation,
	})
	s.relay, err = relay.Open(relay.Config{
		Instrumentation: cfg.Instrumentation,
		TS:              cfg.TS,
		HostResolver:    cfg.HostResolver,
		Transport:       cfg.Transport.Relay(),
	})
	return s, err
}

func (s *Service) OpenIterator(ctx context.Context, cfg IteratorConfig) (*Iterator, error) {
	return s.iterator.New(ctx, cfg)
}

func (s *Service) NewStreamIterator(ctx context.Context, cfg IteratorConfig) (StreamIterator, error) {
	return s.iterator.NewStream(ctx, cfg)
}

func (s *Service) OpenWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	return s.writer.New(ctx, cfg)
}

func (s *Service) NewStreamWriter(ctx context.Context, cfg WriterConfig) (StreamWriter, error) {
	return s.writer.NewStream(ctx, cfg)
}

// Close closes the Service.
func (s *Service) Close() error { return s.relay.Close() }
