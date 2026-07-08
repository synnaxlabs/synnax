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

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// CreateChannel creates the given channels in the database. Index channels are created
// before the other channels in the batch, so a channel may reference an index defined
// anywhere within the same call.
func (db *DB) CreateChannel(ctx context.Context, channels ...Channel) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, ch := range channels {
		if ch.IsIndex {
			if err := db.createChannel(ctx, ch); err != nil {
				return err
			}
		}
	}
	for _, ch := range channels {
		if !ch.IsIndex {
			if err := db.createChannel(ctx, ch); err != nil {
				return err
			}
		}
	}
	return nil
}

// RetrieveChannels retrieves the channels by the specified keys. It is atomic and will
// either return all the channels or no channels if there is an error.
func (db *DB) RetrieveChannels(ctx context.Context, keys ...ChannelKey) ([]Channel, error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
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
		return Channel{}, ErrDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.retrieveChannel(ctx, key)
}

// retrieveChannel retrieves a channel from the database. This method is not safe for
// concurrent use, and the db must be locked before calling.
func (db *DB) retrieveChannel(_ context.Context, key ChannelKey) (Channel, error) {
	if u, ok := db.mu.dbs.unary[key]; ok {
		return u.Channel(), nil
	}
	if v, ok := db.mu.dbs.virtual[key]; ok {
		return v.Channel(), nil
	}
	return Channel{}, channel.NewNotFoundError(key)
}

// RenameChannels finds the specified keys in the database and renames them to the new
// name as specified in names.
func (db *DB) RenameChannels(ctx context.Context, keys []ChannelKey, names []string) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if len(keys) != len(names) {
		return errors.Wrapf(validate.ErrValidation, "keys and names must have the same length")
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
		return ErrDBClosed
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.renameChannel(ctx, key, newName)
}

// RenameChannel renames the channel with the specified key to newName. There is a
// race condition here: one could rename a channel while it is being read or streamed
// from or written to. We choose to not address this since the name is purely
// decorative in Cesium and not used to identify channels whereas the key is the
// unique identifier. The same goes for the virtual database.
func (db *DB) renameChannel(ctx context.Context, key ChannelKey, newName string) error {
	if u, ok := db.mu.dbs.unary[key]; ok {
		if err := u.RenameChannelInMeta(ctx, newName); err != nil {
			return err
		}
		db.mu.dbs.unary[key] = u
		return nil
	}
	if v, ok := db.mu.dbs.virtual[key]; ok {
		if err := v.RenameChannel(newName); err != nil {
			return err
		}
		db.mu.dbs.virtual[key] = v
		return nil
	}
	return channel.NewNotFoundError(key)
}

func (db *DB) createChannel(ctx context.Context, ch Channel) (err error) {
	defer func() {
		lo.Ternary(err == nil, db.L.Debug, db.L.Error)(
			"creating channel",
			zap.Uint32("key", ch.Key),
			zap.Uint32("index", ch.Index),
			zap.String("data_type", string(ch.DataType)),
			zap.Bool("isIndex", ch.IsIndex),
			zap.Error(err),
		)
	}()

	if err = db.validateNewChannel(ch); err != nil {
		return err
	}
	if ch.IsIndex {
		ch.Index = ch.Key
	}
	ch.Version = channel.VersionCurrent
	if ch.Virtual {
		return db.openVirtual(ch)
	}
	fs, err := db.fs.Sub(keyToDirName(ch.Key))
	if err != nil {
		return err
	}
	if ch, err = meta.Open(ctx, fs, ch, db.metaCodec); err != nil {
		return err
	}
	return db.openUnary(ctx, ch, fs)
}

func indexChannelNotFoundError(key ChannelKey) error {
	return errors.Wrapf(query.ErrNotFound, "index channel with key %d does not exist", key)
}

