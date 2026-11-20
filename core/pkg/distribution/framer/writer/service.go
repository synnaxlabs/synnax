// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package writer writes frames to channels. It provides a Writer with a method-based
// interface, and a StreamWriter that passes requests and responses through streams
// (implementing the confluence.Segment interface).
package writer

import (
	"context"
	"slices"
	"sync/atomic"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration necessary for opening a Writer or StreamWriter.
type Config struct {
	// ErrOnUnauthorized controls whether the writer will return an error when
	// attempting to write to a channel that it does not have authority over. In
	// non-control scenarios, this value should be set to true. In scenarios that
	// require control handoff, this value should be set to false.
	//
	// [OPTIONAL] - Defaults to False
	ErrOnUnauthorized *bool
	// EnableAutoCommit determines whether the writer will automatically commit after
	// each write. If EnableAutoCommit is true, then the writer will commit after each
	// write, and will flush that commit to index on FS after the specified
	// AutoIndexPersistInterval.
	//
	// [OPTIONAL] - Defaults to true.
	EnableAutoCommit *bool
	// Sync is set to true if the writer should send acknowledgments for every write
	// request, not just on failed requests.
	//
	// This only applies to write operations, as the writer will always send
	// acknowledgments for calls to Commit and SetAuthority.
	//
	// This setting is good for testing and debugging purposes, as it provides
	// guarantees that a writer has successfully processed a frame, but can have a
	// considerable performance impact.
	//
	// [OPTIONAL] - Defaults to false.
	Sync *bool
	// ControlSubject is an identifier for the writer.
	ControlSubject control.Subject
	// Keys are the channel keys to write to. At least one key must be provided. All
	// Frames written to the Writer must have a array specified for each key, and all
	// series must be the same length (i.e. calls Frame.Even must return true).
	//
	// [REQUIRED]
	Keys channel.Keys
	// Authorities sets the control authority the writer has on each channel for the
	// write. This should either be a single authority for all channels or a slice of
	// authorities with the same length as the number of channels where each authority
	// corresponds to the channel at the same index. Defaults to absolute authority for
	// all channels.
	//
	// [OPTIONAL]
	Authorities []control.Authority
	// Start marks the start of the writer's control region. When Start is zero, the
	// region starts at the time the writer opens. If telemetry occupying a non-zero
	// Start already exists for the provided keys, the writer fails to open.
	//
	// [OPTIONAL]
	Start telem.TimeStamp
	// AutoIndexPersistInterval is the least time between two syncs of auto commits. To
	// sync every auto commit, so that no committed sample is lost, set it to
	// AlwaysIndexPersistOnAutoCommit. A writer that does not auto commit syncs every
	// commit.
	//
	// [OPTIONAL] - Defaults to 1s.
	AutoIndexPersistInterval telem.TimeSpan
	// Mode sets the persistence and streaming mode for the writer. The default mode is
	// WriterModePersistStream.
	//
	// [OPTIONAL] - Defaults to WriterModePersistStream.
	Mode Mode
	// AutoIndex causes the writer to generate timestamps for any index channel in its
	// Keys whose series is omitted from a Write frame. The first sample in each Write
	// call is stamped with telem.Now(), or 1ns after Start or after the last stamp of
	// the index when either is later; remaining samples in the same call are spaced 1ns
	// apart.
	//
	// When AutoIndex is true and the caller's Keys reference data channels whose index
	// channels are not also in Keys, those index channels are implicitly added to the
	// writer at open time so the storage layer opens them for writing.
	//
	// [OPTIONAL] - Defaults to false.
	AutoIndex *bool
}

var _ config.Config[Config] = Config{}

// DefaultConfig is the default configuration for opening a new writer. This
// configuration is not valid by itself and must be overridden by the required fields
// specified in Config.
func DefaultConfig() Config {
	return Config{
		ControlSubject:           control.Subject{Key: uuid.New().String()},
		Authorities:              []control.Authority{control.AuthorityAbsolute},
		ErrOnUnauthorized:        new(false),
		Mode:                     cesium.WriterModePersistStream,
		EnableAutoCommit:         new(true),
		AutoIndexPersistInterval: 1 * telem.Second,
		Sync:                     new(false),
		AutoIndex:                new(false),
	}
}

// split returns a copy of c holding only the channels that Cesium stores, the copy
// holding only the leased virtual channels, and the keys of the free channels. Each
// copy gives each of its channels its own authority. It must only be called after c has
// been validated.
func (c Config) split(
	virtual set.Set[channel.Key],
) (stored, leased Config, free channel.Keys) {
	stored, leased = c, c
	stored.Keys = make(channel.Keys, 0, len(c.Keys))
	stored.Authorities = make([]control.Authority, 0, len(c.Keys))
	leased.Keys, leased.Authorities = nil, nil
	for i, key := range c.Keys {
		auth := c.Authorities[i%len(c.Authorities)]
		switch {
		case key.Free():
			free = append(free, key)
		case virtual.Contains(key):
			leased.Keys = append(leased.Keys, key)
			leased.Authorities = append(leased.Authorities, auth)
		default:
			stored.Keys = append(stored.Keys, key)
			stored.Authorities = append(stored.Authorities, auth)
		}
	}
	return stored, leased, free
}

func (c Config) toStorage() cesium.WriterConfig {
	return cesium.WriterConfig{
		Channels: c.Keys.Uint32(),
		Start:    c.Start,
		Mode:     c.Mode,
	}
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("distribution.framer.writer")
	v.NotEmptySlice("keys", c.Keys)
	v.NotEmptyString("control_subject.key", c.ControlSubject.Key)
	v.NotNil("enable_auto_commit", c.EnableAutoCommit)
	v.NotNil("sync", c.Sync)
	v.NotNil("err_on_unauthorized", c.ErrOnUnauthorized)
	v.NotNil("auto_index", c.AutoIndex)
	v.Ternaryf(
		"authorities",
		len(c.Authorities) != 1 && len(c.Authorities) != len(c.Keys),
		"authorities must be a single authority or a slice of authorities with the same length as keys",
	)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.ControlSubject.Name = override.String(
		c.ControlSubject.Name,
		other.ControlSubject.Name,
	)
	c.ControlSubject.Key = override.String(
		c.ControlSubject.Key,
		other.ControlSubject.Key,
	)
	c.ControlSubject.Group = override.Numeric(
		c.ControlSubject.Group,
		other.ControlSubject.Group,
	)
	c.Keys = override.Slice(c.Keys, other.Keys.Unique())
	c.Start = override.Zero(c.Start, other.Start)
	c.Authorities = override.Slice(c.Authorities, other.Authorities)
	c.ErrOnUnauthorized = override.Nil(c.ErrOnUnauthorized, other.ErrOnUnauthorized)
	c.Mode = override.Numeric(c.Mode, other.Mode)
	c.EnableAutoCommit = override.Nil(c.EnableAutoCommit, other.EnableAutoCommit)
	c.AutoIndexPersistInterval = override.Numeric(
		c.AutoIndexPersistInterval,
		other.AutoIndexPersistInterval,
	)
	c.Sync = override.Nil(c.Sync, other.Sync)
	c.AutoIndex = override.Nil(c.AutoIndex, other.AutoIndex)
	return c
}

// ServiceConfig is the configuration for opening a Writer Service.
type ServiceConfig struct {
	// Writes is where writers send every frame they write, for the relay to stream.
	//
	// [REQUIRED]
	Writes confluence.Inlet[relay.Response]
	// TS is the local time series store to write to.
	//
	// [REQUIRED]
	TS *cesium.DB
	// Clock times the AutoIndexPersistInterval of each writer. It must be monotonic.
	//
	// [OPTIONAL] - Defaults to xtime.Real.
	Clock xtime.Clock
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.writer")
	v.NotNil("writes", cfg.Writes)
	v.NotNil("ts", cfg.TS)
	v.NotNil("clock", cfg.Clock)
	return v.Error()
}

// Override implements config.Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.Writes = override.Nil(cfg.Writes, other.Writes)
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.Clock = override.Nil(cfg.Clock, other.Clock)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	return cfg
}

