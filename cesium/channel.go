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
	"os"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
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

// FindChannels returns the channels of keys that the database holds, in the order of
// keys, and leaves out the others.
func (db *DB) FindChannels(ctx context.Context, keys ...ChannelKey) ([]Channel, error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	var chs []Channel
	for _, key := range keys {
		if ch, ok := db.channelOf(key); ok {
			if chs == nil {
				chs = make([]Channel, 0, len(keys))
			}
			chs = append(chs, ch)
		}
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
	if ch, ok := db.channelOf(key); ok {
		return ch, nil
	}
	return Channel{}, channel.NewNotFoundError(key)
}

// channelOf returns the channel of key, and false when the DB holds none. The caller
// holds db.mu.
func (db *DB) channelOf(key ChannelKey) (Channel, bool) {
	if db.engine != nil {
		ch, ok := db.mu.channels[key]
		return ch, ok
	}
	u, ok := db.mu.dbs.unary[key]
	if !ok {
		return Channel{}, false
	}
	return u.Channel(), true
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
	if db.engine != nil {
		return db.createLogChannel(ch)
	}
	err = db.openChannel(ctx, ch)
	return err
}

// createLogChannel keeps ch in the channel list of the pointer log layout: in memory,
// and durable in the metadata file of its directory.
func (db *DB) createLogChannel(ch Channel) error {
	data, err := db.metaCodec.Encode(ch)
	if err != nil {
		return err
	}
	dir := strconv.FormatUint(uint64(ch.Key), 10)
	sub, err := db.fs.Sub(dir)
	if err != nil {
		return err
	}
	f, err := sub.Open(meta.File, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		return err
	}
	if _, err = f.WriteAt(data, 0); err == nil {
		err = f.Sync()
	}
	if err = errors.Join(err, f.Close()); err != nil {
		return err
	}
	if err = db.fs.SyncDir(dir); err != nil {
		return err
	}
	if err = db.fs.SyncDir(""); err != nil {
		return err
	}
	db.mu.channels[ch.Key] = ch
	return nil
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
	_, exists := db.channelOf(ch.Key)
	_, hiddenExists := db.mu.dbs.hidden[ch.Key]
	if exists || hiddenExists || db.mu.virtualDirs.Contains(ch.Key) {
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
		index, ok := db.channelOf(ch.Index)
		if !ok {
			return validate.PathedError(indexChannelNotFoundError(ch.Index), "index")
		}
		if !index.IsIndex {
			return validate.PathedError(
				channel.Wrap(
					index.Key,
					errors.Wrap(validate.ErrValidation, "not an index"),
				),
				"index",
			)
		}
	}
	return nil
}

// loadChannels reads the channel list of the pointer log layout from the metadata file
// of each channel directory.
func loadChannels(ctx context.Context, o *options) (map[ChannelKey]Channel, error) {
	infos, err := o.fs.List("")
	if err != nil {
		return nil, err
	}
	channels := make(map[ChannelKey]Channel)
	for _, info := range infos {
		name := info.Name()
		if _, err = strconv.ParseUint(name, 10, 32); err != nil || !info.IsDir() {
			continue
		}
		if ok, err := o.fs.Exists(name + "/" + meta.File); err != nil {
			return nil, err
		} else if !ok {
			continue
		}
		sub, err := o.fs.Sub(name)
		if err != nil {
			return nil, err
		}
		ch, err := meta.Read(ctx, sub, o.metaCodec)
		if err != nil {
			return nil, err
		}
		channels[ch.Key] = ch
	}
	return channels, nil
}
