// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/migrate"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
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
	fs fs.FS,
	ch channel.Channel,
	codec encoding.Codec,
) (channel.Channel, error) {
	exists, err := fs.Exists(metaFile)
	if err != nil {
		return channel.Channel{}, err
	}
	if exists {
		ch, err = Read(ctx, fs, codec)
		if err != nil {
			return channel.Channel{}, err
		}
		state := migrate.Migrate(migrate.DBState{Channel: ch, FS: fs})
		if state.ShouldIgnoreChannel {
			return channel.Channel{}, ErrIgnoreChannel
		}
		if state.Channel.Version != ch.Version {
			if err := Create(ctx, fs, codec, state.Channel); err != nil {
				return channel.Channel{}, err
			}
		}
		if err := state.Channel.Validate(); err != nil {
			return channel.Channel{}, err
		}
		return state.Channel, nil
	}
	if err := Create(ctx, fs, codec, ch); err != nil {
		return channel.Channel{}, err
	}
	return ch, nil
}

// ReadVirtualFlag reports whether the metadata file in fs was persisted for a virtual
// channel by a previous version of cesium. Virtual channels are no longer persisted and
// channel.Channel excludes the field from serialization, so the flag can only be read
// through this probe. Returns false if no metadata file exists or the file cannot be
// decoded; decode failures are reported with full context by the subsequent Open.
func ReadVirtualFlag(ctx context.Context, fs fs.FS, codec encoding.Decoder) (virtual bool, err error) {
	exists, err := fs.Exists(metaFile)
	if err != nil || !exists {
		return false, err
	}
	metaF, err := fs.Open(metaFile, os.O_RDONLY)
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Combine(err, metaF.Close()) }()
	var probe struct {
		Virtual bool `json:"virtual"`
	}
	if dErr := codec.DecodeStream(ctx, metaF, &probe); dErr != nil {
		return false, err
	}
	return probe.Virtual, err
}

// Read reads the metadata file for a database whose data is kept in fs and is encoded
// by the provided encoder.
func Read(ctx context.Context, fs fs.FS, codec encoding.Decoder) (ch channel.Channel, err error) {
	s, err := fs.Stat("")
	if err != nil {
		return channel.Channel{}, err
	}
	metaF, err := fs.Open(metaFile, os.O_RDONLY)
	if err != nil {
		return channel.Channel{}, err
	}
	defer func() { err = errors.Combine(err, metaF.Close()) }()

	if err = codec.DecodeStream(ctx, metaF, &ch); err != nil {
		err = errors.Wrapf(
			err, "error decoding meta in folder for channel %s", s.Name(),
		)
		return channel.Channel{}, err
	}
	return ch, nil
}

// Create creates the metadata file for a database whose data is kept in fs and is
// encoded by the provided encoder. The provided channel should have all fields required
// by the DB correctly set.
func Create(ctx context.Context, fs fs.FS, codec encoding.Encoder, ch channel.Channel) (err error) {
	if err = ch.Validate(); err != nil {
		return err
	}
	tempMetaF, err := fs.Open(
		metaTempFile,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Combine(err, fs.Remove(metaTempFile))
		}
	}()
	if err = codec.EncodeStream(ctx, tempMetaF, ch); err != nil {
		err = errors.Combine(err, tempMetaF.Close())
		return err
	}
	if err = tempMetaF.Close(); err != nil {
		return err
	}
	return fs.Rename(metaTempFile, metaFile)
}
