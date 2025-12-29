// Copyright 2025 Synnax Labs, Inc.
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
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration necessary for opening a Writer or StreamWriter.
type Config struct {
	// ControlSubject is an identifier for the writer.
	ControlSubject control.Subject `json:"control_subject" msgpack:"control_subject"`
	// Keys are the channel keys to write to. At least one key must be provided. All
	// Frames written to the Writer must have a array specified for each key, and all series must be the same length (i.e.
	// calls Frame.Even must return true).
	// [REQUIRED]
	Keys channel.Keys `json:"keys" msgpack:"keys"`
	// Start marks the starting timestamp of the first sample in the first frame. If
	// telemetry occupying the given timestamp already exists for the provided keys,
	// the writer will fail to open.
	// [REQUIRED]
	Start telem.TimeStamp `json:"start" msgpack:"start"`
	// Authorities sets the control authority the writer has on each channel for the
	// write. This should either be a single authority for all channels or a slice
	// of authorities with the same length as the number of channels where each
	// authority corresponds to the channel at the same index. Defaults to
	// absolute authority for all channels.
	// [OPTIONAL]
	Authorities []control.Authority `json:"authorities" msgpack:"authorities"`
	// ErrOnUnauthorized controls whether the writer will return an error when
	// attempting to write to a channel that it does not have authority over.
	// In non-control scenarios, this value should be set to true. In scenarios
	// that require control handoff, this value should be set to false.
	// [OPTIONAL] - Defaults to False
	ErrOnUnauthorized *bool
	// Mode sets the persistence and streaming mode for the writer. The default mode is
	// WriterModePersistStream. See the ts.WriterMode documentation for more.
	// [OPTIONAL] - Defaults to WriterModePersistStream.
	Mode ts.WriterMode `json:"mode" msgpack:"mode"`
	// EnableAutoCommit determines whether the writer will automatically commit after
	// each write. If EnableAutoCommit is true, then the writer will commit after each
	// write, and will flush that commit to index on FS after the specified
	// AutoIndexPersistInterval.
	//
	// [OPTIONAL] - Defaults to true.
	EnableAutoCommit *bool `json:"enable_auto_commit" msgpack:"enable_auto_commit"`
	// AutoIndexPersistInterval is the interval at which commits to the index will be persisted.
	// To persist every commit to guarantee minimal loss of data, set AutoIndexPersistInterval
	// to AlwaysAutoPersist.
	// [OPTIONAL] - Defaults to 1s.
	AutoIndexPersistInterval telem.TimeSpan `json:"auto_index_persist_interval" msgpack:"auto_index_persist_interval"`
	// Sync is set to true if the writer should send acknowledgements for every write
	// request, not just on failed requests.
	//
	// This only applies to write operations, as the writer will always send
	// acknowledgements for calls to Commit and SetAuthority.
	//
	// This setting is good for testing and debugging purposes, as it provides
	// guarantees that a writer has successfully processed a frame, but can have a
	// considerable performance impact.
	//
	// [OPTIONAL] - Defaults to false.
	Sync *bool `json:"sync" msgpack:"sync"`
}

func (c Config) setKeyAuthorities(authorities []keyAuthority) Config {
	c.Authorities = make([]control.Authority, len(authorities))
	c.Keys = make([]channel.Key, len(authorities))
	for i, authority := range authorities {
		c.Authorities[i] = authority.authority
		c.Keys[i] = authority.key
	}
	return c
}

// keyAuthority is a temporary struct that lets us shard channel keys across multiple
// nodes in the cluster. Most importantly, it implements proxy.Entry so that we can
// correctly split by host.
type keyAuthority struct {
	key       channel.Key
	authority control.Authority
}

var _ proxy.Entry = keyAuthority{}

// Lease implements proxy.Entry.
func (k keyAuthority) Lease() cluster.NodeKey { return k.key.Lease() }

var _ config.Config[Config] = Config{}

// DefaultConfig is the default configuration for opening a new writer. This
// configuration is not valid by itself and must be overridden by the required fields
// specified in Config.
func DefaultConfig() Config {
	return Config{
		ControlSubject:           control.Subject{Key: uuid.New().String()},
		Authorities:              []control.Authority{control.AuthorityAbsolute},
		ErrOnUnauthorized:        config.False(),
		Mode:                     ts.WriterPersistStream,
		EnableAutoCommit:         config.True(),
		AutoIndexPersistInterval: 1 * telem.Second,
		Sync:                     config.False(),
	}
}

// keyAuthorities returns a slice of keyAuthority structs that can be used to shard
// channel keys across multiple nodes in the cluster. This method should only be called
// after the config has been validated.
func (c Config) keyAuthorities() []keyAuthority {
	authorities := make([]keyAuthority, len(c.Keys))
	for i, key := range c.Keys {
		authorities[i] = keyAuthority{key: key, authority: c.Authorities[i%len(c.Authorities)]}
	}
	return authorities
}

