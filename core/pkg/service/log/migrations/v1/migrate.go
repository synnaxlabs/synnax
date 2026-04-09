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
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v0"
)

// Migrate transforms a v0 log data payload into a v1 payload.
func Migrate(old v0.Data) (Data, error) {
	channels := make([]ChannelEntry, len(old.Channels))
	for i, ch := range old.Channels {
		channels[i] = ChannelEntry{
			Channel:   ch,
			Color:     "",
			Notation:  "standard",
			Precision: -1,
			Alias:     "",
		}
	}
	return Data{
		Key:                  old.Key,
		Name:                 old.Name,
		Channels:             channels,
		RemoteCreated:        old.RemoteCreated,
		TimestampPrecision:   0,
		ShowChannelNames:     true,
		ShowReceiptTimestamp:  true,
	}, nil
}
