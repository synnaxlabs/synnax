// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/binary"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/validate"
	"os"
)

const metaFile = "meta.json"

func readOrCreateMeta(cfg Config) (core.Channel, error) {
	exists, err := cfg.FS.Exists(metaFile)
	if err != nil {
		return cfg.Channel, err
	}
	if !exists {
		if cfg.Channel.Key == "" {
			return cfg.Channel, errors.Wrap(
				validate.Error,
				"[unary] - a channel is required when creating a new database",
			)
		}
		return cfg.Channel, createMeta(cfg.FS, cfg.MetaECD, cfg.Channel)
	}
	return readMeta(cfg.FS, cfg.MetaECD)
}

func readMeta(fs xfs.FS, ecd binary.EncoderDecoder) (core.Channel, error) {
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

func createMeta(fs xfs.FS, ecd binary.EncoderDecoder, ch core.Channel) error {
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