func (c Config) toStorage() ts.WriterConfig {
	return ts.WriterConfig{
		ControlSubject:           c.ControlSubject,
		Channels:                 c.Keys.Storage(),
		Start:                    c.Start,
		Authorities:              c.Authorities,
		ErrOnUnauthorized:        c.ErrOnUnauthorized,
		Mode:                     c.Mode,
		EnableAutoCommit:         c.EnableAutoCommit,
		AutoIndexPersistInterval: c.AutoIndexPersistInterval,
		Sync:                     c.Sync,
	}
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("distribution.framer.writer")
	validate.NotEmptySlice(v, "keys", c.Keys)
	validate.NotEmptyString(v, "control_subject.key", c.ControlSubject.Key)
	validate.NotNil(v, "enable_auto_commit", c.EnableAutoCommit)
	validate.NotNil(v, "sync", c.Sync)
	validate.NotNil(v, "err_on_unauthorized", c.ErrOnUnauthorized)
	v.Ternaryf(
		"authorities",
		len(c.Authorities) != 1 && len(c.Authorities) != len(c.Keys),
		"authorities must be a single authority or a slice of authorities with the same length as keys",
	)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.ControlSubject.Name = override.String(c.ControlSubject.Name, other.ControlSubject.Name)
	c.ControlSubject.Key = override.String(c.ControlSubject.Key, other.ControlSubject.Key)
	c.Keys = override.Slice(c.Keys, other.Keys.Unique())
	c.Start = override.Zero(c.Start, other.Start)
	c.Authorities = override.Slice(c.Authorities, other.Authorities)
	c.ErrOnUnauthorized = override.Nil(c.ErrOnUnauthorized, other.ErrOnUnauthorized)
	c.Mode = override.Numeric(c.Mode, other.Mode)
	c.EnableAutoCommit = override.Nil(c.EnableAutoCommit, other.EnableAutoCommit)
	c.AutoIndexPersistInterval = override.Numeric(c.AutoIndexPersistInterval, other.AutoIndexPersistInterval)
	c.Sync = override.Nil(c.Sync, other.Sync)
	return c
}

// ServiceConfig is the configuration for opening a Writer or StreamWriter.
type ServiceConfig struct {
	alamos.Instrumentation
	// TS is the local time series store to write to.
	// [REQUIRED]
	TS *ts.DB
	// Channel is used to resolve metadata and routing information for the provided
	// keys.
	//
	// [REQUIRED]
	Channel *channel.Service
	// HostResolver is used to resolve the host address for nodes in the cluster in order
	// to route writes.
	// [REQUIRED]
	HostResolver cluster.HostResolver
	// Transport is the network transport for sending and receiving writes from other
	// nodes in the cluster.
	// [REQUIRED]
	Transport Transport
	// FreeWrites is the write pipeline where samples from free channels should be
	// written.
	FreeWrites confluence.Inlet[relay.Response]
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening the writer Service.
	DefaultServiceConfig = ServiceConfig{}
)

// Validate implements config.Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.writer")
	validate.NotNil(v, "ts", cfg.TS)
	validate.NotNil(v, "channel", cfg.Channel)
	validate.NotNil(v, "host_provider", cfg.HostResolver)
	validate.NotNil(v, "transport", cfg.Transport)
	return v.Error()
}

// Override implements config.Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.Channel = override.Nil(cfg.Channel, other.Channel)
	cfg.HostResolver = override.Nil(cfg.HostResolver, other.HostResolver)
	cfg.Transport = override.Nil(cfg.Transport, other.Transport)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.FreeWrites = override.Nil(cfg.FreeWrites, other.FreeWrites)
	return cfg
}

// Service is the central service for the writer package, allowing the caller to open
// Writers and StreamWriters for writing data to the cluster.
type Service struct {
	cfg                 ServiceConfig
	server              *server
	freeWriteAlignments *freeWriteAlignments
}

// NewService opens the writer service using the given configuration. Also binds a
// server to the given transport for receiving writes from other nodes in the cluster.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		cfg:    cfg,
		server: startServer(cfg),
		freeWriteAlignments: &freeWriteAlignments{
			alignments: make(map[channel.Key]*atomic.Uint32),
		},
	}, nil
}

const (
	synchronizerAddr       = address.Address("synchronizer")
	peerSenderAddr         = address.Address("peer_sender")
	gatewayWriterAddr      = address.Address("gateway_writer")
	freeWriterAddr         = address.Address("free_writer")
	peerGatewaySwitchAddr  = address.Address("peer_gateway_free_switch")
	validatorAddr          = address.Address("validator")
	validatorResponsesAddr = address.Address("validator_responses")
)

