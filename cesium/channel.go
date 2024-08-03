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
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/version"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// CreateChannel creates a channel in the database.
func (db *DB) CreateChannel(_ context.Context, ch ...Channel) error {
	if db.closed.Load() {
		return errDBClosed
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, c := range ch {
		if err := db.createChannel(c); err != nil {
			return err
		}
	}
	return nil
}

// RetrieveChannels retrieves the channels by the specified keys. It is atomic and will
// either return all the channels or no channels if there is an error.
func (db *DB) RetrieveChannels(ctx context.Context, keys ...ChannelKey) ([]Channel, error) {
	if db.closed.Load() {
		return nil, errDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	chs := make([]Channel, 0, len(keys))
	for _, key := range keys {
		ch, err := db.retrieveChannel(ctx, key)
		if err != nil {
			return nil, err
		}
		chs = append(chs, ch)
	}
	return chs, nil
}

// RetrieveChannel retrieves one channel from the database.
func (db *DB) RetrieveChannel(ctx context.Context, key ChannelKey) (Channel, error) {
	if db.closed.Load() {
		return Channel{}, errDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.retrieveChannel(ctx, key)
}

// retrieveChannel retrieves a channel from the database. This method is not safe
// for concurrent use, and the db must be locked before calling.
func (db *DB) retrieveChannel(_ context.Context, key ChannelKey) (Channel, error) {
	uCh, uOk := db.unaryDBs[key]
	if uOk {
		return uCh.Channel(), nil
	}
	vCh, vOk := db.virtualDBs[key]
	if vOk {
		return vCh.Channel, nil
	}
	return Channel{}, core.NewErrChannelNotFound(key)
}

// RenameChannels finds the specified keys in the database and renames them to the new
// name as specified in names.
func (db *DB) RenameChannels(ctx context.Context, keys []ChannelKey, names []string) error {
	if db.closed.Load() {
		return errDBClosed
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if len(keys) != len(names) {
		return errors.Wrapf(validate.Error, "keys and names must have the same length")
	}
	for i := range keys {
		if err := db.renameChannel(ctx, keys[i], names[i]); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) RenameChannel(ctx context.Context, key ChannelKey, newName string) error {
	if db.closed.Load() {
		return errDBClosed
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.renameChannel(ctx, key, newName)
}

// RenameChannel renames the channel with the specified key to newName.
func (db *DB) renameChannel(_ context.Context, key ChannelKey, newName string) error {
	udb, uok := db.unaryDBs[key]
	if uok {
		// There is a race condition here: one could rename a channel while it is being
		// read or streamed from or written to. We choose to not address this since
		// the name is purely decorative in Cesium and not used to identify channels
		// whereas the key is the unique identifier. The same goes for the virtual database.
		if err := udb.RenameChannelInMeta(newName); err != nil {
			return err
		}
		db.unaryDBs[key] = udb
		return nil
	}
	vdb, vok := db.virtualDBs[key]
	if vok {
		if err := vdb.RenameChannel(newName); err != nil {
			return err
		}
		db.virtualDBs[key] = vdb
		return nil
	}

	return core.NewErrChannelNotFound(key)
}

func (db *DB) createChannel(ch Channel) (err error) {
	defer func() {
		lo.Ternary(err == nil, db.L.Debug, db.L.Error)(
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
	ch.Version = version.Current
	err = db.openVirtualOrUnary(ch)
	return
}

func (db *DB) validateNewChannel(ch Channel) error {
	v := validate.New("cesium")
	validate.Positive(v, "key", ch.Key)
	validate.NotEmptyString(v, "data_type", ch.DataType)
	v.Exec(func() error {
		_, uOk := db.unaryDBs[ch.Key]
		_, vOk := db.virtualDBs[ch.Key]
		if uOk || vOk {
			return errors.Wrapf(validate.Error, "cannot create channel %v because it already exists", ch)
		}
		return nil
	})
	if ch.Virtual {
		v.Ternaryf("index", ch.Index != 0, "virtual channel cannot be indexed")
		v.Ternaryf("index", ch.Rate != 0, "virtual channel cannot have a rate")
	} else {
		v.Ternary("index", ch.DataType == telem.StringT, "persisted channels cannot have string data types")
		if ch.IsIndex {
			v.Ternary("data_type", ch.DataType != telem.TimeStampT, "index channel must be of type timestamp")
			v.Ternaryf("index", ch.Index != 0 && ch.Index != ch.Key, "index channel cannot be indexed by another channel")
		} else if ch.Index != 0 {
			validate.MapContainsf(v, ch.Index, db.unaryDBs, "index channel <%d> does not exist", ch.Index)
			indexDB, ok := db.unaryDBs[ch.Index]
			if ok {
				v.Ternaryf("index", !indexDB.Channel().IsIndex, "channel %v is not an index", indexDB.Channel())
			}
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
		return errors.Newf(
			"cannot rekey channel to %d since a channel with the same key already exists in the database",
			newKey,
		)
	}

	oldDir := keyToDirName(oldKey)
	newDir := keyToDirName(newKey)
	udb, uok := db.unaryDBs[oldKey]
	if uok {
		if err := udb.Close(); err != nil {
			return err
		}
		if err := db.fs.Rename(oldDir, newDir); err != nil {
			return err
		}
		newFS, err := db.fs.Sub(keyToDirName(newKey))
		if err != nil {
			return err
		}
		newCh := udb.Channel()
		newCh.Key = newKey
		if newCh.IsIndex {
			newCh.Index = newKey
		}
		_udb, err := unary.Open(unary.Config{
			Instrumentation: db.Instrumentation,
			MetaCodec:       db.metaCodec,
			Channel:         newCh,
			FS:              newFS,
		})
		if err != nil {
			return err
		}
		if err = _udb.SetChannelKeyInMeta(newKey); err != nil {
			return err
		}
		delete(db.unaryDBs, oldKey)
		db.unaryDBs[newKey] = *_udb

		// If the DB is an index channel, we need to update the databases that depend
		// on this channel.
		if udb.Channel().IsIndex {
			for otherDBKey := range db.unaryDBs {
				otherDB := db.unaryDBs[otherDBKey]
				// If the other database uses this channel as its index, and it's not
				// the index itself.
				if otherDB.Channel().Index == oldKey && otherDBKey != newKey {
					if err = otherDB.SetIndexKeyInMeta(newKey); err != nil {
						return err
					}
					otherDB.SetIndex((*_udb).Index())
					db.unaryDBs[otherDBKey] = otherDB
				}
			}
		}
		return nil
	}
	vdb, vok := db.virtualDBs[oldKey]
	if vok {
		if err := vdb.Close(); err != nil {
			return err
		}
		if err := db.fs.Rename(oldDir, newDir); err != nil {
			return err
		}
		newFS, err := db.fs.Sub(keyToDirName(newKey))
		if err != nil {
			return err
		}
		newConfig := vdb.Config
		newConfig.Channel.Key = newKey
		newConfig.FS = newFS
		_vdb, err := virtual.Open(newConfig)
		if err != nil {
			return err
		}
		if err = _vdb.SetChannelKeyInMeta(newKey); err != nil {
			return err
		}
		delete(db.virtualDBs, oldKey)
		db.virtualDBs[newKey] = *_vdb
	}

	return nil
}
