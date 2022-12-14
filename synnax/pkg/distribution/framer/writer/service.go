// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package writer exposes the Synnax cluster for framed writes as a monolithic data space.
// It provides a Writer interface that automatically handles the distribution of writes
// across the cluster. It also provides a StreamWriter interface that enables the user to
// optimize the concurrency of writes by passing requests and receiving responses through
// a channel (implementing the confluence.Segment interface).
//
// As Synnax is in its early stages, the writer package still has a number of issues, the
// most relevant of which is a lack of proper distributed transaction support. This means
// that commits that succeed on one node may fail on another. Caveat emptor.
package writer

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Config is the configuration necessary for opening a Writer or StreamWriter.
type Config struct {
	// Keys is keys to write to. At least one key must be provided. All keys must
	// have the same data rate OR the same index. All Frames written to the Writer must
	// have an array specified for each key, and all arrays must be the same length (i.e.
	// calls to Frame.Even must return true).
	// [REQUIRED]
	Keys channel.Keys `json:"keys" msgpack:"keys"`
	// Start marks the starting timestamp of the first sample in the first frame. If
	// telemetry occupying the given timestamp already exists for the provided keys,
	// the writer will fail to open.
	// [REQUIRED]
	Start telem.TimeStamp `json:"start" msgpack:"start"`
}

func (c Config) toStorage() storage.WriterConfig {
	return storage.WriterConfig{Channels: c.Keys.Strings(), Start: c.Start}
}

// ServiceConfig is the configuration for opening a Writer or StreamWriter.
type ServiceConfig struct {
	// TS is the local time series store to write to.
	// [REQUIRED]
	TS storage.StreamWritableTS
	// ChannelReader is used to resolve metadata and routing information for the provided
	// keys.
	// [REQUIRED]
	ChannelReader channel.Reader
	// HostResolver is used to resolve the host address for nodes in the cluster in order
	// to route writes.
	// [REQUIRED]
	HostResolver dcore.HostResolver
	// Transport is the network transport for sending and receiving writes from other
	// nodes in the cluster.
	// [REQUIRED]
	Transport Transport
	// Logger is the witness of it all.
	// [OPTIONAL]
	Logger *zap.Logger
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultConfig is the default configuration for opening a Writer or StreamWriter. It
	// is not complete and must be supplemented with the required fields.
	DefaultConfig = ServiceConfig{Logger: zap.NewNop()}
)

// Validate implements config.Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.writer")
	validate.NotNil(v, "TS", cfg.TS)
	validate.NotNil(v, "ChannelReader", cfg.ChannelReader)
	validate.NotNil(v, "HostResolver", cfg.HostResolver)
	validate.NotNil(v, "Transport", cfg.Transport)
	validate.NotNil(v, "Logger", cfg.Logger)
	return v.Error()
}

// Override implements config.Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.ChannelReader = override.Nil(cfg.ChannelReader, other.ChannelReader)
	cfg.HostResolver = override.Nil(cfg.HostResolver, other.HostResolver)
	cfg.Transport = override.Nil(cfg.Transport, other.Transport)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	return cfg
}

// Service is the central service for the writer package, allowing the caller to open
// Writers and StreamWriters for writing data to the cluster.
type Service struct {
	ServiceConfig
	server *server
}

// OpenService opens the writer service using the given configuration. Also binds a server
// to the given transport for receiving writes from other nodes in the cluster.
func OpenService(configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	return &Service{ServiceConfig: cfg, server: startServer(cfg)}, err
}

const (
	synchronizerAddr       = address.Address("synchronizer")
	peerSenderAddr         = address.Address("peerSender")
	gatewayWriterAddr      = address.Address("gatewayWriter")
	peerGatewaySwitchAddr  = address.Address("peerGatewaySwitch")
	validatorAddr          = address.Address("validator")
	validatorResponsesAddr = address.Address("validatorResponses")
)

// New opens a new writer using the given configuration. The provided context is used to
// control the lifetime of goroutines spawned by the writer. If the given context is cancelled,
// the writer will immediately abort all pending writes and return an error.
func (s *Service) New(ctx context.Context, cfg Config) (Writer, error) {
	sCtx, cancel := signal.WithCancel(
		ctx,
		signal.WithContextKey("writer"),
		signal.WithLogger(s.Logger),
	)
	seg, err := s.NewStream(ctx, cfg)
	if err != nil {
		return nil, err
	}
	req := confluence.NewStream[Request]()
	res := confluence.NewStream[Response]()
	seg.InFrom(req)
	seg.OutTo(res)
	seg.Flow(sCtx, confluence.CloseInletsOnExit(), confluence.CancelOnExitErr())
	return &writer{requests: req, responses: res, wg: sCtx, shutdown: cancel}, nil
}

