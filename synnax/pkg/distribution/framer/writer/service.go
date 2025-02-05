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

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
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
	// Keys is keys to write to. At least one key must be provided. All keys must
	// have the same data rate OR the same index. All Frames written to the Writer must
	// have an array specified for each key, and all series must be the same length (i.e.
	// calls to Frame.Even must return true).
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
	// EnableAutoCommit determines whether the writer will automatically commit after each write.
	// If EnableAutoCommit is true, then the writer will commit after each write, and will
	// flush that commit to index on FS after the specified AutoIndexPersistInterval.
	// [OPTIONAL] - Defaults to false.
	EnableAutoCommit *bool `json:"enable_auto_commit" msgpack:"enable_auto_commit"`
	// AutoIndexPersistInterval is the interval at which commits to the index will be persisted.
	// To persist every commit to guarantee minimal loss of data, set AutoIndexPersistInterval
	// to AlwaysAutoPersist.
	// [OPTIONAL] - Defaults to 1s.
	AutoIndexPersistInterval telem.TimeSpan `json:"auto_index_persist_interval" msgpack:"auto_index_persist_interval"`
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
func (k keyAuthority) Lease() dcore.NodeKey { return k.key.Lease() }

var _ config.Config[Config] = Config{}

func DefaultConfig() Config {
	return Config{
		ControlSubject: control.Subject{
			Key: uuid.New().String(),
		},
		Authorities:              []control.Authority{control.Absolute},
		ErrOnUnauthorized:        config.False(),
		Mode:                     ts.WriterPersistStream,
		EnableAutoCommit:         config.False(),
		AutoIndexPersistInterval: 1 * telem.Second,
	}
}

// keyAuthorities returns a slice of keyAuthority structs that can be used to shard
// channel keys across multiple nodes in the cluster. This method should only be valled
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
	}
}

// Validate implements config.Properties.
func (c Config) Validate() error {
	v := validate.New("distribution.framer.writer")
	validate.NotEmptySlice(v, "keys", c.Keys)
	validate.NotEmptyString(v, "ControlSubject.Task", c.ControlSubject.Key)
	v.Ternaryf(
		"authorities",
		len(c.Authorities) != 1 && len(c.Authorities) != len(c.Keys),
		"authorities must be a single authority or a slice of authorities with the same length as keys",
	)
	return v.Error()
}

// Override implements config.Properties.
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
	return c
}

// ServiceConfig is the configuration for opening a Writer or StreamWriter.
type ServiceConfig struct {
	alamos.Instrumentation
	// TS is the local time series store to write to.
	// [REQUIRED]
	TS *ts.DB
	// ChannelReader is used to resolve metadata and routing information for the provided
	// keys.
	// [REQUIRED]
	ChannelReader channel.Readable
	// HostResolver is used to resolve the host address for nodes in the cluster in order
	// to route writes.
	// [REQUIRED]
	HostResolver dcore.HostResolver
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

// Validate implements config.Properties.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.writer")
	validate.NotNil(v, "TS", cfg.TS)
	validate.NotNil(v, "ChannelReader", cfg.ChannelReader)
	validate.NotNil(v, "HostProvider", cfg.HostResolver)
	validate.NotNil(v, "Transport", cfg.Transport)
	return v.Error()
}

// Override implements config.Properties.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.ChannelReader = override.Nil(cfg.ChannelReader, other.ChannelReader)
	cfg.HostResolver = override.Nil(cfg.HostResolver, other.HostResolver)
	cfg.Transport = override.Nil(cfg.Transport, other.Transport)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.FreeWrites = override.Nil(cfg.FreeWrites, other.FreeWrites)
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
	cfg, err := config.New(DefaultServiceConfig, configs...)
	return &Service{ServiceConfig: cfg, server: startServer(cfg)}, err
}

const (
	synchronizerAddr       = address.Address("synchronizer")
	peerSenderAddr         = address.Address("peerSender")
	gatewayWriterAddr      = address.Address("gatewayWriter")
	freeWriterAddr         = address.Address("freeWriter")
	peerGatewaySwitchAddr  = address.Address("peerGatewayFreeSwitch")
	validatorAddr          = address.Address("validator")
	validatorResponsesAddr = address.Address("validatorResponses")
)

// New opens a new writer using the given configuration. The provided context is used to
// control the lifetime of goroutines spawned by the writer. If the given context is cancelled,
// the writer will immediately abort all pending writes and return an error.
func (s *Service) New(ctx context.Context, cfgs ...Config) (*Writer, error) {
	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Instrumentation))
	seg, err := s.NewStream(ctx, cfgs...)
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
		requests:  req,
		responses: res,
		wg:        sCtx,
		shutdown:  signal.NewShutdown(sCtx, cancel),
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

	if err := s.validateChannelKeys(ctx, cfg.Keys); err != nil {
		return nil, err
	}

	var (
		hostKey           = s.HostResolver.HostKey()
		batch             = proxy.BatchFactory[keyAuthority]{Host: hostKey}.Batch(cfg.keyAuthorities())
		pipe              = plumber.New()
		hasPeer           = len(batch.Peers) > 0
		hasGateway        = len(batch.Gateway) > 0
		hasFree           = len(batch.Free) > 0
		receiverAddresses []address.Address
		routeValidatorTo  address.Address
	)

	v := &validator{signal: make(chan bool, 1), keys: cfg.Keys}
	plumber.SetSegment[Request, Request](pipe, validatorAddr, v)
	plumber.SetSource[Response](pipe, validatorResponsesAddr, &v.responses)
	plumber.SetSegment[Response, Response](
		pipe,
		synchronizerAddr,
		newSynchronizer(len(cfg.Keys.UniqueLeaseholders()), v.signal),
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
		plumber.SetSink[Request](pipe, peerSenderAddr, sender)
		receiverAddresses = _receiverAddresses
		for i, receiver := range receivers {
			plumber.SetSource[Response](pipe, _receiverAddresses[i], receiver)
		}
	}

	if hasGateway {
		routeValidatorTo = gatewayWriterAddr
		switchTargets = append(switchTargets, gatewayWriterAddr)
		w, err := s.newGateway(ctx, cfg.setKeyAuthorities(batch.Gateway))
		if err != nil {
			return nil, err
		}
		plumber.SetSegment[Request, Response](pipe, gatewayWriterAddr, w)
		receiverAddresses = append(receiverAddresses, gatewayWriterAddr)
	}

	if hasFree {
		routeValidatorTo = freeWriterAddr
		switchTargets = append(switchTargets, freeWriterAddr)
		w := s.newFree(cfg.Mode)
		plumber.SetSegment[Request, Response](pipe, freeWriterAddr, w)
		receiverAddresses = append(receiverAddresses, freeWriterAddr)
	}

	if len(switchTargets) > 1 {
		routeValidatorTo = peerGatewaySwitchAddr
		plumber.SetSegment[Request, Request](
			pipe,
			peerGatewaySwitchAddr,
			newPeerGatewayFreeSwitch(hostKey, hasPeer, hasGateway, hasFree),
		)
		plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{peerGatewaySwitchAddr},
			SinkTargets:   switchTargets,
			Stitch:        plumber.StitchWeave,
			Capacity:      len(switchTargets),
		}.MustRoute(pipe)
	}

	plumber.MustConnect[Request](pipe, validatorAddr, routeValidatorTo, 1)

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
		Exec(ctx, nil); err != nil {
		return err
	}
	if len(channels) != len(keys) {
		missing, _ := lo.Difference(keys, channel.KeysFromChannels(channels))
		return errors.Wrapf(validate.Error, "missing channels: %v", missing)
	}
	return nil
}