// ChannelResolver resolves the records of channels, which live above the distribution
// layer. The service-layer channel service implements it, and
// Service.SetChannelResolver registers it.
type ChannelResolver interface {
	// Resolve returns the record of each channel in keys that has one.
	Resolve(
		ctx context.Context,
		keys channel.Keys,
	) (map[channel.Key]channel.Channel, error)
}

// Service is the central service for the writer package, allowing the caller to open
// Writers and StreamWriters for writing data to channels.
type Service struct {
	// freeWriteAlignments tracks per-index alignment counters shared across all writers
	// opened on this service, so concurrent free writers stamp distinct alignment
	// domains.
	freeWriteAlignments *freeWriteAlignments
	// resolver looks up the records of free and virtual channels at open time. Set once
	// at startup via SetChannelResolver; opening a writer on such channels fails until
	// it is set.
	resolver atomic.Pointer[ChannelResolver]
	claims   *claims
	controls *controls
	cfg      ServiceConfig
}

// ControlStates returns the state of control over each channel that a writer controls.
func (s *Service) ControlStates() []ControlState { return s.controls.states() }

// OnControlUpdate calls handler with each batch of control transfers between writers.
func (s *Service) OnControlUpdate(
	handler func(context.Context, ControlUpdate),
) observe.Disconnect {
	return s.controls.updates.OnChange(handler)
}

