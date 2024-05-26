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
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// CreateChannel implements DB.
func (db *DB) CreateChannel(_ context.Context, ch ...Channel) error {
	if db.closed.Load() {
		return ErrDBClosed
	}

	for _, c := range ch {
		if err := db.createChannel(c); err != nil {
			return err
		}
	}
	return nil
}

// RetrieveChannels implements DB.
func (db *DB) RetrieveChannels(ctx context.Context, keys ...ChannelKey) ([]Channel, error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}

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
	if db.closed.Load() {
		return Channel{}, ErrDBClosed
	}

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
	return Channel{}, core.NewErrChannelNotFound(key)
}

// RenameChannel renames the channel with the specified key to newName.
func (db *DB) RenameChannel(_ context.Context, key ChannelKey, newName string) error {
	if db.closed.Load() {
		return ErrDBClosed
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	udb, uok := db.unaryDBs[key]
	if uok {
		// There is a race condition here: one could rename a channel while it is being
		// read or  streamed from or written to. We choose to not address this since
		// the name is purely decorative in Cesium and not used to identify channels
		// whereas the key is the unique identifier. The same goes for the virtual database.
		if udb.Channel.Name == newName {
			return nil
		}
		udb.Channel.Name = newName
		err := meta.Create(udb.FS, db.metaECD, udb.Channel)
		if err != nil {
			return err
		}
		db.unaryDBs[key] = udb
		return nil
	}
	vdb, vok := db.virtualDBs[key]
	if vok {
		if vdb.Channel.Name == newName {
			return nil
		}
		vdb.Channel.Name = newName
		err := meta.Create(vdb.FS, db.metaECD, vdb.Channel)
		if err != nil {
			return err
		}
		db.virtualDBs[key] = vdb
		return nil
	}

	return core.NewErrChannelNotFound(key)
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
	validate.NotEmptyString(v, "data_type", ch.DataType)
	v.Exec(func() error {
		db.mu.RLock()
		defer db.mu.RUnlock()
		_, uOk := db.unaryDBs[ch.Key]
		_, vOk := db.virtualDBs[ch.Key]
		if uOk || vOk {
			return errors.Wrapf(validate.Error, "cannot create channel [%s]<%d> because it already exists", ch.Name, ch.Key)
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
			v.Funcf(func() bool {
				return !db.unaryDBs[ch.Index].Channel.IsIndex
			}, "channel [%s]<%d> is not an index", db.unaryDBs[ch.Index].Channel.Name, db.unaryDBs[ch.Index].Channel.Key)
		} else {
			validate.Positive(v, "rate", ch.Rate)
		}
	}
	return v.Error()
}
