// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"github.com/samber/lo"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy/v0"
	notation "github.com/synnaxlabs/x/notation/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

// Migrate lifts a typed v0.Data into a v1.Data, applying the v0→v1 schema
// transformation: bare channel keys become ChannelEntry records with default display
// options, and new v1 fields take their schema defaults.
func Migrate(old v0.Data) Data {
	return Data{
		Channels: lo.Map(old.Channels, func(ch channel.Key, _ int) ChannelEntry {
			return ChannelEntry{
				Channel:   ch,
				Notation:  notation.NotationStandard,
				Precision: -1,
				Timestamp: TimestampConfig{
					Format:   telem.TimestampFormatPreciseDate,
					TimeZone: telem.TimeZoneLocal,
				},
			}
		}),
		TimestampPrecision:   0,
		ShowChannelNames:     true,
		ShowReceiptTimestamp: true,
	}
}
