// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import "github.com/synnaxlabs/x/errors"

const Version = 5

// ChannelEntry is a channel reference with display configuration.
type ChannelEntry struct {
	Channel   int    `json:"channel"`
	Color     string `json:"color"`
	Notation  string `json:"notation"`
	Precision int    `json:"precision"`
	Alias     string `json:"alias"`
}

// Data is the frozen type for log data at version 1.0.0. Channels are stored as
// config entries with display options.
type Data struct {
	Key                  string         `json:"key"`
	Name                 string         `json:"name"`
	Channels             []ChannelEntry `json:"channels"`
	RemoteCreated        bool           `json:"remote_created"`
	TimestampPrecision   int            `json:"timestamp_precision"`
	ShowChannelNames     bool           `json:"show_channel_names"`
	ShowReceiptTimestamp bool           `json:"show_receipt_timestamp"`
}

// Validate enforces the structural invariants that the v1 Schema enforced
// implicitly: channels must be present (nil is treated as missing; empty slice
// is acceptable).
func (d Data) Validate() error {
	if d.Channels == nil {
		return errors.New("v1 log data: channels field is required")
	}
	return nil
}
