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
	"slices"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// WriterConfig sets the configuration used to open a new writer on the DB.
type WriterConfig struct {
	// Channels are the keys that the writer writes: its clock, the channels that the
	// clock times, or both. A writer of channels without their clock writes against the
	// stored stamps of the clock.
	//
	// [REQUIRED]
	Channels []channel.Key
	// Clock is the key of the clock that times every channel of the writer.
	//
	// [REQUIRED]
	Clock channel.Key
	// Start marks the start of the writer's claim. When Start is zero, the region
	// starts at the time the writer opens. If a sample exists for any channel at a
	// non-zero Start, the writer fails to open. The domains of the clock start at the
	// first timestamp written to it, not at Start. The region moves back to a domain
	// that starts before it.
	//
	// A writer that writes channels without their clock needs a Start that is a sample
	// of the clock.
	//
	// [OPTIONAL]
	Start telem.TimeStamp
	// Bounded limits the writer's claim to the time it has committed, so that it can
	// write behind a live writer. The writer fails to open when another writer holds
	// its Start, and a commit fails when it reaches another writer's claim.
	//
	// [OPTIONAL] - Defaults to false.
	Bounded bool
}

// commitsNeverPersist is the persist interval of the internal writers. Their commits
// never persist the index: Writer.Sync and Writer.Close do.
const commitsNeverPersist = telem.TimeSpanMax

var _ config.Config[WriterConfig] = WriterConfig{}

// Validate implements config.Config.
func (c WriterConfig) Validate() error {
	v := validate.New("cesium.writer_config")
	v.NotEmptySlice("channels", c.Channels)
	v.Ternary("clock", c.Clock == 0, "must be set")
	return v.Error()
}

// Override implements config.Config.
func (c WriterConfig) Override(other WriterConfig) WriterConfig {
	c.Start = override.Zero(c.Start, other.Start)
	c.Channels = override.Slice(c.Channels, other.Channels)
	c.Clock = override.Numeric(c.Clock, other.Clock)
	c.Bounded = c.Bounded || other.Bounded
	return c
}

func (c WriterConfig) unaryConfig() unary.WriterConfig {
	return unary.WriterConfig{
		EnableAutoCommit:         new(true),
		AutoIndexPersistInterval: commitsNeverPersist,
		Start:                    c.Start,
		Bounded:                  new(c.Bounded),
	}
}

// OpenWriter implements DB.
func (db *DB) OpenWriter(ctx context.Context, cfgs ...WriterConfig) (*Writer, error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	cfg, err := config.New(WriterConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	idx, err := db.openClockWriter(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Writer{WriterConfig: cfg, idx: idx}, nil
}

// openClockWriter opens the writers of the clock of cfg and of each channel of cfg that
// it times. Alignment is not decided here: data channels resolve it from their clock on
// every write, so a writer that joins an existing claim cannot end up in a different
// alignment space than the clock it writes against. The caller must hold db.mu.
func (db *DB) openClockWriter(
	ctx context.Context,
	cfg WriterConfig,
) (_ *idxWriter, err error) {
	idx, err := db.openDomainIdxWriter(cfg.Clock, cfg)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Combine(err, idx.Close())
		}
	}()
	// The clock opens first, so that it holds its claim before its channels do.
	if slices.Contains(cfg.Channels, cfg.Clock) {
		if err = db.openChannelWriter(ctx, cfg, idx, cfg.Clock); err != nil {
			return nil, err
		}
		idx.writingToIdx = true
	} else if cfg.Start.IsZero() {
		return nil, channel.Wrap(cfg.Channels[0], errors.Wrapf(
			validate.ErrValidation,
			"writer needs a start to write the channel without its index %v",
			idx.idx.ch,
		))
	}
	for _, key := range cfg.Channels {
		if key == cfg.Clock {
			continue
		}
		u, ok := db.mu.dbs.unary[key]
		if !ok {
			return nil, channel.NewNotFoundError(key)
		}
		if ch := u.Channel(); ch.IsIndex || ch.Index != cfg.Clock {
			return nil, channel.Wrap(key, errors.Wrapf(
				validate.ErrValidation,
				"channel is not timed by the writer clock %v",
				idx.idx.ch,
			))
		}
		if err = db.openChannelWriter(ctx, cfg, idx, key); err != nil {
			return nil, err
		}
	}
	return idx, nil
}

// openChannelWriter opens the unary writer of key and adds it to idx. The caller must
// hold db.mu.
func (db *DB) openChannelWriter(
	ctx context.Context,
	cfg WriterConfig,
	idx *idxWriter,
	key ChannelKey,
) error {
	u, ok := db.mu.dbs.unary[key]
	if !ok {
		return channel.NewNotFoundError(key)
	}
	uW, err := u.OpenWriter(ctx, cfg.unaryConfig())
	if err != nil {
		return err
	}
	idx.internal[key] = &unaryWriterState{Writer: *uW}
	return nil
}

func (db *DB) openDomainIdxWriter(
	idxKey ChannelKey,
	cfg WriterConfig,
) (*idxWriter, error) {
	u, ok := db.mu.dbs.unary[idxKey]
	if !ok {
		return nil, channel.NewNotFoundError(idxKey)
	}
	if !u.Channel().IsIndex {
		return nil, channel.Wrap(idxKey, errors.Wrap(
			validate.ErrValidation,
			"writer clock is not an index",
		))
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