// NewStream opens a new StreamWriter using the given configuration. The provided context
// is only used for opening the stream and is not used for concurrent flow control. The
// context for managing flow control must be provided to StreamWriter.Flow.
func (s *Service) NewStream(ctx context.Context, cfg Config) (StreamWriter, error) {
	if err := s.validateChannelKeys(ctx, cfg.Keys); err != nil {
		return nil, err
	}

	var (
		hostID             = s.HostResolver.HostID()
		batch              = proxy.NewBatchFactory[channel.Key](hostID).Batch(cfg.Keys)
		pipe               = plumber.New()
		needPeerRouting    = len(batch.Peers) > 0
		needGatewayRouting = len(batch.Gateway) > 0
		receiverAddresses  []address.Address
		routeBulkheadTo    address.Address
	)

	v := &validator{signal: make(chan bool, 1), keys: cfg.Keys}
	plumber.SetSegment[Request, Request](pipe, validatorAddr, v)
	plumber.SetSource[Response](pipe, validatorResponsesAddr, &v.responses)
	plumber.SetSegment[Response, Response](
		pipe,
		synchronizerAddr,
		newSynchronizer(len(cfg.Keys.UniqueNodeIDs()), v.signal),
	)

	if needPeerRouting {
		routeBulkheadTo = peerSenderAddr
		sender, receivers, _receiverAddresses, err := s.openManyPeers(ctx, batch.Peers)
		if err != nil {
			return nil, err
		}
		plumber.SetSink[Request](pipe, peerSenderAddr, sender)
		receiverAddresses = _receiverAddresses
		for i, receiver := range receivers {
			plumber.SetSource[Response](pipe, _receiverAddresses[i], receiver)
		}
	}

	if needGatewayRouting {
		routeBulkheadTo = gatewayWriterAddr
		w, err := s.newGateway(Config{Start: cfg.Start, Keys: batch.Gateway})
		if err != nil {
			return nil, err
		}
		plumber.SetSegment[Request, Response](pipe, gatewayWriterAddr, w)
		receiverAddresses = append(receiverAddresses, gatewayWriterAddr)
	}

	if needPeerRouting && needGatewayRouting {
		routeBulkheadTo = peerGatewaySwitchAddr
		plumber.SetSegment[Request, Request](
			pipe,
			peerGatewaySwitchAddr,
			newPeerGatewaySwitch(hostID),
		)
		plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{peerGatewaySwitchAddr},
			SinkTargets:   []address.Address{peerSenderAddr, gatewayWriterAddr},
			Stitch:        plumber.StitchWeave,
			Capacity:      2,
		}.MustRoute(pipe)
	}

	plumber.MustConnect[Request](pipe, validatorAddr, routeBulkheadTo, 1)

	plumber.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{synchronizerAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      len(receiverAddresses),
	}.MustRoute(pipe)

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo(validatorAddr))
	lo.Must0(seg.RouteOutletFrom(validatorResponsesAddr, synchronizerAddr))
	return seg, nil
}

func (s *Service) validateChannelKeys(ctx context.Context, keys channel.Keys) error {
	v := validate.New("distribution.framer.writer")
	if validate.NotEmptySlice(v, "keys", keys) {
		return v.Error()
	}
	var channels []channel.Channel
	if err := s.ChannelReader.
		NewRetrieve().
		Entries(&channels).
		WhereKeys(keys...).
		Exec(ctx); err != nil {
		return err
	}
	var (
		refIndex channel.Key
		refRate  telem.Rate
	)
	for i, c := range channels {
		if i == 0 {
			refIndex = c.Index()
			refRate = c.Rate
			continue
		}
		if c.Rate != 0 {
			if c.Rate != refRate {
				return v.Newf("channel rate mismatch: expected %s, found %s", c.Rate, refRate)
			}
		} else if c.Index() != refIndex {
			return v.Newf("keys must have the same index: expected %s, found %s", refIndex, c.Index())
		}
	}
	return nil
}
