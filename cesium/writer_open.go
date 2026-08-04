// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/config"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// WriterMode sets the operating mode of the writer, optionally enabling or disabling
// persistence and streaming.
type WriterMode uint8

// Persist returns true if the current mode should persist data.
func (mode WriterMode) Persist() bool { return mode != WriterModeStreamOnly }

// Stream returns true if the current mode should stream data.
func (mode WriterMode) Stream() bool { return mode != WriterModePersistOnly }

const (
	WriterModePersistStream WriterMode = iota + 1
	WriterModePersistOnly
	WriterModeStreamOnly
)

// WriterConfig sets the configuration used to open a new writer on the DB.
type WriterConfig struct {
	// ErrOnUnauthorized controls whether the writer will return an error when
	// attempting to open a writer on a channel that it does not have authority over.
	// This value should be set to false for control related scenarios.
	//
	// [OPTIONAL] - Defaults to false.
	ErrOnUnauthorized *bool
	// EnableAutoCommit determines whether the writer will automatically commit after
	// each write. If EnableAutoCommit is true, then the writer will commit after each
	// write, and will flush that commit to index on FS after the specified
	// AutoIndexPersistInterval.
	//
	// [OPTIONAL] - Defaults to true.
	EnableAutoCommit *bool
	// Sync sets whether the writer should acknowledge all write requests with a
	// corresponding writer response. Defaults to false, in which the writer will
	// acknowledge Commit() and SetAuthority() commands, but not Write commands. Using
	// sync mode is useful for acknowledging writes, but can clobber performance as the
	// next write cannot be started before the previous write is completed.
	//
	// [OPTIONAL] - Defaults to false.
	Sync *bool
	// Name sets the human-readable name for the writer, which is useful for identifying
	// it in control transfer scenarios.
	//
	// [OPTIONAL] - Defaults to an empty string.
	ControlSubject xcontrol.Subject
	// Channels sets the channels that the writer will write to. If a channel does not
	// exist, the writer fill fail to open.
	Channels []channel.Key
	// Authorities marks the starting control authorities of the writer. This value must
	// be empty (so the default is applied), have a length of 1 (apply the same
	// authority to all channels), or have a length equal to the number of channels
	// (apply granular authorities to each channel).
	//
	// [OPTIONAL] - Defaults to control.AuthorityAbsolute on all channels.
	Authorities []xcontrol.Authority
	// Start marks the starting timestamp of the first sample to be written by the
	// writer. If a sample exists for any channel at this timestamp, the writer will
	// fail to open.
	Start telem.TimeStamp
	// AutoIndexPersistInterval is the interval at which commits to the index will be
	// persisted. To persist every commit to guarantee minimal loss of data, set
	// AutoIndexPersistInterval to AlwaysIndexPersistOnAutoCommit.
	//
	// [OPTIONAL] - Defaults to 1s.
	AutoIndexPersistInterval telem.TimeSpan
	// Mode sets the persistence and streaming mode of the writer. The default mode is
	// WriterModePersistStream. See the WriterMode documentation for more.
	//
	// [OPTIONAL] - Defaults to WriterModePersistStream.
	Mode WriterMode
	// AutoIndex causes the writer to generate timestamps for any index channel
	// referenced by the writer's data channels whose series is omitted from a Write
	// frame. The first sample in each Write call is stamped with telem.Now() on this
	// node; remaining samples in the same call are spaced 1ns apart. Auto-stamps are
	// strictly monotonic across Write calls — the next call's first sample is greater
	// than the last sample of the previous auto-stamp.
	//
	// When AutoIndex is true, any index channel referenced by a data channel in
	// Channels but not present in Channels itself is implicitly opened for writing.
	// SetAuthority calls that name a data channel propagate to its index channel,
	// taking the max authority across all data channels referencing that index.
	//
	// When AutoIndex is true and Start is left as its zero value, Start is defaulted to
	// telem.Now() at open time so the writer's domain aligns with the auto-stamped
	// timestamps. Callers who pass an explicit index series whose timestamps fall
	// before this defaulted Start will have that write rejected.
	//
	// [OPTIONAL] - Defaults to false.
	AutoIndex *bool
}

const AlwaysIndexPersistOnAutoCommit telem.TimeSpan = -1

var _ config.Config[WriterConfig] = WriterConfig{}

// Validate implements config.Config.
func (c WriterConfig) Validate() error {
	v := validate.New("cesium.writer_config")
	validate.NotEmptySlice(v, "channels", c.Channels)
	validate.NotNil(v, "err_on_unauthorized_open", c.ErrOnUnauthorized)
	validate.NotNil(v, "sync", c.Sync)
	validate.NotNil(v, "auto_index", c.AutoIndex)
	v.Exec(c.ControlSubject.Validate)
	v.Ternary(
		"authorities",
		len(c.Authorities) != len(c.Channels) && len(c.Authorities) != 1,
		"authority count must be 1 or equal to channel count",
	)
	return v.Error()
}

