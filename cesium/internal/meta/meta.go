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
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
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
		return state.Channel, Validate(state.Channel)
	}
	return ch, Create(ctx, fs, codec, ch)
}

// Read reads the metadata file for a database whose data is kept in fs and is encoded
// by the provided encoder.
func Read(ctx context.Context, fs xfs.FS, codec binary.Codec) (ch core.Channel, err error) {
	s, err := fs.Stat("")
	if err != nil {
		return
	}
	metaF, err := fs.Open(metaFile, os.O_RDONLY)
	if err != nil {
		return
	}
	defer func() { err = errors.Combine(err, metaF.Close()) }()

	if err = codec.DecodeStream(ctx, metaF, &ch); err != nil {
		err = errors.Wrapf(err, "error decoding meta in folder for channel %s", s.Name())
	}
	return
}

// Create creates the metadata file for a database whose data is kept in fs and is
// encoded by the provided encoder. The provided channel should have all fields
// required by the DB correctly set.
func Create(ctx context.Context, fs xfs.FS, codec binary.Codec, ch core.Channel) error {
	if err := Validate(ch); err != nil {
		return err
	}
	metaF, err := fs.Open(metaFile, os.O_CREATE|os.O_WRONLY)
	if err != nil {
		return err
	}
	b, err := codec.Encode(ctx, ch)
	if err != nil {
		return err
	}
	if _, err = metaF.Write(b); err != nil {
		return err
	}
	return metaF.Close()
}

// Validate checks that the meta file read from or about to be written to a meta file
// is well-defined.
func Validate(ch core.Channel) error {
	v := validate.New("meta")
	validate.Positive(v, "key", ch.Key)
	validate.NotEmptyString(v, "data_type", ch.DataType)
	if ch.Virtual {
		v.Ternaryf("index", ch.Index != 0, "virtual channel cannot be indexed")
	} else {
		v.Ternary("data_type", ch.DataType == telem.StringT, "persisted channels cannot have string data types")
		if ch.IsIndex {
			v.Ternary("data_type", ch.DataType != telem.TimeStampT, "index channel must be of type timestamp")
			v.Ternaryf("index", ch.Index != 0 && ch.Index != ch.Key, "index channel cannot be indexed by another channel")
		} else {
			v.Ternaryf("index", ch.Index == 0, "non-indexed channel must have an index")
		}
	}
	return v.Error()
}
