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
	"encoding/json"
	"io"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
)

func Publish(
	ctx context.Context,
	provider *signals.Provider,
	obs observe.Observable[gorp.TxReader[channel.Key, channel.Channel]],
) (io.Closer, error) {
	cfg := signals.GorpPublisherConfig[channel.Key, channel.Channel]{
		Observable:     obs,
		DeleteDataType: telem.Uint32T,
		SetDataType:    telem.JSONT,
		MarshalDelete: func(k channel.Key) ([]byte, error) {
			return unsafe.CastToBytes(k), nil
		},
		MarshalSet: func(c channel.Channel) ([]byte, error) {
			v, err := json.Marshal(c.ToPayload())
			if err != nil {
				return nil, err
			}
			return telem.MarshalVariableSample(v), nil
		},
	}
	return signals.PublishFromGorp(ctx, provider, cfg)
}