// Override implements config.Config.
func (c WriterConfig) Override(other WriterConfig) WriterConfig {
	c.Start = override.Zero(c.Start, other.Start)
	c.Channels = override.Slice(c.Channels, other.Channels)
	c.Authorities = override.Slice(c.Authorities, other.Authorities)
	c.ControlSubject = c.ControlSubject.Override(other.ControlSubject)
	c.ErrOnUnauthorized = override.Nil(c.ErrOnUnauthorized, other.ErrOnUnauthorized)
	c.Mode = override.Numeric(c.Mode, other.Mode)
	c.Sync = override.Nil(c.Sync, other.Sync)
	c.EnableAutoCommit = override.Nil(c.EnableAutoCommit, other.EnableAutoCommit)
	c.AutoIndexPersistInterval = override.Zero(
		c.AutoIndexPersistInterval,
		other.AutoIndexPersistInterval,
	)
	c.AutoIndex = override.Nil(c.AutoIndex, other.AutoIndex)
	return c
}

func (c WriterConfig) authority(i int) xcontrol.Authority {
	if len(c.Authorities) == 1 {
		return c.Authorities[0]
	}
	return c.Authorities[i]
}

// NewStreamWriter implements DB.
func (db *DB) NewStreamWriter(
	ctx context.Context,
	cfgs ...WriterConfig,
) (StreamWriter, error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.newStreamWriter(ctx, cfgs...)
}

// OpenWriter implements DB.
func (db *DB) OpenWriter(ctx context.Context, cfgs ...WriterConfig) (*Writer, error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	iw, err := db.newStreamWriter(ctx, cfgs...)
	if err != nil {
		return nil, err
	}
	return wrapStreamWriter(iw.WriterConfig, iw), nil
}

func (db *DB) newStreamWriter(
	ctx context.Context,
	cfgs ...WriterConfig,
) (w *streamWriter, err error) {
	cfg, err := config.New(WriterConfig{
		ControlSubject:           xcontrol.Subject{Key: uuid.New().String()},
		Authorities:              []xcontrol.Authority{xcontrol.AuthorityAbsolute},
		ErrOnUnauthorized:        new(false),
		Mode:                     WriterModePersistStream,
		EnableAutoCommit:         new(true),
		AutoIndexPersistInterval: 1 * telem.Second,
		Sync:                     new(false),
		AutoIndex:                new(false),
	}, cfgs...)
	if err != nil {
		return nil, err
	}
	if *cfg.AutoIndex {
		if cfg.Start.IsZero() {
			cfg.Start = telem.Now()
		}
		cfg = db.expandKeysForAutoIndex(cfg)
	}
	var (
		domainWriters  = make(map[ChannelKey]*idxWriter)
		virtualWriters map[ChannelKey]*virtual.Writer
		controlUpdate  ControlUpdate
		keyToIdx       map[ChannelKey]*idxWriter
	)
	if *cfg.AutoIndex {
		keyToIdx = make(map[ChannelKey]*idxWriter, len(cfg.Channels))
	}
	defer func() {
		if err == nil {
			return
		}
		for _, idx := range domainWriters {
			_, errClose := idx.Close()
			err = errors.Combine(errClose, err)
		}
		for _, vw := range virtualWriters {
			_, errClose := vw.Close()
			err = errors.Combine(errClose, err)
		}
	}()

	makeUnaryConfig := func(i int) unary.WriterConfig {
		return unary.WriterConfig{
			Subject:                  cfg.ControlSubject,
			ErrOnUnauthorizedOpen:    cfg.ErrOnUnauthorized,
			EnableAutoCommit:         cfg.EnableAutoCommit,
			AutoIndexPersistInterval: cfg.AutoIndexPersistInterval,
			Start:                    cfg.Start,
			Persist:                  new(cfg.Mode.Persist()),
			Authority:                cfg.authority(i),
		}
	}

	// Two passes:
	//   Pass 1: Open virtual writers and index-channel writers.
	//   Pass 2: Open all non-index unary writers (both fixed-density and
	//     variable-length) and attach them to the appropriate idxWriter group.
	// Alignment is not decided here. Data channels resolve it from their index on every
	// write, so a writer that joins an existing control region cannot end up in a
	// different alignment space than the index it is being written against.
	for i, key := range cfg.Channels {
		u, isUnary := db.mu.dbs.unary[key]
		v, isVirtual := db.mu.dbs.virtual[key]
		if !isVirtual && !isUnary {
			return nil, channel.NewNotFoundError(key)
		}
		if isUnary && !u.Channel().IsIndex {
			continue
		}
		var transfer control.Transfer
		if isVirtual {
			if virtualWriters == nil {
				virtualWriters = make(map[ChannelKey]*virtual.Writer)
			}
			virtualWriters[key], transfer, err = v.OpenWriter(ctx, virtual.WriterConfig{
				Subject:               cfg.ControlSubject,
				Start:                 cfg.Start,
				Authority:             cfg.authority(i),
				ErrOnUnauthorizedOpen: cfg.ErrOnUnauthorized,
			})
			if err != nil {
				return nil, err
			}
		} else {
			var uW *unary.Writer
			uW, transfer, err = u.OpenWriter(ctx, makeUnaryConfig(i))
			if err != nil {
				return nil, err
			}
			var idxW *idxWriter
			idxW, err = db.openDomainIdxWriter(u.Channel().Index, cfg)
			if err != nil {
				return nil, err
			}
			idxW.writingToIdx = true
			idxW.internal[key] = &unaryWriterState{Writer: *uW}
			domainWriters[u.Channel().Index] = idxW
			if *cfg.AutoIndex {
				keyToIdx[key] = idxW
				idxW.dataAuth = make(map[ChannelKey]xcontrol.Authority)
			}
		}
		if transfer.Occurred() {
			controlUpdate.Transfers = append(controlUpdate.Transfers, transfer)
		}
	}

	// Pass 2: open all non-index unary channel writers (fixed and variable) and attach
	// them to the appropriate idxWriter group.
	for i, key := range cfg.Channels {
		u, ok := db.mu.dbs.unary[key]
		if !ok || u.Channel().IsIndex || u.Channel().Index == 0 {
			continue
		}
		idxKey := u.Channel().Index
		idxW, ok := domainWriters[idxKey]
		if !ok {
			idxW, err = db.openDomainIdxWriter(idxKey, cfg)
			if err != nil {
				return nil, err
			}
			idxW.writingToIdx = false
			domainWriters[idxKey] = idxW
		}
		var (
			uW       *unary.Writer
			transfer control.Transfer
		)
		uW, transfer, err = u.OpenWriter(ctx, makeUnaryConfig(i))
		if err != nil {
			return nil, err
		}
		if transfer.Occurred() {
			controlUpdate.Transfers = append(controlUpdate.Transfers, transfer)
		}
		idxW.internal[key] = &unaryWriterState{Writer: *uW}
		if *cfg.AutoIndex {
			keyToIdx[key] = idxW
			if idxW.writingToIdx {
				idxW.dataAuth[key] = cfg.authority(i)
			}
		}
	}

	if len(controlUpdate.Transfers) > 0 {
		if err = db.updateControlDigests(ctx, controlUpdate); err != nil {
			return nil, err
		}
	}

	w = &streamWriter{
		WriterConfig: cfg,
		internal:     make([]*idxWriter, 0, len(domainWriters)),
		relay:        db.relay.inlet,
		virtual: &virtualWriter{
			internal:  virtualWriters,
			digestKey: db.mu.digests.key,
		},
		keyToIdx: keyToIdx,
		updateDBControl: func(ctx context.Context, update ControlUpdate) error {
			db.mu.RLock()
			defer db.mu.RUnlock()
			return db.updateControlDigests(ctx, update)
		},
	}
	for _, idx := range domainWriters {
		w.internal = append(w.internal, idx)
	}
	return w, nil
}

