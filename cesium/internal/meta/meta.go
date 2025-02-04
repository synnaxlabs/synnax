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
	"os"

	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const metaFile = "meta.json"

// ReadOrCreate reads the metadata file for a database whose data is kept in fs and is
// encoded by the provided encoder. If the file does not exist, it will be created. If
// the file does exist, it will be read and returned. The provided channel should have
// all fields required by the DB correctly set.
func ReadOrCreate(fs xfs.FS, ch core.Channel, codec binary.Codec) (core.Channel, error) {
	exists, err := fs.Exists(metaFile)
	if err != nil {
		return ch, err
	}
	if exists {
		ch, err = Read(fs, codec)
		if err != nil {
			return ch, err
		}
		return ch, validateMeta(ch)
	}

	return ch, Create(fs, codec, ch)
}

// Read reads the metadata file for a database whose data is kept in fs and is encoded
// by the provided encoder.
func Read(fs xfs.FS, codec binary.Codec) (ch core.Channel, err error) {
	s, err := fs.Stat("")
	if err != nil {
		return
	}
	metaF, err := fs.Open(metaFile, os.O_RDONLY)
	if err != nil {
		return
	}
	defer func() { err = errors.Combine(err, metaF.Close()) }()

	err = codec.DecodeStream(nil, metaF, &ch)
	if err != nil {
		err = errors.Wrapf(err, "error decoding meta in folder for channel %s", s.Name())
	}

	return
}

// Create creates the metadata file for a database whose data is kept in fs and is
// encoded by the provided encoder. The provided channel should have all fields
// required by the DB correctly set.
func Create(fs xfs.FS, codec binary.Codec, ch core.Channel) error {
	err := validateMeta(ch)
	if err != nil {
		return err
	}

	metaF, err := fs.Open(metaFile, os.O_CREATE|os.O_WRONLY)
	if err != nil {
		return err
	}
	b, err := codec.Encode(nil, ch)
	if err != nil {
		return err
	}
	if _, err = metaF.Write(b); err != nil {
		return err
	}
	return metaF.Close()
}

// validateMeta checks that the meta file read from or about to be written to a meta file
// is well-defined.
func validateMeta(ch core.Channel) error {
	v := validate.New("meta")
	validate.Positive(v, "key", ch.Key)
	validate.NotEmptyString(v, "dataType", ch.DataType)
	if ch.Virtual {
		v.Ternaryf("index", ch.Index != 0, "virtual channel cannot be indexed")
		v.Ternaryf("rate", ch.Rate != 0, "virtual channel cannot have a rate")
	} else {
		v.Ternary("data_type", ch.DataType == telem.StringT, "persisted channels cannot have string data types")
		if ch.IsIndex {
			v.Ternary("data_type", ch.DataType != telem.TimeStampT, "index channel must be of type timestamp")
			v.Ternaryf("index", ch.Index != 0 && ch.Index != ch.Key, "index channel cannot be indexed by another channel")
		} else if ch.Index == 0 {
			validate.Positive(v, "rate", ch.Rate)
		}
	}
	return v.Error()
}
