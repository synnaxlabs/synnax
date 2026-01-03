// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Config the configuration for opening an Iterator or StreamIterator.
type Config struct {
	// Keys are the keys of the channels to iterator over. At least one key must
	// be specified. An iterator cannot iterate over non-calculated virtual channels
	// or free channels, and calls to Open or NewStream will return an error when
	// attempting to iterate over channels of these types.
	// [REQUIRED] - must have at least one key.
	Keys channel.Keys `json:"keys" msgpack:"keys"`
	// Bounds sets the time range to iterate over. This time range must be valid i.e.,
	// the start value must be before or equal to the end value.
	// [REQUIRED]
	Bounds telem.TimeRange `json:"bounds" msgpack:"bounds"`
	// ChunkSize sets the default number of samples to iterate over per-channel when
	// calling Next or Prev with AutoSpan.
	ChunkSize int64 `json:"chunk_size" msgpack:"chunk_size"`
}

// ServiceConfig is the configuration for opening the iterator Service, the main
// entrypoint for using iterators.
type ServiceConfig struct {
	// Instrumentation is used for Logging, Tracing, and Metrics.
	// [OPTIONAL]
	alamos.Instrumentation
	// TS is the underlying storage layer time-series database for reading frames.
	// [REQUIRED]
	TS *ts.DB
	// Channel retrieves channel information.
	//
	// [REQUIRED}
	Channel *channel.Service
	// HostResolver is used to resolve reachable addresses for nodes in a Synnax cluster.
	// [REQUIRED]
	HostResolver aspen.HostResolver
	// Transport is the network transport for moving telemetry frames across nodes.
	// [REQUIRED]
	Transport Transport
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening a new iterator
	// service. This configuration is not valid on its own and must be overridden
	// with the required fields specified in ServiceConfig.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.Channel = override.Nil(cfg.Channel, other.Channel)
	cfg.Transport = override.Nil(cfg.Transport, other.Transport)
	cfg.HostResolver = override.Nil(cfg.HostResolver, other.HostResolver)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	return cfg
}

// Validate implements Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.iterator")
	validate.NotNil(v, "ts", cfg.TS)
	validate.NotNil(v, "channel", cfg.Channel)
	validate.NotNil(v, "aspen_transport", cfg.Transport)
	validate.NotNil(v, "resolver", cfg.HostResolver)
	return v.Error()
}

// Service is the distribution layer entry point for using iterators within Synnax.
// Iterators allow for reading chunks of historical data from channels distributed
// across a multi-node cluster.
type Service struct {
	cfg    ServiceConfig
	server *server
}

// NewService opens a new iterator service using the provided configuration. If the
// configuration is invalid, NewService returns a nil service and an error.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg, server: newServer(cfg)}, nil
}

const (
	peerSenderAddr   address.Address = "peer_sender"
	gatewayIterAddr  address.Address = "gateway_writer"
	broadcasterAddr  address.Address = "broadcaster"
	synchronizerAddr address.Address = "synchronizer"
)

// Open opens a new iterator for reading historical data from a Synnax cluster.
// If the returned error is nil, the iterator must be closed after use. For
// information on configuration parameters, see the IteratorConfig struct.
//
// The returned iterator uses a synchronous, method-based model. For a channel-based
// iterator model, use NewStream.
func (s *Service) Open(ctx context.Context, cfg Config) (*Iterator, error) {
	stream, err := s.NewStream(ctx, cfg)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.cfg.Instrumentation))
	req := confluence.NewStream[Request]()
	res := confluence.NewStream[Response]()
	stream.InFrom(req)
	stream.OutTo(res)
	stream.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.CancelOnFail(),
		confluence.RecoverWithErrOnPanic(),
	)
	return &Iterator{requests: req, responses: res, shutdown: cancel, wg: sCtx}, nil
}

// NewStream returns an iterator for reading historical data from a Synnax cluster.
// The returned StreamIterator is a confluence.Segment that uses a channel-based interface,
// where requests are sent through an input stream, and responses are received through
// an output stream.
func (s *Service) NewStream(ctx context.Context, cfg Config) (StreamIterator, error) {
	if err := s.validateChannelKeys(ctx, cfg.Keys); err != nil {
		return nil, err
	}
	cfg.Keys = cfg.Keys.Unique()
	var (
		hostID             = s.cfg.HostResolver.HostKey()
		batch              = proxy.BatchFactory[channel.Key]{Host: hostID}.Batch(cfg.Keys)
		pipe               = plumber.New()
		needPeerRouting    = len(batch.Peers) > 0
		needGatewayRouting = len(batch.Gateway) > 0
		receiverAddresses  []address.Address
		routeInletTo       address.Address
	)

	if needPeerRouting {
		routeInletTo = peerSenderAddr
		sender, receivers, err := s.openManyPeers(ctx, cfg.Bounds, cfg.ChunkSize, batch.Peers, !needGatewayRouting)
		if err != nil {
			return nil, err
		}
		plumber.SetSink[Request](pipe, peerSenderAddr, sender)
		receiverAddresses = make([]address.Address, len(receivers))
		for i, c := range receivers {
			addr := address.Newf("client_%v", i+1)
			receiverAddresses[i] = addr
			plumber.SetSource[Response](pipe, addr, c)
		}
	}

	if needGatewayRouting {
		routeInletTo = gatewayIterAddr
		gatewayIter, err := s.newGateway(
			Config{Keys: batch.Gateway, Bounds: cfg.Bounds, ChunkSize: cfg.ChunkSize},
			!needPeerRouting,
		)
		if err != nil {
			return nil, err
		}
		plumber.SetSegment[Request, Response](pipe, gatewayIterAddr, gatewayIter)
		receiverAddresses = append(receiverAddresses, gatewayIterAddr)
	}

	if needPeerRouting && needGatewayRouting {
		routeInletTo = broadcasterAddr
		plumber.SetSegment[Request, Request](pipe, broadcasterAddr, newBroadcaster())
		plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{broadcasterAddr},
			SinkTargets:   []address.Address{peerSenderAddr, gatewayIterAddr},
			Stitch:        plumber.StitchWeave,
			Capacity:      2,
		}.MustRoute(pipe)
	}

	plumber.SetSegment[Response, Response](
		pipe,
		synchronizerAddr,
		newSynchronizer(len(cfg.Keys.UniqueLeaseholders()), s.cfg.Instrumentation),
	)

	plumber.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{synchronizerAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      len(receiverAddresses),
	}.MustRoute(pipe)

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteOutletFrom(synchronizerAddr))
	lo.Must0(seg.RouteInletTo(routeInletTo))
	return seg, nil
}

func (s *Service) validateChannelKeys(ctx context.Context, keys channel.Keys) error {
	v := validate.New("distribution.framer.iterator")
	if validate.NotEmptySlice(v, "keys", keys) {
		return v.Error()
	}
	for _, k := range keys {
		if k.Free() {
			return errors.Wrapf(validate.Error, "cannot read from free channel %v", k)
		}
	}
	q := s.cfg.Channel.NewRetrieve().WhereKeys(keys...)
	exists, err := q.Exists(ctx, nil)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(query.NotFound, "some channel keys %v not found", keys)
	}
	return nil
}
