// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package meta

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/binary"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"os"
)

const metaFile = "meta.json"

// ReadOrCreate reads the metadata file for a database whose data is kept in fs and is
// encoded by the provided encoder. If the file does not exist, it will be created. If
// the file does exist, it will be read and returned. The provided channel should have
// all fields required by the DB correctly set.
func ReadOrCreate(fs xfs.FS, ch core.Channel, ecd binary.EncoderDecoder) (core.Channel, error) {
	exists, err := fs.Exists(metaFile)
	if err != nil {
		return ch, err
	}
	if exists {
		return Read(fs, ecd)
	}

	if ch.DataType == telem.UnknownT {
		return ch, errors.Wrapf(
			validate.Error,
			"[meta] - cannot resolve properties for channel <%d> while creating db",
			ch.Key,
		)
	}

	return ch, Create(fs, ecd, ch)
}

// Read reads the metadata file for a database whose data is kept in fs and is encoded
// by the provided encoder.
func Read(fs xfs.FS, ecd binary.EncoderDecoder) (core.Channel, error) {
	metaF, err := fs.Open(metaFile, os.O_RDONLY)
	var ch core.Channel
	if err != nil {
		return ch, err
	}
	if err := ecd.DecodeStream(nil, metaF, &ch); err != nil {
		return ch, err
	}
	return ch, metaF.Close()
}

// Create creates the metadata file for a database whose data is kept in fs and is
// encoded by the provided encoder. The provided channel should have all fields
// required by the DB correctly set.
func Create(fs xfs.FS, ecd binary.EncoderDecoder, ch core.Channel) error {
	metaF, err := fs.Open(metaFile, os.O_CREATE|os.O_WRONLY)
	if err != nil {
		return err
	}
	b, err := ecd.Encode(nil, ch)
	if err != nil {
		return err
	}
	if _, err := metaF.Write(b); err != nil {
		return err
	}
	return metaF.Close()
}
