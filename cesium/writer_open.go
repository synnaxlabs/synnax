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

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// WriterMode sets the operating mode of the writer, optionally enabling or disabling
// persistence and streaming.
type WriterMode uint8

func (mode WriterMode) persist() bool { return mode != WriterModeStreamOnly }

// Stream returns true if the current mode should stream data.
func (mode WriterMode) Stream() bool { return mode != WriterModePersistOnly }

const (
	WriterModePersistStream WriterMode = iota + 1
	// WriterModePersistOnly stores samples without streaming them. The writer holds
	// only the time it has committed and can write behind a live writer. It fails to
	// open when another writer holds its Start, and a commit fails when it reaches
	// another writer's claim. It ignores virtual channels.
	WriterModePersistOnly
	WriterModeStreamOnly
)

// WriterConfig sets the configuration used to open a new writer on the DB.
type WriterConfig struct {
	// Channels sets the channels that the writer will write to. If a channel does not
	// exist, the writer fill fail to open.
	Channels []channel.Key
	// Start marks the start of the writer's claim. When Start is zero, the
	// region starts at the time the writer opens. If a sample exists for any channel at
	// a non-zero Start, the writer fails to open. The domains of each index start at
	// the first timestamp written to that index, not at Start. The region moves back to
	// a domain that starts before it.
	//
	// A writer that writes data channels without their index needs a Start that is a
	// sample of that index.
	//
	// [OPTIONAL]
	Start telem.TimeStamp
	// Mode sets the persistence and streaming mode of the writer. The default mode is
	// WriterModePersistStream. See the WriterMode documentation for more.
	//
	// [OPTIONAL] - Defaults to WriterModePersistStream.
	Mode WriterMode
}

// commitsNeverPersist is the persist interval of the internal writers. Their commits
// never persist the index: Writer.Sync and Writer.Close do.
const commitsNeverPersist = telem.TimeSpanMax

var _ config.Config[WriterConfig] = WriterConfig{}

// Validate implements config.Config.
func (c WriterConfig) Validate() error {
	v := validate.New("cesium.writer_config")
	v.NotEmptySlice("channels", c.Channels)
	return v.Error()
}

// Override implements config.Config.
func (c WriterConfig) Override(other WriterConfig) WriterConfig {
	c.Start = override.Zero(c.Start, other.Start)
	c.Channels = override.Slice(c.Channels, other.Channels)
	c.Mode = override.Numeric(c.Mode, other.Mode)
	return c
}

func (c WriterConfig) unaryConfig() unary.WriterConfig {
	return unary.WriterConfig{
		EnableAutoCommit:         new(true),
		AutoIndexPersistInterval: commitsNeverPersist,
		Start:                    c.Start,
		Persist:                  new(c.Mode.persist()),
		Bounded:                  new(!c.Mode.Stream()),
	}
}

// OpenWriter implements DB.
func (db *DB) OpenWriter(ctx context.Context, cfgs ...WriterConfig) (*Writer, error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.newWriter(ctx, cfgs...)
}

func (db *DB) newWriter(
	ctx context.Context,
	cfgs ...WriterConfig,
) (w *Writer, err error) {
	cfg, err := config.New(WriterConfig{Mode: WriterModePersistStream}, cfgs...)
	if err != nil {
		return nil, err
	}
	var (
		domainWriters = make(map[ChannelKey]*idxWriter)
		keyToIdx      = make(map[ChannelKey]*idxWriter, len(cfg.Channels))
	)
	defer func() {
		if err != nil {
			err = closeWriters(err, domainWriters)
		}
	}()

	// Two passes:
	//   Pass 1: Open index-channel writers.
	//   Pass 2: Open all non-index unary writers (both fixed-density and
	//     variable-length) and attach them to the appropriate idxWriter group.
	// Alignment is not decided here. Data channels resolve it from their index on every
	// write, so a writer that joins an existing claim cannot end up in a different
	// alignment space than the index it is being written against.
	if err = db.openIndexWriters(ctx, cfg, domainWriters, keyToIdx); err != nil {
		return nil, err
	}
	if err = db.openDataWriters(ctx, cfg, domainWriters, keyToIdx); err != nil {
		return nil, err
	}

	return db.assembleWriter(cfg, domainWriters, keyToIdx), nil
}

