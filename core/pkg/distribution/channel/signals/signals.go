// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

import (
	"context"
	"io"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

func Publish(
	ctx context.Context,
	prov *signals.Provider,
	db *gorp.DB,
) (io.Closer, error) {
	cfg := signals.GorpPublisherConfigPureNumeric[channel.Key, channel.Channel](db, telem.Uint32T)
	cfg.SetDataType = telem.JSONT
	cfg.MarshalSet = func(c channel.Channel) ([]byte, error) {
		v, err := (&binary.JSONCodec{}).Encode(ctx, channel.ToPayload(c))
		if err != nil {
			return nil, err
		}
		return append(v, '\n'), nil
	}
	return signals.PublishFromGorp(ctx, prov, cfg)
}
