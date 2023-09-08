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
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/telem"
)

// NewStreamWriter implements DB.
func (db *DB) NewStreamWriter(ctx context.Context, cfg WriterConfig) (StreamWriter, error) {
	return db.newStreamWriter(ctx, cfg)
}

// OpenWriter implements DB.
func (db *DB) OpenWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	internal, err := db.newStreamWriter(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return wrapStreamWriter(internal), nil
}

func (db *DB) newStreamWriter(ctx context.Context, cfg WriterConfig) (w *streamWriter, err error) {
	var (
		domainWriters map[ChannelKey]*idxWriter
		rateWriters   map[telem.Rate]*idxWriter
	)

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

	for _, key := range cfg.Channels {
		u, ok := db.dbs[key]
		if !ok {
			return nil, ChannelNotFound
		}

		w, err := u.OpenWriter(ctx, domain.WriterConfig{Start: cfg.Start})
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
		internal: make([]*idxWriter, 0, len(domainWriters)+len(rateWriters)),
		relay:    db.relay.inlet,
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
	idx := &index.Domain{DB: u.Ranger, Instrumentation: db.Instrumentation}
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
