// Copyright 2023 Synnax Labs, Inc.
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

type Config struct {
	Keys      channel.Keys    `json:"keys" msgpack:"keys"`
	Bounds    telem.TimeRange `json:"bounds" msgpack:"bounds"`
	ChunkSize int64           `json:"chunk_size" msgpack:"chunk_size"`
}

type ServiceConfig struct {
	alamos.Instrumentation
	TS            *ts.DB
	ChannelReader channel.Readable
	HostResolver  aspen.HostResolver
	Transport     Transport
}

var (
	_             config.Config[ServiceConfig] = ServiceConfig{}
	DefaultConfig                              = ServiceConfig{}
)

// Override implements Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.ChannelReader = override.Nil(cfg.ChannelReader, other.ChannelReader)
	cfg.Transport = override.Nil(cfg.Transport, other.Transport)
	cfg.HostResolver = override.Nil(cfg.HostResolver, other.HostResolver)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	return cfg
}

// Validate implements Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.Iterator")
	validate.NotNil(v, "TS", cfg.TS)
	validate.NotNil(v, "ChannelReader", cfg.ChannelReader)
	validate.NotNil(v, "Transport", cfg.Transport)
	validate.NotNil(v, "Resolver", cfg.HostResolver)
	return v.Error()
}

type Service struct {
	ServiceConfig
	server *server
}

func OpenService(configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	return &Service{
		ServiceConfig: cfg,
		server:        startServer(cfg),
	}, err
}

const (
	peerSenderAddr   address.Address = "peerSender"
	gatewayIterAddr  address.Address = "gatewayWriter"
	broadcasterAddr  address.Address = "broadcaster"
	synchronizerAddr address.Address = "synchronizer"
)

func (s *Service) New(ctx context.Context, cfg Config) (*Iterator, error) {
	stream, err := s.NewStream(ctx, cfg)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.Instrumentation))
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

func (s *Service) NewStream(ctx context.Context, cfg Config) (StreamIterator, error) {
	if err := s.validateChannelKeys(ctx, cfg.Keys); err != nil {
		return nil, err
	}
	cfg.Keys = cfg.Keys.Unique()

	var (
		hostID             = s.HostResolver.HostKey()
		batch              = proxy.BatchFactory[channel.Key]{Host: hostID}.Batch(cfg.Keys)
		pipe               = plumber.New()
		needPeerRouting    = len(batch.Peers) > 0
		needGatewayRouting = len(batch.Gateway) > 0
		receiverAddresses  []address.Address
		routeInletTo       address.Address
	)

	if needPeerRouting {
		routeInletTo = peerSenderAddr
		sender, receivers, err := s.openManyPeers(ctx, cfg.Bounds, cfg.ChunkSize, batch.Peers)
		if err != nil {
			return nil, err
		}
		plumber.SetSink[Request](pipe, peerSenderAddr, sender)
		receiverAddresses = make([]address.Address, len(receivers))
		for i, c := range receivers {
			addr := address.Newf("client-%v", i+1)
			receiverAddresses[i] = addr
			plumber.SetSource[Response](pipe, addr, c)
		}
	}

	if needGatewayRouting {
		routeInletTo = gatewayIterAddr
		gatewayIter, err := s.newGateway(Config{Keys: batch.Gateway, Bounds: cfg.Bounds, ChunkSize: cfg.ChunkSize})
		if err != nil {
			return nil, err
		}
		plumber.SetSegment[Request, Response](pipe, gatewayIterAddr, gatewayIter)
		receiverAddresses = append(receiverAddresses, gatewayIterAddr)
	}

	if needPeerRouting && needGatewayRouting {
		routeInletTo = broadcasterAddr
		plumber.SetSegment[Request, Request](
			pipe,
			broadcasterAddr,
			&confluence.DeltaMultiplier[Request]{},
		)
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
		newSynchronizer(len(cfg.Keys.UniqueLeaseholders())),
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
	v := validate.New("distribution.framer.Iterator")
	if validate.NotEmptySlice(v, "Keys", keys) {
		return v.Error()
	}
	for _, k := range keys {
		if k.Free() {
			return errors.Wrapf(validate.Error, "cannot read from free channel %v", k)
		}
	}
	q := s.ChannelReader.NewRetrieve().WhereKeys(keys...)
	exists, err := q.Exists(ctx, nil)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(query.NotFound, "some channel keys %v not found", keys)
	}
	return nil
}