// ClaimDelete claims keys for a delete and returns the function that releases the
// claim. While the claim is held, a writer on one of keys fails to open. ClaimDelete
// fails with ErrChannelInUse when a writer holds one of keys.
func (s *Service) ClaimDelete(keys channel.Keys) (release func(), err error) {
	if release, err = s.claims.claim(keys); err == nil {
		s.controls.forget(keys)
	}
	return release, err
}

// SetChannelResolver injects the resolver used to look up the records of free and
// virtual channels when a writer is opened. It must be called before opening writers on
// such channels; the service layer registers its channel service here at startup.
func (s *Service) SetChannelResolver(r ChannelResolver) { s.resolver.Store(&r) }

// NewService opens the writer service using the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{Clock: xtime.Real}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		cfg:      cfg,
		claims:   newClaims(),
		controls: newControls(),
		freeWriteAlignments: &freeWriteAlignments{
			counters: make(map[channel.Key]*telem.AlignmentCounter),
		},
	}, nil
}

const (
	synchronizerAddr  = address.Address("synchronizer")
	storageWriterAddr = address.Address("storage_writer")
	freeWriterAddr    = address.Address("free_writer")
	switchAddr        = address.Address("switch")
	sequencerAddr     = address.Address("sequencer")
)

// Open a new writer using the given configuration. The provided context is used to
// control the lifetime of goroutines spawned by the writer. If the given context is
// cancelled, the writer will immediately abort all pending writes and return an error.
func (s *Service) Open(ctx context.Context, cfgs ...Config) (*Writer, error) {
	sCtx, cancel := signal.WithCancel(
		ctx,
		signal.WithInstrumentation(s.cfg.Instrumentation),
	)
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

// NewStream opens a new StreamWriter using the given configuration. The provided
// context is only used for opening the stream and is not used for concurrent flow
// control. The context for managing flow control must be provided to StreamWriter.Flow.
func (s *Service) NewStream(ctx context.Context, cfgs ...Config) (StreamWriter, error) {
	cfg, err := config.New(DefaultConfig(), cfgs...)
	if err != nil {
		return nil, err
	}

	stored, virtualChannels, records, err := s.resolve(ctx, cfg.Keys)
	if err != nil {
		return nil, err
	}
	var (
		virtual = set.New(lo.Map(
			virtualChannels,
			func(ch cesium.Channel, _ int) channel.Key { return channel.Key(ch.Key) },
		)...)
		storedCfg, leased, freeKeys     = cfg.split(virtual)
		hasStorage                      = len(storedCfg.Keys) > 0
		hasFree                         = len(freeKeys) > 0 || len(leased.Keys) > 0
		pipe                            = plumber.New()
		receiverAddresses               []address.Address
		routeSequencerTo                address.Address
		freeIndexes                     map[channel.Key]channel.Key
		freeAlignments, leasedAlignment map[channel.Key]telem.Alignment
	)

	// Reserve the write alignments before any pipeline resources are opened, so a
	// failure requires no cleanup.
	if len(freeKeys) > 0 {
		freeIndexes = freeIndexesOf(freeKeys, records)
		if freeAlignments, err = s.freeWriteAlignments.reserve(
			freeKeys,
			freeIndexes,
		); err != nil {
			return nil, err
		}
	}
	if leasedAlignment, err = s.freeWriteAlignments.reserve(
		leased.Keys,
		nil,
	); err != nil {
		return nil, err
	}
	freeAlignments = lo.Assign(freeAlignments, leasedAlignment)
	leasedGates, transfers, err := s.controls.openGates(leased, virtualChannels)
	if err != nil {
		return nil, err
	}

	pipe.SetSegment(sequencerAddr, &sequencer{})
	pipe.SetSegment(synchronizerAddr, newSynchronizer(
		lo.Ternary(hasStorage, 1, 0)+lo.Ternary(hasFree, 1, 0),
		*cfg.Sync,
		s.cfg.Instrumentation,
	))

	if hasStorage {
		routeSequencerTo = storageWriterAddr
		w, err := s.newStorage(ctx, storedCfg, stored)
		if err != nil {
			leasedGates.release()
			return nil, err
		}
		pipe.SetSegment(storageWriterAddr, w)
		receiverAddresses = append(receiverAddresses, storageWriterAddr)
	}
	s.controls.notify(transfers)

	if hasFree {
		routeSequencerTo = freeWriterAddr
		w := s.newFree(
			cfg.Mode,
			*cfg.Sync,
			append(slices.Clone(freeKeys), leased.Keys...),
			freeIndexes,
			freeAlignments,
			cfg.ControlSubject.Group,
			leasedGates,
		)
		pipe.SetSegment(freeWriterAddr, w)
		receiverAddresses = append(receiverAddresses, freeWriterAddr)
	}

	if hasStorage && hasFree {
		routeSequencerTo = switchAddr
	}
	return newSegment(
		pipe,
		receiverAddresses,
		routeSequencerTo,
		newSwitch(set.New(storedCfg.Keys...)),
	), nil
}

// resolve returns the Cesium channels of the keys that Cesium stores, the channels of
// the leased virtual keys, and the records of the free and leased virtual keys. Cesium
// holds the facts of the channels it stores, and the resolver holds the facts of the
// others. From Phase 6, Cesium keeps no channel facts, and every key resolves here.
func (s *Service) resolve(
	ctx context.Context,
	keys channel.Keys,
) (
	stored, virtual []cesium.Channel,
	records map[channel.Key]channel.Channel,
	err error,
) {
	var (
		unresolved channel.Keys
		notFound   = make(map[channel.Key]error)
	)
	for _, k := range keys {
		if k.Free() {
			unresolved = append(unresolved, k)
			continue
		}
		ch, err := s.cfg.TS.RetrieveChannel(ctx, uint32(k))
		if errors.Is(err, cesium.ErrChannelNotFound) {
			unresolved = append(unresolved, k)
			notFound[k] = err
			continue
		}
		if err != nil {
			return nil, nil, nil, err
		}
		stored = append(stored, ch)
	}
	if len(unresolved) == 0 {
		return stored, nil, nil, nil
	}
	rp := s.resolver.Load()
	if rp == nil || *rp == nil {
		return nil, nil, nil, errors.New(
			"cannot open a writer on free or virtual channels: no channel resolver " +
				"configured",
		)
	}
	if records, err = (*rp).Resolve(ctx, unresolved); err != nil {
		return nil, nil, nil, err
	}
	for _, k := range unresolved {
		if k.Free() {
			continue
		}
		r, ok := records[k]
		if !ok || !r.Virtual {
			return nil, nil, nil, notFound[k]
		}
		virtual = append(virtual, cesium.Channel{
			Key:         uint32(k),
			Name:        r.Name,
			DataType:    r.DataType,
			Virtual:     true,
			Concurrency: r.Concurrency,
		})
	}
	return stored, virtual, records, nil
}

// freeIndexesOf returns the index of each free channel in keys whose record has one.
func freeIndexesOf(
	keys channel.Keys,
	records map[channel.Key]channel.Channel,
) map[channel.Key]channel.Key {
	indexes := make(map[channel.Key]channel.Key)
	for _, k := range keys {
		if r, ok := records[k]; ok && r.Index() != 0 {
			indexes[k] = r.Index()
		}
	}
	return indexes
}

// newSegment connects the sequencer to routeSequencerTo and the receivers to the
// synchronizer, then returns pipe as a segment from the sequencer to the synchronizer.
// When routeSequencerTo is the switch, it also routes the switch to the receivers.
func newSegment(
	pipe *plumber.Pipeline,
	receiverAddresses []address.Address,
	routeSequencerTo address.Address,
	sw func(context.Context, Request, map[address.Address]Request) error,
) *plumber.Segment[Request, Response] {
	if routeSequencerTo == switchAddr {
		pipe.SetSegment(
			switchAddr,
			&confluence.BatchSwitch[Request, Request]{Switch: sw},
		)
		plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{switchAddr},
			SinkTargets:   receiverAddresses,
			Stitch:        plumber.StitchWeave,
			Capacity:      len(receiverAddresses) * 5,
		}.MustRoute(pipe)
	}
	pipe.MustConnect[Request](sequencerAddr, routeSequencerTo, 30)
	plumber.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{synchronizerAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      len(receiverAddresses),
	}.MustRoute(pipe)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo(sequencerAddr))
	lo.Must0(seg.RouteOutletFrom(synchronizerAddr))
	return seg
}
