// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/config"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// WriterMode sets the operating mode of the writer, optionally enabling or disabling
// persistence and streaming.
type WriterMode uint8

// Persist returns true if the current mode should persist data.
func (mode WriterMode) Persist() bool { return mode != WriterStreamOnly }

// Stream returns true if the current mode should stream data.
func (mode WriterMode) Stream() bool { return mode != WriterPersistOnly }

const (
	WriterPersistStream = iota + 1
	WriterPersistOnly
	WriterStreamOnly
)

// WriterConfig sets the configuration used to open a new writer on the DB.
type WriterConfig struct {
	// Name sets the human-readable name for the writer, which is useful for identifying
	// it in control transfer scenarios.
	// [OPTIONAL] - Defaults to an empty string.
	ControlSubject xcontrol.Subject
	// Start marks the starting timestamp of the first sample to be written by the
	// writer. If a sample exists for any channel at this timestamp, the writer will
	// fail to open.
	Start telem.TimeStamp
	// Channels sets the channels that the writer will write to. If a channel does not
	// exist, the writer fill fail to open.
	Channels []core.ChannelKey
	// Authorities marks the starting control authorities of the writer. This value must
	// be empty (so the default is applied), have a length of 1 (apply the same
	// authority to all channels), or have a length equal to the number of channels
	// (apply granular authorities to each channel).
	//
	// [OPTIONAL] - Defaults to control.AuthorityAbsolute on all channels.
	Authorities []xcontrol.Authority
	// ErrOnUnauthorized controls whether the writer will return an error when
	// attempting to open a writer on a channel that it does not have authority over.
	// This value should be set to false for control related scenarios.
	//
	// [OPTIONAL] - Defaults to false.
	ErrOnUnauthorized *bool
	// Mode sets the persistence and streaming mode of the writer. The default mode is
	// WriterModePersistStream. See the WriterMode documentation for more.
	//
	// [OPTIONAL] - Defaults to WriterModePersistStream.
	Mode WriterMode
	// EnableAutoCommit determines whether the writer will automatically commit after
	// each write. If EnableAutoCommit is true, then the writer will commit after each
	// write, and will flush that commit to index on FS after the specified
	// AutoIndexPersistInterval.
	//
	// [OPTIONAL] - Defaults to true.
	EnableAutoCommit *bool
	// AutoIndexPersistInterval is the interval at which commits to the index will be
	// persisted. To persist every commit to guarantee minimal loss of data, set
	// AutoIndexPersistInterval to AlwaysIndexPersistOnAutoCommit.
	//
	// [OPTIONAL] - Defaults to 1s.
	AutoIndexPersistInterval telem.TimeSpan
	// Sync sets whether the writer should acknowledge all write requests with a
	// corresponding writer response. Defaults to false, in which the writer will
	// acknowledge Commit() and SetAuthority() commands, but not Write commands. Using
	// sync mode is useful for acknowledging writes, but can clobber performance as the
	// next write cannot be started before the previous write is completed.
	//
	// [OPTIONAL] - Defaults to false.
	Sync *bool
}

const AlwaysIndexPersistOnAutoCommit telem.TimeSpan = -1

var (
	_ config.Config[WriterConfig] = WriterConfig{}
)

func DefaultWriterConfig() WriterConfig {
	return WriterConfig{
		ControlSubject:           xcontrol.Subject{Key: uuid.New().String()},
		Authorities:              []xcontrol.Authority{xcontrol.AuthorityAbsolute},
		ErrOnUnauthorized:        config.False(),
		Mode:                     WriterPersistStream,
		EnableAutoCommit:         config.True(),
		AutoIndexPersistInterval: 1 * telem.Second,
		Sync:                     config.False(),
	}
}

// Validate implements config.Config.
func (c WriterConfig) Validate() error {
	v := validate.New("cesium.writer_config")
	validate.NotEmptySlice(v, "channels", c.Channels)
	validate.NotNil(v, "err_on_unauthorized_open", c.ErrOnUnauthorized)
	validate.NotNil(v, "sync", c.Sync)
	validate.NotEmptyString(v, "control_subject.key", c.ControlSubject.Key)
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
	c.ControlSubject.Name = override.String(c.ControlSubject.Name, other.ControlSubject.Name)
	c.ControlSubject.Key = override.String(c.ControlSubject.Key, other.ControlSubject.Key)
	c.ErrOnUnauthorized = override.Nil(c.ErrOnUnauthorized, other.ErrOnUnauthorized)
	c.Mode = override.Numeric(c.Mode, other.Mode)
	c.Sync = override.Nil(c.Sync, other.Sync)
	c.EnableAutoCommit = override.Nil(c.EnableAutoCommit, other.EnableAutoCommit)
	c.AutoIndexPersistInterval = override.Zero(c.AutoIndexPersistInterval, other.AutoIndexPersistInterval)
	return c
}

func (c WriterConfig) authority(i int) xcontrol.Authority {
	if len(c.Authorities) == 1 {
		return c.Authorities[0]
	}
	return c.Authorities[i]
}