func (db *DB) validateNewChannel(ch Channel) error {
	if err := ch.Validate(); err != nil {
		return err
	}
	_, unaryExists := db.mu.dbs.unary[ch.Key]
	_, virtualExists := db.mu.dbs.virtual[ch.Key]
	if unaryExists || virtualExists {
		return errors.Wrapf(validate.ErrValidation, "cannot create channel %v because it already exists", ch)
	}
	if ch.Virtual {
		if ch.Index != 0 && !ch.IsIndex {
			indexDB, ok := db.mu.dbs.virtual[ch.Index]
			if !ok {
				if _, unaryOk := db.mu.dbs.unary[ch.Index]; unaryOk {
					return validate.PathedError(
						errors.Wrapf(validate.ErrValidation, "virtual channel cannot be indexed by stored channel %d", ch.Index),
						"index",
					)
				}
				return validate.PathedError(indexChannelNotFoundError(ch.Index), "index")
			}
			if !indexDB.Channel().IsIndex {
				return validate.PathedError(
					errors.Wrapf(validate.ErrValidation, "channel %v is not an index", indexDB.Channel()),
					"index",
				)
			}
		}
		return nil
	}
	if ch.Index != 0 && !ch.IsIndex {
		indexDB, ok := db.mu.dbs.unary[ch.Index]
		if !ok {
			if _, virtualOk := db.mu.dbs.virtual[ch.Index]; virtualOk {
				return validate.PathedError(
					errors.Wrapf(validate.ErrValidation, "stored channel cannot be indexed by virtual channel %d", ch.Index),
					"index",
				)
			}
			return validate.PathedError(indexChannelNotFoundError(ch.Index), "index")
		}
		if !indexDB.Channel().IsIndex {
			return validate.PathedError(
				errors.Wrapf(validate.ErrValidation, "channel %v is not an index", indexDB.Channel()),
				"index",
			)
		}
	}
	return nil
}

// RekeyChannel changes the key of channel oldKey into newKey. This operation is
// idempotent and does not return an error if the channel does not exist. RekeyChannel
// returns an error if there are open iterators/writers on the given channel.
func (db *DB) RekeyChannel(ctx context.Context, oldKey ChannelKey, newKey channel.Key) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, ok := db.mu.dbs.unary[newKey]; ok {
		return errors.Newf(
			"cannot rekey channel to %d since a channel with the same key already exists in the database",
			newKey,
		)
	}
	if _, ok := db.mu.dbs.virtual[newKey]; ok {
		return errors.Newf(
			"cannot rekey channel to %d since a channel with the same key already exists in the database",
			newKey,
		)
	}

	oldDir := keyToDirName(oldKey)
	newDir := keyToDirName(newKey)
	if u, ok := db.mu.dbs.unary[oldKey]; ok {
		if err := u.Close(); err != nil {
			return err
		}
		if err := db.fs.Rename(oldDir, newDir); err != nil {
			return err
		}
		newFS, err := db.fs.Sub(keyToDirName(newKey))
		if err != nil {
			return err
		}
		newCh := u.Channel()
		newCh.Key = newKey
		if newCh.IsIndex {
			newCh.Index = newKey
		}
		newDB, err := unary.Open(ctx, unary.Config{
			Instrumentation: db.Instrumentation,
			MetaCodec:       db.metaCodec,
			Channel:         newCh,
			FS:              newFS,
		})
		if err != nil {
			return err
		}
		if err = newDB.SetChannelKeyInMeta(ctx, newKey); err != nil {
			return err
		}
		delete(db.mu.dbs.unary, oldKey)
		db.mu.dbs.unary[newKey] = *newDB

		// If the DB is an index channel, update every unary DB that referenced the
		// old key as its index.
		if u.Channel().IsIndex {
			for otherDBKey := range db.mu.dbs.unary {
				otherDB := db.mu.dbs.unary[otherDBKey]
				if otherDB.Channel().Index == oldKey && otherDBKey != newKey {
					if err = otherDB.SetIndexKeyInMeta(ctx, newKey); err != nil {
						return err
					}
					otherDB.SetIndex((*newDB).Index())
					db.mu.dbs.unary[otherDBKey] = otherDB
				}
			}
		}
		return nil
	}
	if vDB, ok := db.mu.dbs.virtual[oldKey]; ok {
		if err := vDB.Close(); err != nil {
			return err
		}
		newChannel := vDB.Channel()
		newChannel.Key = newKey
		newDB, err := virtual.Open(virtual.Config{
			Instrumentation: db.Instrumentation,
			Channel:         newChannel,
		})
		if err != nil {
			return err
		}
		delete(db.mu.dbs.virtual, oldKey)
		db.mu.dbs.virtual[newKey] = *newDB
	}

	return nil
}
