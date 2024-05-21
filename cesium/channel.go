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
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
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

// RekeyChannel changes the key of channel oldKey into newKey. This operation is
// idempotent and does not return an error if the channel does not exist.
// RekeyChannel returns an error if there are open iterators/writers on the given channel.
func (db *DB) RekeyChannel(oldKey ChannelKey, newKey core.ChannelKey) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, uok := db.unaryDBs[newKey]
	_, vok := db.virtualDBs[newKey]
	if uok || vok {
		return errors.Newf("cannot rekey to %d since it channel %d already exists", newKey, newKey)
	}

	udb, uok := db.unaryDBs[oldKey]
	if uok {
		if err := udb.TryClose(); err != nil {
			return err
		}
		if err := db.fs.Rename(keyToDirName(oldKey), keyToDirName(newKey)); err != nil {
			return err
		}

		newFS, err := db.fs.Sub(keyToDirName(newKey))
		if err != nil {
			return err
		}

		newConfig := udb.Config
		newConfig.FS = newFS
		newConfig.Channel.Key = newKey
		if newConfig.Channel.IsIndex {
			newConfig.Channel.Index = newKey
		}

		if err = meta.Create(newFS, db.metaECD, newConfig.Channel); err != nil {
			return err
		}

		_udb, err := unary.Open(newConfig)
		if err != nil {
			return err
		}

		delete(db.unaryDBs, oldKey)
		db.unaryDBs[newKey] = *_udb

		if udb.Channel.IsIndex {
			for otherDBKey := range db.unaryDBs {
				otherDB := db.unaryDBs[otherDBKey]
				if otherDB.Channel.Index == oldKey && otherDBKey != newKey {
					if err = otherDB.TryClose(); err != nil {
						return err
					}

					newFS, err = db.fs.Sub(keyToDirName(otherDBKey))
					if err != nil {
						return err
					}

					newConfig = otherDB.Config
					newConfig.Channel.Index = newKey

					if err = meta.Create(newFS, db.metaECD, newConfig.Channel); err != nil {
						return err
					}

					_otherDB, err := unary.Open(newConfig)
					if err != nil {
						return err
					}
					_otherDB.SetIndex((*_udb).Index())
					db.unaryDBs[otherDBKey] = *_otherDB
				}
			}
		}

		return nil
	}
	vdb, vok := db.virtualDBs[oldKey]
	if vok {
		if err := vdb.TryClose(); err != nil {
			return err
		}
		if err := db.fs.Rename(keyToDirName(oldKey), keyToDirName(newKey)); err != nil {
			return err
		}

		newFS, err := db.fs.Sub(keyToDirName(newKey))
		if err != nil {
			return err
		}

		newConfig := vdb.Config
		newConfig.Channel.Key = newKey

		if err = meta.Create(newFS, db.metaECD, newConfig.Channel); err != nil {
			return err
		}

		_vdb, err := virtual.Open(newConfig)
		if err != nil {
			return err
		}

		delete(db.virtualDBs, oldKey)
		db.virtualDBs[newKey] = *_vdb
	}

	return nil
}
