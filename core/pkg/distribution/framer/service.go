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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
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

// Service is the distribution layer interface for reading, writing, and streaming
// telemetry frames through Synnax. The service provides access to three APIs: writers
// for writing data, iterators for reading historical data, and streamers for consuming
// real-time data.
//
// To create a new service, call Open with a valid ServiceConfig. The framer service
// must be closed after used.
type Service struct {
	cfg             ServiceConfig
	Relay           *relay.Relay
	writer          *writer.Service
	iterator        *iterator.Service
	deleter         *deleter.Service
	controlStateKey channel.Key
}

// ServiceConfig is the configuration for the Service.
type ServiceConfig struct {
	// Instrumentation is used for logging, tracing, etc.
	// [OPTIONAL]
	alamos.Instrumentation
	// Channel is used to retrieve channel information.
	//
	// [REQUIRED]
	Channel *channel.Service
	// TS is the underlying storage time-series database for reading and writing telemetry.
	// [REQUIRED]
	TS *ts.DB
	// Transport is the network transport for moving telemetry across nodes.
	// [REQUIRED]
	Transport Transport
	// HostResolved is used to resolve address information about hosts on the network.
	// [REQUIRED]
	HostResolver cluster.HostResolver
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening the framer service.
	// This configuration is not valid on its own and must be overridden by a
	// user-provided configuration. See Config for more information.
	DefaultServiceConfig = ServiceConfig{}
)

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.framer")
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "ts", c.TS)
	validate.NotNil(v, "transport", c.Transport)
	validate.NotNil(v, "host_resolver", c.HostResolver)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.TS = override.Nil(c.TS, other.TS)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	return c
}

const freeWritePipelineBuffer = 4000

// OpenService opens a new service using the provided configuration(s). Fields defined
// in each subsequent configuration override those in previous configurations. See the
// Config struct for information required fields.
//
// Returns the Service and a nil error if opened successfully, and a nil Service and
// non-nil error if the configuration is invalid or another error occurs.
//
// The Service must be closed after use.
func OpenService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	s.iterator, err = iterator.NewService(iterator.ServiceConfig{
		TS:              cfg.TS,
		HostResolver:    cfg.HostResolver,
		Transport:       cfg.Transport.Iterator(),
		Channel:         cfg.Channel,
		Instrumentation: cfg.Child("writer"),
	})
	if err != nil {
		return nil, err
	}
	freeWrites := confluence.NewStream[relay.Response](freeWritePipelineBuffer)
	s.Relay, err = relay.Open(relay.Config{
		Instrumentation: cfg.Child("relay"),
		Channel:         cfg.Channel,
		TS:              cfg.TS,
		HostResolver:    cfg.HostResolver,
		Transport:       cfg.Transport.Relay(),
		FreeWrites:      freeWrites,
	})
	if err != nil {
		return nil, err
	}
	s.writer, err = writer.NewService(writer.ServiceConfig{
		TS:              cfg.TS,
		HostResolver:    cfg.HostResolver,
		Transport:       cfg.Transport.Writer(),
		Channel:         cfg.Channel,
		Instrumentation: cfg.Child("writer"),
		FreeWrites:      freeWrites,
	})
	if err != nil {
		return nil, err
	}
	s.deleter, err = deleter.NewService(deleter.ServiceConfig{
		HostResolver: cfg.HostResolver,
		Channel:      cfg.Channel,
		TSChannel:    cfg.TS,
		Transport:    cfg.Transport.Deleter(),
	})
	return s, err
}

// OpenIterator opens a new iterator for reading historical data from a Synnax cluster.
// If the returned error is nil, the iterator must be closed after use. For
// information on configuration parameters, see the IteratorConfig struct.
//
// The returned iterator uses a synchronous, method-based model. For a channel-based
// iterator model, use NewStreamIterator.
func (s *Service) OpenIterator(ctx context.Context, cfg IteratorConfig) (*Iterator, error) {
	return s.iterator.Open(ctx, cfg)
}

// NewStreamIterator returns an iterator for reading historical data from a Synnax cluster.
// The returned StreamIterator is a confluence.Segment that uses a channel-based interface,
// where requests are sent through an input stream, and responses are received through
// an output stream.
func (s *Service) NewStreamIterator(ctx context.Context, cfg IteratorConfig) (StreamIterator, error) {
	return s.iterator.NewStream(ctx, cfg)
}

// OpenWriter opens a new writer for writing data to a Synnax cluster. If the returned
// error is nil, the writer must be closed after use. For information on configuration
// parameters, see the WriterConfig struct.
//
// The returned writer uses a synchronous, method-based model. For a channel-based
// writer model, use NewStreamWriter.
func (s *Service) OpenWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	return s.writer.Open(ctx, cfg)
}

// NewStreamWriter returns a writer for writing data to a Synnax cluster. The returned
// writer is a confluence.Segment that uses a channel-based interface, where requests
// are sent through an input stream, and responses are received through an output stream.
func (s *Service) NewStreamWriter(ctx context.Context, cfg WriterConfig) (StreamWriter, error) {
	return s.writer.NewStream(ctx, cfg)
}

// NewDeleter opens a new deleter for deleting data from a Synnax cluster.
func (s *Service) NewDeleter() Deleter { return s.deleter.New() }

// ConfigureControlUpdateChannel sets the name and key of the channel used to propagate
// control transfers between opened writers.
func (s *Service) ConfigureControlUpdateChannel(ctx context.Context, ch channel.Key, name string) error {
	s.controlStateKey = ch
	return s.cfg.TS.ConfigureControlUpdateChannel(ctx, ts.ChannelKey(ch), name)
}

// Close closes the Service.
func (s *Service) Close() error { return s.Relay.Close() }