// NewStreamWriter implements DB.
func (db *DB) NewStreamWriter(ctx context.Context, cfgs ...WriterConfig) (StreamWriter, error) {
	if db.closed.Load() {
		return nil, errDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.newStreamWriter(ctx, cfgs...)
}

// OpenWriter implements DB.
func (db *DB) OpenWriter(ctx context.Context, cfgs ...WriterConfig) (*Writer, error) {
	if db.closed.Load() {
		return nil, errDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	iw, err := db.newStreamWriter(ctx, cfgs...)
	if err != nil {
		return nil, err
	}
	return wrapStreamWriter(iw.WriterConfig, iw), nil
}

func (db *DB) newStreamWriter(ctx context.Context, cfgs ...WriterConfig) (w *streamWriter, err error) {
	cfg, err := config.New(DefaultWriterConfig(), cfgs...)
	if err != nil {
		return nil, err
	}
	var (
		domainWriters  map[ChannelKey]*idxWriter
		virtualWriters map[ChannelKey]*virtual.Writer
		controlUpdate  ControlUpdate
	)
	defer func() {
		if err == nil {
			return
		}
		for _, idx := range domainWriters {
			_, err_ := idx.Close()
			err = errors.Combine(err_, err)
		}
	}()

	makeUnaryConfig := func(
		i int,
		domainAlignment uint32,
	) unary.WriterConfig {
		return unary.WriterConfig{
			Subject:                  cfg.ControlSubject,
			ErrOnUnauthorizedOpen:    cfg.ErrOnUnauthorized,
			EnableAutoCommit:         cfg.EnableAutoCommit,
			AutoIndexPersistInterval: cfg.AutoIndexPersistInterval,
			Start:                    cfg.Start,
			Persist:                  config.Bool(cfg.Mode.Persist()),
			Authority:                cfg.authority(i),
			AlignmentDomainIndex:     domainAlignment,
		}
	}

	// We do two passes when opening all individual writers. The first pass:
	// 1. Opens all virtual writers.
	// 2. Opens all write based writers.
	// 3. Opens the indexes of all domain indexed writers (if the indexes are in the
	//    list of channels).
	//
	// For the second pass, we open all indexed writers for particular indexes. This
	// ensures that we provide a valid domain alignment to all unary writers for a
	// particular index group.
	for i, key := range cfg.Channels {
		u, isUnary := db.mu.unaryDBs[key]
		v, isVirtual := db.mu.virtualDBs[key]
		if !isVirtual && !isUnary {
			return nil, core.NewErrChannelNotFound(key)
		}
		var (
			auth     = cfg.authority(i)
			transfer control.Transfer
		)
		if isVirtual {
			// If the channel is virtual.
			if virtualWriters == nil {
				virtualWriters = make(map[ChannelKey]*virtual.Writer)
			}
			virtualWriters[key], transfer, err = v.OpenWriter(ctx, virtual.WriterConfig{
				Subject:               cfg.ControlSubject,
				Start:                 cfg.Start,
				Authority:             auth,
				ErrOnUnauthorizedOpen: cfg.ErrOnUnauthorized,
			})
			if err != nil {
				return nil, err
			}
		} else if u.Channel().IsIndex {
			var unaryW *unary.Writer
			unaryW, transfer, err = u.OpenWriter(
				ctx,
				// A domain alignment of 0 lets the writer choose the domain alignment,
				// which is what we want for an index.
				makeUnaryConfig(i, 0),
			)
			if err != nil {
				return nil, err
			}
			// Hot path optimization: in the common case we only write to virtual channels
			// XOR indexed channels. In either case we can avoid a map allocation.
			if domainWriters == nil {
				domainWriters = make(map[ChannelKey]*idxWriter)
			}
			// If there is no existing index writer for this index-group.
			idxW, err := db.openDomainIdxWriter(u.Channel().Index, cfg)
			if err != nil {
				return nil, err
			}
			idxW.writingToIdx = true
			idxW.domainAlignment = unaryW.DomainIndex()
			idxW.internal[key] = &unaryWriterState{Writer: *unaryW}
			domainWriters[u.Channel().Index] = idxW
		}
		if transfer.Occurred() {
			controlUpdate.Transfers = append(controlUpdate.Transfers, transfer)
		}
	}

	// On the second pass, we open all domain-indexed writers that have indexes.
	for i, key := range cfg.Channels {
		u, uOk := db.mu.unaryDBs[key]
		// Ignore virtual, index, and rate-based channels.
		if !uOk || u.Channel().IsIndex || u.Channel().Index == 0 {
			continue
		}
		idxW, ok := domainWriters[u.Channel().Index]
		if !ok {
			if domainWriters == nil {
				domainWriters = make(map[ChannelKey]*idxWriter)
			}
			idxW, err = db.openDomainIdxWriter(u.Channel().Index, cfg)
			if err != nil {
				return nil, err
			}
			idxW.writingToIdx = false
			domainWriters[u.Channel().Index] = idxW
		}
		unaryW, transfer, err := u.OpenWriter(
			ctx,
			makeUnaryConfig(i, idxW.domainAlignment),
		)
		if err != nil {
			return nil, err
		}
		if transfer.Occurred() {
			controlUpdate.Transfers = append(controlUpdate.Transfers, transfer)
		}
		idxW.internal[key] = &unaryWriterState{Writer: *unaryW}
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
		virtual:      &virtualWriter{internal: virtualWriters, digestKey: db.mu.digests.key},
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

func (db *DB) openDomainIdxWriter(
	idxKey ChannelKey,
	cfg WriterConfig,
) (*idxWriter, error) {
	u, ok := db.mu.unaryDBs[idxKey]
	if !ok {
		return nil, core.NewErrChannelNotFound(idxKey)
	}
	w := &idxWriter{internal: make(map[ChannelKey]*unaryWriterState)}
	w.idx.ch = u.Channel()
	w.idx.Domain = u.Index()
	w.idx.highWaterMark = cfg.Start
	w.writingToIdx = false
	w.start = cfg.Start
	return w, nil
}
