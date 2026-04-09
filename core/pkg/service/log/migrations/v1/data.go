// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import "github.com/synnaxlabs/x/zyn"

const Version = "1.0.0"

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

var channelEntrySchema = zyn.Object(map[string]zyn.Schema{
	"channel":   zyn.Number(),
	"color":     zyn.String(),
	"notation":  zyn.String(),
	"precision": zyn.Number(),
	"alias":     zyn.String(),
})

// Schema validates the full log resource at version 1.0.0.
var Schema = zyn.Object(map[string]zyn.Schema{
	"key":                  zyn.String().Optional(),
	"name":                 zyn.String().Optional(),
	"channels":             zyn.Array(channelEntrySchema),
	"remote_created":       zyn.Bool(),
	"timestamp_precision":  zyn.Number(),
	"show_channel_names":   zyn.Bool(),
	"show_receipt_timestamp": zyn.Bool(),
})
