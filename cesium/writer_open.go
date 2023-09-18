// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// WriterConfig sets the configuration used to open a new writer on the DB.
type WriterConfig struct {
	Name string
	// Start marks the starting timestamp of the first sample to be written by the
	// writer. If a sample exists for any channel at this timestamp, the writer will
	// fail to open.
	Start telem.TimeStamp
	// Channels sets the channels that the writer will write to. If a channel does
	// not exist, the writer fill fail to open.
	Channels []core.ChannelKey
	// Authorities marks the starting control authorities of the writer.
	Authorities        []control.Authority
	SendControlDigests bool
}

var (
	_                   config.Config[WriterConfig] = WriterConfig{}
	DefaultWriterConfig                             = WriterConfig{
		Authorities: []control.Authority{control.Absolute},
	}
)

// Validate implements config.Config.
func (w WriterConfig) Validate() error {
	v := validate.New("cesium.WriterConfig")
	validate.NotEmptySlice(v, "channels", w.Channels)
	v.Ternary(
		len(w.Authorities) != len(w.Channels) && len(w.Authorities) != 1,
		"authority count must be 1 or equal to channel count",
	)
	return v.Error()
}

// Override implements config.Config.
func (w WriterConfig) Override(other WriterConfig) WriterConfig {
	w.Start = override.Zero(w.Start, other.Start)
	w.Channels = override.Slice(w.Channels, other.Channels)
	w.Authorities = override.Slice(w.Authorities, other.Authorities)
	return w
}

func (w WriterConfig) authority(i int) control.Authority {
	if len(w.Authorities) == 1 {
		return w.Authorities[0]
	}
	return w.Authorities[i]
}

// NewStreamWriter implements DB.
func (db *DB) NewStreamWriter(ctx context.Context, cfgs ...WriterConfig) (StreamWriter, error) {
	return db.newStreamWriter(ctx, cfgs...)
}

// OpenWriter implements DB.
func (db *DB) OpenWriter(ctx context.Context, cfgs ...WriterConfig) (*Writer, error) {
	internal, err := db.newStreamWriter(ctx, cfgs...)
	if err != nil {
		return nil, err
	}
	return wrapStreamWriter(internal), nil
}

func (db *DB) newStreamWriter(ctx context.Context, cfgs ...WriterConfig) (w *streamWriter, err error) {
	cfg, err := config.New(DefaultWriterConfig, cfgs...)
	if err != nil {
		return nil, err
	}

	var (
		controlDigests *confluence.Stream[controller.Digest] = nil
		domainWriters  map[ChannelKey]*idxWriter
		rateWriters    map[telem.Rate]*idxWriter
	)
	if len(cfg.Channels) > 1 && cfg.SendControlDigests {
		controlDigests = confluence.NewStream[controller.Digest](len(cfg.Channels))
	}

	defer func() {
		if err == nil {
			return
		}
		for _, idx := range domainWriters {
			err = errors.CombineErrors(idx.Close(), err)
		}
		for _, idx := range rateWriters {
			err = errors.CombineErrors(idx.Close(), err)
		}
	}()

	for i, key := range cfg.Channels {
		u, ok := db.dbs[key]
		if !ok {
			return nil, ChannelNotFound
		}

		w, err := u.OpenWriter(ctx, unary.WriterConfig{
			Start:          cfg.Start,
			Authority:      cfg.authority(i),
			ControlDigests: controlDigests,
		})
		if err != nil {
			return nil, err
		}

		if u.Channel.Index != 0 {

			// Hot path optimization: in the common case we only write to a rate based
			// index or a domain indexed channel, not both. In either case we can avoid a
			// map allocation.
			if domainWriters == nil {
				domainWriters = make(map[ChannelKey]*idxWriter)
			}

			idxW, exists := domainWriters[u.Channel.Index]
			if !exists {
				idxW, err = db.openDomainIdxWriter(u.Channel.Index, cfg)
				if err != nil {
					return nil, err
				}
				idxW.writingToIdx = u.Channel.IsIndex
				domainWriters[u.Channel.Index] = idxW
			} else if u.Channel.IsIndex {
				idxW.writingToIdx = true
				domainWriters[u.Channel.Index] = idxW
			}

			idxW.internal[key] = &unaryWriterState{Writer: *w}
		} else {

			// Hot path optimization: in the common case we only write to a rate based
			// index or an indexed channel, not both. In either case we can avoid a
			// map allocation.
			if rateWriters == nil {
				rateWriters = make(map[telem.Rate]*idxWriter)
			}

			idxW, ok := rateWriters[u.Channel.Rate]
			if !ok {
				idxW = db.openRateIdxWriter(u.Channel.Rate, cfg)
				rateWriters[u.Channel.Rate] = idxW
			}

			idxW.internal[key] = &unaryWriterState{Writer: *w}
		}
	}

	w = &streamWriter{
		internal:       make([]*idxWriter, 0, len(domainWriters)+len(rateWriters)),
		relay:          db.relay.inlet,
		controlDigests: controlDigests,
	}
	for _, idx := range domainWriters {
		w.internal = append(w.internal, idx)
	}
	for _, idx := range rateWriters {
		w.internal = append(w.internal, idx)
	}
	return w, nil
}

func (db *DB) openDomainIdxWriter(
	chKey ChannelKey,
	cfg WriterConfig,
) (*idxWriter, error) {
	u, err := db.getUnary(chKey)
	if err != nil {
		return nil, err
	}
	idx := &index.Domain{DB: u.Domain, Instrumentation: db.Instrumentation}
	w := &idxWriter{internal: make(map[ChannelKey]*unaryWriterState)}
	w.idx.key = chKey
	w.idx.Index = idx
	w.idx.highWaterMark = cfg.Start
	w.writingToIdx = false
	w.start = cfg.Start
	return w, nil
}

func (db *DB) openRateIdxWriter(
	rate telem.Rate,
	cfg WriterConfig,
) *idxWriter {
	idx := index.Rate{Rate: rate}
	w := &idxWriter{internal: make(map[ChannelKey]*unaryWriterState)}
	w.idx.Index = idx
	w.start = cfg.Start
	return w
}
