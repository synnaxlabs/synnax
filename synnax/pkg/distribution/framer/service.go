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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Service struct {
	config          Config
	writer          *writer.Service
	iterator        *iterator.Service
	deleter         deleter.Service
	controlStateKey channel.Key
	Relay           *relay.Relay
}

type Writable interface {
	OpenWriter(ctx context.Context, cfg WriterConfig) (*Writer, error)
	NewStreamWriter(ctx context.Context, cfg WriterConfig) (StreamWriter, error)
}

type Readable interface {
	OpenIterator(ctx context.Context, cfg IteratorConfig) (*Iterator, error)
	NewStreamIterator(ctx context.Context, cfg IteratorConfig) (StreamIterator, error)
}

type Streamable interface {
	NewStreamer(ctx context.Context, cfg StreamerConfig) (Streamer, error)
}

type ReadWriteable interface {
	Writable
	Readable
}

type ReadWriteStreamable interface {
	Writable
	Readable
	Streamable
}

type WriteStreamable interface {
	Writable
	Streamable
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

// Validate implements config.Properties.
func (c Config) Validate() error {
	v := validate.New("distribution.framer")
	validate.NotNil(v, "ChannelReader", c.ChannelReader)
	validate.NotNil(v, "TS", c.TS)
	validate.NotNil(v, "Transport", c.Transport)
	validate.NotNil(v, "HostProvider", c.HostResolver)
	return v.Error()
}

// Override implements config.Properties.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
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
	s := &Service{config: cfg}
	s.iterator, err = iterator.OpenService(iterator.ServiceConfig{
		TS:              cfg.TS,
		HostResolver:    cfg.HostResolver,
		Transport:       cfg.Transport.Iterator(),
		ChannelReader:   cfg.ChannelReader,
		Instrumentation: cfg.Instrumentation.Child("writer"),
	})
	if err != nil {
		return nil, err
	}
	freeWrites := confluence.NewStream[relay.Response](25)
	s.Relay, err = relay.Open(relay.Config{
		Instrumentation: cfg.Instrumentation.Child("Relay"),
		ChannelReader:   cfg.ChannelReader,
		TS:              cfg.TS,
		HostResolver:    cfg.HostResolver,
		Transport:       cfg.Transport.Relay(),
		FreeWrites:      freeWrites,
	})
	if err != nil {
		return nil, err
	}
	s.writer, err = writer.OpenService(writer.ServiceConfig{
		TS:              cfg.TS,
		HostResolver:    cfg.HostResolver,
		Transport:       cfg.Transport.Writer(),
		ChannelReader:   cfg.ChannelReader,
		Instrumentation: cfg.Instrumentation.Child("writer"),
		FreeWrites:      freeWrites,
	})
	if err != nil {
		return nil, err
	}
	s.deleter, err = deleter.New(deleter.ServiceConfig{
		HostResolver:  cfg.HostResolver,
		ChannelReader: cfg.ChannelReader,
		TSChannel:     cfg.TS,
		Transport:     cfg.Transport.Deleter(),
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

func (s *Service) NewDeleter() Deleter {
	return s.deleter.NewDeleter()
}

func (s *Service) ConfigureControlUpdateChannel(ctx context.Context, ch channel.Key) error {
	s.controlStateKey = ch
	return s.config.TS.ConfigureControlUpdateChannel(ctx, ts.ChannelKey(ch))
}

// Close closes the Service.
func (s *Service) Close() error { return s.Relay.Close() }
