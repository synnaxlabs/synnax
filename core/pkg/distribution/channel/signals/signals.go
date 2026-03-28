// Copyright 2026 Synnax Labs, Inc.
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

	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

func Publish(
	ctx context.Context,
	prov *signals.Provider,
	obs observe.Observable[gorp.TxReader[channel.Key, channel.Channel]],
) (io.Closer, error) {
	cfg := signals.GorpPublisherConfig[channel.Key, channel.Channel]{
		Observable:     obs,
		DeleteDataType: telem.Uint32T,
		SetDataType:    telem.JSONT,
		MarshalDelete: func(k channel.Key) ([]byte, error) {
			return xunsafe.CastToBytes(k), nil
		},
		MarshalSet: func(c channel.Channel) ([]byte, error) {
			v, err := json.Marshal(channel.ToPayload(c))
			if err != nil {
				return nil, err
			}
			return append(v, '\n'), nil
		},
	}
	return signals.PublishFromGorp(ctx, prov, cfg)
}
