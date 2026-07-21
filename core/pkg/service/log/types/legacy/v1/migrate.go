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
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/types/legacy/v0"
	notation "github.com/synnaxlabs/x/notation/types/v0"
	telem "github.com/synnaxlabs/x/telem/types/v0"
)

// Migrate lifts a typed v0.Data into a v1.Data, applying the v0→v1 schema
// transformation: bare channel keys become ChannelEntry records with default display
// options, and new v1 fields take their schema defaults.
func Migrate(old v0.Data) Data {
	channels := make([]ChannelEntry, len(old.Channels))
	for i, ch := range old.Channels {
		channels[i] = ChannelEntry{
			Channel:   ch,
			Notation:  notation.NotationStandard,
			Precision: -1,
			Timestamp: TimestampConfig{
				Format: telem.TimestampFormatPreciseDate,
				Tz:     telem.TimeZoneLocal,
			},
		}
	}
	return Data{
		Channels:             channels,
		RemoteCreated:        old.RemoteCreated,
		TimestampPrecision:   0,
		ShowChannelNames:     true,
		ShowReceiptTimestamp: true,
	}
}