// expandKeysForAutoIndex returns a config whose Channels include any index channels
// referenced by non-index channels in cfg.Channels that are not already present. When
// per-channel authorities are supplied, each appended index inherits the maximum
// authority of the data channels that reference it, since writing to a data channel
// requires successfully writing its index. When a single broadcast authority is
// supplied, the appended index keys naturally inherit it.
func (db *DB) expandKeysForAutoIndex(cfg WriterConfig) WriterConfig {
	existing := set.New(cfg.Channels...)
	perChannelAuth := len(cfg.Authorities) > 1
	indexToAuth := make(map[ChannelKey]xcontrol.Authority)
	var implicit []ChannelKey
	for i, k := range cfg.Channels {
		u, ok := db.mu.dbs.unary[k]
		if !ok {
			continue
		}
		ch := u.Channel()
		if ch.IsIndex {
			continue
		}
		idxKey := ch.Index
		if idxKey == 0 {
			continue
		}
		if existing.Contains(idxKey) {
			continue
		}
		if _, seen := indexToAuth[idxKey]; !seen {
			implicit = append(implicit, idxKey)
			indexToAuth[idxKey] = 0
		}
		if perChannelAuth {
			if a := cfg.Authorities[i]; a > indexToAuth[idxKey] {
				indexToAuth[idxKey] = a
			}
		}
	}
	if len(implicit) == 0 {
		return cfg
	}
	cfg.Channels = append(cfg.Channels, implicit...)
	if perChannelAuth {
		indexAuths := lo.Map(implicit, func(k ChannelKey, _ int) xcontrol.Authority {
			return indexToAuth[k]
		})
		cfg.Authorities = append(cfg.Authorities, indexAuths...)
	}
	return cfg
}

func (db *DB) openDomainIdxWriter(
	idxKey ChannelKey,
	cfg WriterConfig,
) (*idxWriter, error) {
	u, ok := db.mu.dbs.unary[idxKey]
	if !ok {
		return nil, channel.NewNotFoundError(idxKey)
	}
	w := &idxWriter{internal: make(map[ChannelKey]*unaryWriterState)}
	w.idx.ch = u.Channel()
	w.idx.Domain = u.Index()
	w.idx.db = &u
	w.idx.highWaterMark = cfg.Start
	w.idx.autoStampClock = cfg.Start
	w.writingToIdx = false
	w.start = cfg.Start
	w.lastCommitEnd = cfg.Start
	return w, nil
}
