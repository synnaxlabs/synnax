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

// ErrIgnoreChannel lets callers know that this channel is no longer valid and should
// be ignored when opening a DB.
var ErrIgnoreChannel = errors.New("channel should be ignored")

// Open reads the metadata file for a database whose data is kept in fs and is
// encoded by the provided encoder. If the file does not exist, it will be created. If
// the file does exist, it will be read and returned. The provided channel should have
// all fields required by the DB correctly set.
func Open(ctx context.Context, fs xfs.FS, ch core.Channel, codec binary.Codec) (core.Channel, error) {
	exists, err := fs.Exists(metaFile)
	if err != nil {
		return ch, err
	}
	if exists {
		ch, err = Read(ctx, fs, codec)
		if err != nil {
			return ch, err
		}
		state := migrate.Migrate(migrate.DBState{Channel: ch, FS: fs})
		if state.ShouldIgnoreChannel {
			return ch, ErrIgnoreChannel
		}
		if state.Channel.Version != ch.Version {
			if err := Create(ctx, fs, codec, state.Channel); err != nil {
				return ch, err
			}
		}
		return state.Channel, state.Channel.Validate()
	}
	if err := Create(ctx, fs, codec, ch); err != nil {
		return core.Channel{}, err
	}
	return ch, nil
}

// Read reads the metadata file for a database whose data is kept in fs and is encoded
// by the provided encoder.
func Read(ctx context.Context, fs xfs.FS, codec binary.Codec) (core.Channel, error) {
	var ch core.Channel
	s, err := fs.Stat("")
	if err != nil {
		return ch, err
	}
	metaF, err := fs.Open(metaFile, os.O_RDONLY)
	if err != nil {
		return ch, err
	}
	defer func() { err = errors.Combine(err, metaF.Close()) }()

	if err = codec.DecodeStream(ctx, metaF, &ch); err != nil {
		err = errors.Wrapf(err, "error decoding meta in folder for channel %s", s.Name())
		return ch, err
	}
	return ch, err
}

// Create creates the metadata file for a database whose data is kept in fs and is
// encoded by the provided encoder. The provided channel should have all fields
// required by the DB correctly set.
func Create(ctx context.Context, fs xfs.FS, codec binary.Codec, ch core.Channel) error {
	if err := ch.Validate(); err != nil {
		return err
	}
	metaF, err := fs.Open(metaFile, os.O_CREATE|os.O_WRONLY)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, metaF.Close()) }()
	b, err := codec.Encode(ctx, ch)
	if err != nil {
		return err
	}
	_, err = metaF.Write(b)
	return err
}
