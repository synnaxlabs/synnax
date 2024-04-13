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
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// CreateChannel implements DB.
func (db *DB) CreateChannel(_ context.Context, ch ...Channel) error {
	for _, c := range ch {
		if err := db.createChannel(c); err != nil {
			return err
		}
	}
	return nil
}

// RetrieveChannels implements DB.
func (db *DB) RetrieveChannels(ctx context.Context, keys ...ChannelKey) ([]Channel, error) {
	chs := make([]Channel, 0, len(keys))
	for _, key := range keys {
		ch, err := db.RetrieveChannel(ctx, key)
		if err != nil {
			return nil, err
		}
		chs = append(chs, ch)
	}
	return chs, nil
}

// RetrieveChannel implements DB.
func (db *DB) RetrieveChannel(_ context.Context, key ChannelKey) (Channel, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	uCh, uOk := db.unaryDBs[key]
	if uOk {
		return uCh.Channel, nil
	}
	vCh, vOk := db.virtualDBs[key]
	if vOk {
		return vCh.Channel, nil
	}
	return Channel{}, core.ChannelNotFound
}

func (db *DB) createChannel(ch Channel) (err error) {
	defer func() {
		lo.Ternary(err == nil, db.L.Info, db.L.Error)(
			"creating channel",
			zap.Uint32("key", ch.Key),
			zap.Uint32("index", ch.Index),
			zap.Float64("rate", float64(ch.Rate)),
			zap.String("datatype", string(ch.DataType)),
			zap.Bool("isIndex", ch.IsIndex),
			zap.Error(err),
		)
	}()

	if err = db.validateNewChannel(ch); err != nil {
		return
	}
	if ch.IsIndex {
		ch.Index = ch.Key
	}
	err = db.openVirtualOrUnary(ch)
	return
}

func (db *DB) validateNewChannel(ch Channel) error {
	v := validate.New("cesium")
	validate.Positive(v, "key", ch.Key)
	validate.NotEmptyString(v, "data type", ch.DataType)
	v.Exec(func() error {
		_, uOk := db.unaryDBs[ch.Key]
		_, vOk := db.virtualDBs[ch.Key]
		if uOk || vOk {
			return errors.Wrapf(validate.Error, "[cesium] - channel %d already exists", ch.Key)
		}
		return nil
	})
	if ch.Virtual {
		v.Ternaryf(ch.Index != 0, "virtual channel cannot be indexed")
		v.Ternaryf(ch.Rate != 0, "virtual channel cannot have a rate")
	} else {
		v.Ternary(ch.DataType == telem.StringT, "persisted channels cannot have string data types")
		if ch.IsIndex {
			v.Ternary(ch.DataType != telem.TimeStampT, "index channel must be of type timestamp")
			v.Ternaryf(ch.Index != 0 && ch.Index != ch.Key, "index channel cannot be indexed by another channel")
		} else if ch.Index != 0 {
			validate.MapContainsf(v, ch.Index, db.unaryDBs, "index %v does not exist", ch.Index)
			v.Funcf(func() bool {
				return !db.unaryDBs[ch.Index].Channel.IsIndex
			}, "channel %v is not an index", ch.Index)
		} else {
			validate.Positive(v, "rate", ch.Rate)
		}
	}
	return v.Error()
}
