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
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// CreateChannel implements DB.
func (db *cesium) CreateChannel(ch ...Channel) error {
	for _, c := range ch {
		if err := db.createChannel(c); err != nil {
			return err
		}
	}
	return nil
}

// RetrieveChannels implements DB.
func (db *cesium) RetrieveChannels(keys ...string) ([]Channel, error) {
	chs := make([]Channel, 0, len(keys))
	for _, key := range keys {
		ch, err := db.RetrieveChannel(key)
		if err != nil {
			return nil, err
		}
		chs = append(chs, ch)
	}
	return chs, nil
}

// RetrieveChannel implements DB.
func (db *cesium) RetrieveChannel(key string) (Channel, error) {
	ch, ok := db.dbs[key]
	if !ok {
		return Channel{}, ChannelNotFound
	}
	return ch.Channel, nil
}

func (db *cesium) createChannel(ch Channel) (err error) {
	defer func() {
		db.logger.Sugar().Infow("creating channel",
			"key", ch.Key,
			"index", ch.Index,
			"rate", ch.Rate,
			"datatype", ch.DataType,
			"isIndex", ch.IsIndex,
			"error", err,
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

func (db *cesium) validateNewChannel(ch Channel) error {
	v := validate.New("cesium")
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
