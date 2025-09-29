// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package meta

import (
	"context"
	"os"

	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/migrate"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

const metaFile = "meta.json"
const metaTempFile = "meta.json.tmp"

// ErrIgnoreChannel lets callers know that this channel is no longer valid and should be
// ignored when opening a DB.
var ErrIgnoreChannel = errors.New("channel should be ignored")

// Open reads the metadata file for a database whose data is kept in fs and is encoded
// by the provided encoder. If the file does not exist, it will be created. If the file
// does exist, it will be read and returned. The provided channel should have all fields
// required by the DB correctly set.
func Open(
	ctx context.Context,
	fs xfs.FS,
	ch core.Channel,
	codec binary.Codec,
) (core.Channel, error) {
	exists, err := fs.Exists(metaFile)
	if err != nil {
		return core.Channel{}, err
	}
	if exists {
		ch, err = Read(ctx, fs, codec)
		if err != nil {
			return core.Channel{}, err
		}
		state := migrate.Migrate(migrate.DBState{Channel: ch, FS: fs})
		if state.ShouldIgnoreChannel {
			return core.Channel{}, ErrIgnoreChannel
		}
		if state.Channel.Version != ch.Version {
			if err := Create(ctx, fs, codec, state.Channel); err != nil {
				return core.Channel{}, err
			}
		}
		if err := state.Channel.Validate(); err != nil {
			return core.Channel{}, err
		}
		return state.Channel, nil
	}
	if err := Create(ctx, fs, codec, ch); err != nil {
		return core.Channel{}, err
	}
	return ch, nil
}

// Read reads the metadata file for a database whose data is kept in fs and is encoded
// by the provided encoder.
func Read(ctx context.Context, fs xfs.FS, codec binary.Decoder) (core.Channel, error) {
	s, err := fs.Stat("")
	if err != nil {
		return core.Channel{}, err
	}
	metaF, err := fs.Open(metaFile, os.O_RDONLY)
	if err != nil {
		return core.Channel{}, err
	}
	defer func() { err = errors.Combine(err, metaF.Close()) }()

	var ch core.Channel
	if err = codec.DecodeStream(ctx, metaF, &ch); err != nil {
		err = errors.Wrapf(
			err, "error decoding meta in folder for channel %s", s.Name(),
		)
		return core.Channel{}, err
	}
	return ch, nil
}

// Create creates the metadata file for a database whose data is kept in fs and is
// encoded by the provided encoder. The provided channel should have all fields required
// by the DB correctly set.
func Create(ctx context.Context, fs xfs.FS, codec binary.Encoder, ch core.Channel) error {
	if err := ch.Validate(); err != nil {
		return err
	}
	tempMetaF, err := fs.Open(
		metaTempFile,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
	)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, fs.Remove(metaTempFile)) }()
	if err = codec.EncodeStream(ctx, tempMetaF, ch); err != nil {
		err = errors.Combine(err, tempMetaF.Close())
		return err
	}
	if err = tempMetaF.Close(); err != nil {
		return err
	}
	if err = fs.Rename(metaTempFile, metaFile); err != nil {
		return err
	}
	err = fs.Remove(metaTempFile)
	return err
}