// Open a new writer using the given configuration. The provided context is used to
// control the lifetime of goroutines spawned by the writer. If the given context is
// cancelled, the writer will immediately abort all pending writes and return an error.
func (s *Service) Open(ctx context.Context, cfgs ...Config) (*Writer, error) {
	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.cfg.Instrumentation))
	cfg, err := config.New(DefaultConfig(), cfgs...)
	if err != nil {
		return nil, err
	}
	seg, err := s.NewStream(ctx, cfg)
	if err != nil {
		return nil, err
	}
	req := confluence.NewStream[Request]()
	res := confluence.NewStream[Response]()
	seg.InFrom(req)
	seg.OutTo(res)
	seg.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.CancelOnFail(),
		confluence.RecoverWithErrOnPanic(),
	)
	return &Writer{
		cfg:       cfg,
		requests:  req,
		responses: res,
		shutdown:  signal.NewHardShutdown(sCtx, cancel),
	}, nil
}

// NewStream opens a new StreamWriter using the given configuration. The provided context
// is only used for opening the stream and is not used for concurrent flow control. The
// context for managing flow control must be provided to StreamWriter.Flow.
func (s *Service) NewStream(ctx context.Context, cfgs ...Config) (StreamWriter, error) {
	cfg, err := config.New(DefaultConfig(), cfgs...)
	if err != nil {
		return nil, err
	}

	channels, err := s.validateChannelKeys(ctx, cfg.Keys)
	if err != nil {
		return nil, err
	}

	var (
		hostKey           = s.cfg.HostResolver.HostKey()
		batch             = proxy.BatchFactory[keyAuthority]{Host: hostKey}.Batch(cfg.keyAuthorities())
		pipe              = plumber.New()
		hasPeer           = len(batch.Peers) > 0
		hasGateway        = len(batch.Gateway) > 0
		hasFree           = len(batch.Free) > 0
		receiverAddresses []address.Address
		routeValidatorTo  address.Address
	)

	v := &validator{keys: cfg.Keys}
	plumber.SetSegment(pipe, validatorAddr, v)
	plumber.SetSource(pipe, validatorResponsesAddr, &v.responses)
	plumber.SetSegment(
		pipe,
		synchronizerAddr,
		newSynchronizer(len(cfg.Keys.UniqueLeaseholders()), s.cfg.Instrumentation),
	)

	switchTargets := make([]address.Address, 0, 3)
	if hasPeer {
		routeValidatorTo = peerSenderAddr
		switchTargets = append(switchTargets, peerSenderAddr)
		sender, receivers, _receiverAddresses, err := s.openManyPeers(
			ctx,
			cfg,
			batch.Peers,
		)
		if err != nil {
			return nil, err
		}
		plumber.SetSink(pipe, peerSenderAddr, sender)
		receiverAddresses = _receiverAddresses
		for i, receiver := range receivers {
			plumber.SetSource(pipe, _receiverAddresses[i], receiver)
		}
	}

	if hasGateway {
		routeValidatorTo = gatewayWriterAddr
		switchTargets = append(switchTargets, gatewayWriterAddr)
		w, err := s.newGateway(ctx, cfg.setKeyAuthorities(batch.Gateway))
		if err != nil {
			return nil, err
		}
		plumber.SetSegment(pipe, gatewayWriterAddr, w)
		receiverAddresses = append(receiverAddresses, gatewayWriterAddr)
	}

	if hasFree {
		routeValidatorTo = freeWriterAddr
		switchTargets = append(switchTargets, freeWriterAddr)
		w := s.newFree(cfg.Mode, *cfg.Sync, channels)
		plumber.SetSegment(pipe, freeWriterAddr, w)
		receiverAddresses = append(receiverAddresses, freeWriterAddr)
	}

	if len(switchTargets) > 1 {
		routeValidatorTo = peerGatewaySwitchAddr
		plumber.SetSegment(
			pipe,
			peerGatewaySwitchAddr,
			newPeerGatewayFreeSwitch(hostKey, hasPeer, hasGateway, hasFree),
		)
		plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{peerGatewaySwitchAddr},
			SinkTargets:   switchTargets,
			Stitch:        plumber.StitchWeave,
			Capacity:      len(switchTargets) * 5,
		}.MustRoute(pipe)
	}

	plumber.MustConnect[Request](pipe, validatorAddr, routeValidatorTo, 30)

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

func (s *Service) validateChannelKeys(ctx context.Context, keys channel.Keys) ([]channel.Channel, error) {
	v := validate.New("distribution.framer.writer")
	if validate.NotEmptySlice(v, "keys", keys) {
		return nil, v.Error()
	}
	var channels []channel.Channel
	if err := s.cfg.Channel.
		NewRetrieve().
		Entries(&channels).
		WhereKeys(keys...).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	if len(channels) != len(keys) {
		missing, _ := lo.Difference(keys, channel.KeysFromChannels(channels))
		return nil, errors.Wrapf(validate.Error, "missing channels: %v", missing)
	}
	return channels, nil
}