// assembleWriter returns a Writer over the opened writers.
func (db *DB) assembleWriter(
	cfg WriterConfig,
	domainWriters map[ChannelKey]*idxWriter,
	keyToIdx map[ChannelKey]*idxWriter,
) *Writer {
	w := &Writer{
		WriterConfig: cfg,
		internal:     make([]*idxWriter, 0, len(domainWriters)),
		keyToIdx:     keyToIdx,
	}
	for _, idx := range domainWriters {
		w.internal = append(w.internal, idx)
	}
	return w
}

// closeWriters closes the given writers and returns err combined with any errors from
// closing them.
func closeWriters(err error, domainWriters map[ChannelKey]*idxWriter) error {
	for _, idx := range domainWriters {
		err = errors.Combine(idx.Close(), err)
	}
	return err
}

// openIndexWriters opens a writer for each index channel in cfg, adding the index
// writers to domainWriters and keyToIdx.
func (db *DB) openIndexWriters(
	ctx context.Context,
	cfg WriterConfig,
	domainWriters map[ChannelKey]*idxWriter,
	keyToIdx map[ChannelKey]*idxWriter,
) error {
	for i, key := range cfg.Channels {
		u, ok := db.mu.dbs.unary[key]
		if !ok {
			return channel.NewNotFoundError(key)
		}
		if !u.Channel().IsIndex {
			continue
		}
		idxW, err := db.openIndexWriter(ctx, cfg, &u, i)
		if err != nil {
			return err
		}
		domainWriters[u.Channel().Index] = idxW
		keyToIdx[key] = idxW
	}
	return nil
}

// openIndexWriter opens the writer for the index channel u at position i of cfg.
func (db *DB) openIndexWriter(
	ctx context.Context,
	cfg WriterConfig,
	u *unary.DB,
	i int,
) (*idxWriter, error) {
	uW, err := u.OpenWriter(ctx, cfg.unaryConfig())
	if err != nil {
		return nil, err
	}
	idxW, err := db.openDomainIdxWriter(u.Channel().Index, cfg)
	if err != nil {
		return nil, errors.Combine(err, uW.Close())
	}
	idxW.writingToIdx = true
	idxW.internal[cfg.Channels[i]] = &unaryWriterState{Writer: *uW}
	return idxW, nil
}

// openDataWriters opens a writer for each non-index unary channel in cfg and attaches
// it to the idxWriter of its index, opening that idxWriter when domainWriters lacks it.
func (db *DB) openDataWriters(
	ctx context.Context,
	cfg WriterConfig,
	domainWriters map[ChannelKey]*idxWriter,
	keyToIdx map[ChannelKey]*idxWriter,
) error {
	for _, key := range cfg.Channels {
		u, ok := db.mu.dbs.unary[key]
		if !ok || u.Channel().IsIndex || u.Channel().Index == 0 {
			continue
		}
		idxKey := u.Channel().Index
		idxW, ok := domainWriters[idxKey]
		if !ok {
			var err error
			if idxW, err = db.openDataOnlyIdxWriter(cfg, u.Channel()); err != nil {
				return err
			}
			domainWriters[idxKey] = idxW
		}
		uW, err := u.OpenWriter(ctx, cfg.unaryConfig())
		if err != nil {
			return err
		}
		idxW.internal[key] = &unaryWriterState{Writer: *uW}
		keyToIdx[key] = idxW
	}
	return nil
}

// openDataOnlyIdxWriter opens an idxWriter for the index of the data channel ch when
// the writer does not write that index.
func (db *DB) openDataOnlyIdxWriter(
	cfg WriterConfig,
	ch channel.Channel,
) (*idxWriter, error) {
	idxW, err := db.openDomainIdxWriter(ch.Index, cfg)
	if err != nil {
		return nil, err
	}
	if cfg.Start.IsZero() {
		return nil, channel.Wrap(ch.Key, errors.Wrapf(
			validate.ErrValidation,
			"writer needs a start to write the channel without its index %v",
			idxW.idx.ch,
		))
	}
	idxW.writingToIdx = false
	return idxW, nil
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
	w.writingToIdx = false
	w.start = cfg.Start
	w.lastCommitEnd = cfg.Start
	return w, nil
}
