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
	"github.com/samber/lo"
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
func (db *DB) RetrieveChannels(ctx context.Context, keys ...string) ([]Channel, error) {
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
func (db *DB) RetrieveChannel(_ context.Context, key string) (Channel, error) {
	ch, ok := db.dbs[key]
	if !ok {
		return Channel{}, ChannelNotFound
	}
	return ch.Channel, nil
}

func (db *DB) createChannel(ch Channel) (err error) {
	defer func() {
		lo.Ternary(err == nil, db.L.Info, db.L.Error)(
			"creating channel",
			zap.String("key", ch.Key),
			zap.String("index", ch.Index),
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
	err = db.openUnary(ch)
	return
}

func (db *DB) validateNewChannel(ch Channel) error {
	v := validate.New("DB")
	validate.NotEmptyString(v, "key", ch.Key)
	validate.NotEmptyString(v, "data type", ch.DataType)
	validate.MapDoesNotContainF(v, ch.Key, db.dbs, "channel %s already exists", ch.Key)
	if ch.IsIndex {
		v.Ternary(ch.DataType != telem.TimeStampT, "index channel must be of type timestamp")
		v.Ternaryf(ch.Index != "" && ch.Index != ch.Key, "index channel cannot be indexed by another channel")
	} else if ch.Index != "" {
		validate.MapContainsf(v, ch.Index, db.dbs, "index %s does not exist", ch.Index)
		v.Funcf(func() bool {
			return !db.dbs[ch.Index].Channel.IsIndex
		}, "channel %s is not an index", ch.Index)

	} else {
		validate.Positive(v, "rate", ch.Rate)
	}
	return v.Error()
}
