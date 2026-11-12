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
	"sync/atomic"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
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
	// AutoIndexPersistInterval is the interval at which commits to the index will be
	// persisted. To persist every commit to guarantee minimal loss of data, set
	// AutoIndexPersistInterval to AlwaysAutoPersist.
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
	// call is stamped with telem.Now(); remaining samples in the same call are spaced
	// 1ns apart. A high-water mark guarantees strict monotonicity across Write calls.
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
		Mode:                     ts.WriterModePersistStream,
		EnableAutoCommit:         new(true),
		AutoIndexPersistInterval: 1 * telem.Second,
		Sync:                     new(false),
		AutoIndex:                new(false),
	}
}

// splitFree returns a copy of c holding only the channels that have storage, each with
// its own authority, and the keys of the free channels. It must only be called after c
// has been validated.
func (c Config) splitFree() (stored Config, free channel.Keys) {
	stored = c
	stored.Keys = make(channel.Keys, 0, len(c.Keys))
	stored.Authorities = make([]control.Authority, 0, len(c.Keys))
	for i, key := range c.Keys {
		if key.Free() {
			free = append(free, key)
			continue
		}
		stored.Keys = append(stored.Keys, key)
		stored.Authorities = append(
			stored.Authorities,
			c.Authorities[i%len(c.Authorities)],
		)
	}
	return stored, free
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
		AutoIndex:                c.AutoIndex,
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
	// FreeWrites is the write pipeline where samples from free channels should be
	// written.
	//
	// [REQUIRED]
	FreeWrites confluence.Inlet[relay.Response]
	// TS is the local time series store to write to.
	//
	// [REQUIRED]
	TS *ts.DB
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.writer")
	v.NotNil("free_writes", cfg.FreeWrites)
	v.NotNil("ts", cfg.TS)
	return v.Error()
}

// Override implements config.Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.FreeWrites = override.Nil(cfg.FreeWrites, other.FreeWrites)
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	return cfg
}

// FreeIndexResolver resolves the index channels of free channels, whose records live
// above the distribution layer. The service-layer channel service implements this
// interface and is registered via Service.SetFreeIndexResolver.
type FreeIndexResolver interface {
	// ResolveFreeIndexes returns the index channel key for each free channel in keys
	// that has an index. Free frames are stamped with per-index alignments derived from
	// this map, and a free channel it leaves out counts its own alignments. Any error
	// is propagated to the caller opening the writer.
	ResolveFreeIndexes(
		context.Context, channel.Keys,
	) (map[channel.Key]channel.Key, error)
}

// Service is the central service for the writer package, allowing the caller to open
// Writers and StreamWriters for writing data to channels.
type Service struct {
	// freeWriteAlignments tracks per-index alignment counters shared across all writers
	// opened on this service, so concurrent free writers stamp distinct alignment
	// domains.
	freeWriteAlignments *freeWriteAlignments
	// resolver looks up free channel indexes at open time. Set once at startup via
	// SetFreeIndexResolver; opening a writer on free channels fails until it is set.
	resolver atomic.Pointer[FreeIndexResolver]
	cfg      ServiceConfig
}

// SetFreeIndexResolver injects the resolver used to look up free channel indexes when
// a writer is opened. It must be called before opening writers on free channels; the
// service layer registers its channel service here at startup.
func (s *Service) SetFreeIndexResolver(r FreeIndexResolver) { s.resolver.Store(&r) }

// NewService opens the writer service using the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		cfg: cfg,
		freeWriteAlignments: &freeWriteAlignments{
			counters: make(map[channel.Key]*cesium.AlignmentCounter),
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

	var (
		stored, freeKeys  = cfg.splitFree()
		hasStorage        = len(stored.Keys) > 0
		hasFree           = len(freeKeys) > 0
		pipe              = plumber.New()
		receiverAddresses []address.Address
		routeSequencerTo  address.Address
	)

	// Resolve free channel indexes and alignments before any pipeline resources are
	// opened, so a failure requires no cleanup.
	var (
		freeIndexes    map[channel.Key]channel.Key
		freeAlignments map[channel.Key]telem.Alignment
	)
	if hasFree {
		rp := s.resolver.Load()
		if rp == nil || *rp == nil {
			return nil, errors.New(
				"cannot open a writer on free channels: no free index resolver configured",
			)
		}
		if freeIndexes, err = (*rp).ResolveFreeIndexes(ctx, freeKeys); err != nil {
			return nil, err
		}
		freeAlignments, err = s.freeWriteAlignments.reserve(freeKeys, freeIndexes)
		if err != nil {
			return nil, err
		}
	}

	pipe.SetSegment(sequencerAddr, &sequencer{})
	pipe.SetSegment(synchronizerAddr, newSynchronizer(
		lo.Ternary(hasStorage, 1, 0)+lo.Ternary(hasFree, 1, 0),
		s.cfg.Instrumentation,
	))

	if hasStorage {
		routeSequencerTo = storageWriterAddr
		w, err := s.newStorage(ctx, stored)
		if err != nil {
			return nil, err
		}
		pipe.SetSegment(storageWriterAddr, w)
		receiverAddresses = append(receiverAddresses, storageWriterAddr)
	}

	if hasFree {
		routeSequencerTo = freeWriterAddr
		w := s.newFree(
			cfg.Mode,
			*cfg.Sync,
			freeKeys,
			freeIndexes,
			freeAlignments,
			cfg.ControlSubject.Group,
		)
		pipe.SetSegment(freeWriterAddr, w)
		receiverAddresses = append(receiverAddresses, freeWriterAddr)
	}

	if hasStorage && hasFree {
		routeSequencerTo = switchAddr
		pipe.SetSegment(
			switchAddr,
			&confluence.BatchSwitch[Request, Request]{Switch: switchFree},
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
	return seg, nil
}
