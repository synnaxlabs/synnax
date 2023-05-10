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
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/validate"
)

// NewStreamWriter implements DB.
func (db *DB) NewStreamWriter(ctx context.Context, cfg WriterConfig) (StreamWriter, error) {
	return db.newStreamWriter(ctx, cfg)
}

// NewWriter implements DB.
func (db *DB) NewWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	internal, err := db.newStreamWriter(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return wrapStreamWriter(internal), nil
}

func (db *DB) newStreamWriter(ctx context.Context, cfg WriterConfig) (*streamWriter, error) {
	var (
		idx          index.Index
		writingToIdx bool
		idxChannel   Channel
		internal     = make(map[core.ChannelKey]unary.Writer, len(cfg.Channels))
	)
	for i, key := range cfg.Channels {
		u, ok := db.dbs[key]
		if !ok {
			return nil, ChannelNotFound
		}
		if u.Channel.IsIndex {
			writingToIdx = true
		}
		if i == 0 {
			if u.Channel.Index != 0 {
				idxU, err := db.getUnary(u.Channel.Index)
				if err != nil {
					return nil, err
				}
				idx = &index.Domain{DB: idxU.Ranger, Instrumentation: db.Instrumentation}
				idxChannel = idxU.Channel
			} else {
				idx = index.Rate{Rate: u.Channel.Rate}
				idxChannel = u.Channel
			}
		} else {
			if err := validateSameIndex(u.Channel, idxChannel); err != nil {
				return nil, err
			}
		}
		w, err := u.NewWriter(ctx, domain.WriterConfig{Start: cfg.Start})
		if err != nil {
			return nil, err
		}
		internal[key] = *w
	}

	w := &streamWriter{internal: internal, relay: db.relay.inlet}
	w.Start = cfg.Start
	w.idx.key = idxChannel.Key
	w.writingToIdx = writingToIdx
	w.idx.highWaterMark = cfg.Start
	w.idx.Index = idx
	return w, nil
}

func validateSameIndex(chOne, chTwo Channel) error {
	if chOne.Index == 0 && chTwo.Index == 0 {
		if chOne.Rate != chTwo.Rate {
			return errors.Wrapf(validate.Error, "channels must have the same rate")
		}
	}
	if chOne.Index != chTwo.Index {
		return errors.Wrapf(validate.Error, "channels must have the same index")
	}
	return nil
}
