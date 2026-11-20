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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// CreateChannel creates a channel in the database.
func (db *DB) CreateChannel(ctx context.Context, ch ...Channel) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, c := range ch {
		if err := db.createChannel(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

// RetrieveChannels retrieves the channels by the specified keys. It is atomic and will
// either return all the channels or no channels if there is an error.
func (db *DB) RetrieveChannels(
	ctx context.Context,
	keys ...ChannelKey,
) ([]Channel, error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	chs := make([]Channel, 0, len(keys))
	for _, key := range keys {
		ch, err := db.retrieveChannel(key)
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
	return db.retrieveChannel(key)
}

// retrieveChannel retrieves a channel from the database. This method is not safe for
// concurrent use, and the db must be locked before calling.
func (db *DB) retrieveChannel(key ChannelKey) (Channel, error) {
	if u, ok := db.mu.dbs.unary[key]; ok {
		return u.Channel(), nil
	}
	return Channel{}, channel.NewNotFoundError(key)
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
	err = db.openChannel(ctx, ch)
	return err
}

func indexChannelNotFoundError(key ChannelKey) error {
	return errors.Wrapf(
		query.ErrNotFound,
		"index channel with key %d does not exist",
		key,
	)
}

func (db *DB) validateNewChannel(ch Channel) error {
	if err := ch.Validate(); err != nil {
		return err
	}
	_, unaryExists := db.mu.dbs.unary[ch.Key]
	_, hiddenExists := db.mu.dbs.hidden[ch.Key]
	if unaryExists || hiddenExists || db.mu.virtualDirs.Contains(ch.Key) {
		return channel.Wrap(ch.Key, errors.Wrap(
			validate.ErrValidation,
			"cannot create the channel because it already exists",
		))
	}
	if ch.Virtual {
		return channel.Wrap(ch.Key, errors.Wrap(
			validate.ErrValidation,
			"cannot create a virtual channel, which has no storage",
		))
	}
	if ch.Index != 0 && !ch.IsIndex {
		indexDB, ok := db.mu.dbs.unary[ch.Index]
		if !ok {
			return validate.PathedError(indexChannelNotFoundError(ch.Index), "index")
		}
		if !indexDB.Channel().IsIndex {
			return validate.PathedError(
				channel.Wrap(
					indexDB.Channel().Key,
					errors.Wrap(validate.ErrValidation, "not an index"),
				),
				"index",
			)
		}
	}
	return nil
}
