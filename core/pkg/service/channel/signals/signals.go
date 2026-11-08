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
	"encoding/json/v2"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/telem"
)

// TableConfig returns the configuration of the set and delete channels of the channel
// table, which sends each channel as its payload.
func TableConfig(
	svc *channel.Service,
) signals.TableConfig[channel.Key, channel.Channel] {
	return signals.TableConfig[channel.Key, channel.Channel]{
		Changes: svc.Changes,
		MarshalSet: func(c channel.Channel) ([]byte, error) {
			v, err := json.Marshal(c.ToPayload())
			if err != nil {
				return nil, err
			}
			return telem.MarshalVariableSample(v), nil
		},
	}
}
