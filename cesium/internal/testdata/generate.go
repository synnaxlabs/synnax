// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testdata

import (
	"context"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
)

var (
	ctx      = context.Background()
	channels = Channels
	frames   = Frames
)

func Generate() error {
	db, err := cesium.Open("data", cesium.WithFileSize(20*telem.ByteSize))
	if err != nil {
		return err
	}

	err = db.CreateChannel(ctx, channels...)
	if err != nil {
		return err
	}

	err = db.Write(ctx, 0, frames[0])
	if err != nil {
		return err
	}

	err = db.Write(ctx, 0, frames[1])
	if err != nil {
		return err
	}

	err = db.Write(ctx, 10*telem.SecondTS, frames[2])
	if err != nil {
		return err
	}

	err = db.Write(ctx, 13*telem.SecondTS, frames[3])
	if err != nil {
		return err
	}

	err = db.Write(ctx, 20*telem.SecondTS, frames[4])
	if err != nil {
		return err
	}

	return db.Close()
}
