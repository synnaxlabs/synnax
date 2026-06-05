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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/zyn"
)

const Version imex.Version = 1

// ChannelEntry is a channel reference with display configuration. The yaml and toml tags
// mirror the json tags so the entry serializes identically across every portable format;
// the tags_test.go drift test enforces that the three stay in sync.
type ChannelEntry struct {
	Channel   int    `json:"channel" yaml:"channel" toml:"channel"`
	Color     string `json:"color" yaml:"color" toml:"color"`
	Notation  string `json:"notation" yaml:"notation" toml:"notation"`
	Precision int    `json:"precision" yaml:"precision" toml:"precision"`
	Alias     string `json:"alias" yaml:"alias" toml:"alias"`
}

// Data is the frozen type for log data at version 1. Channels are stored as config
// entries with display options. Key, Name, Type, and Version are envelope-level fields
// and are not part of Data.
type Data struct {
	Channels             []ChannelEntry `json:"channels" yaml:"channels" toml:"channels"`
	RemoteCreated        bool           `json:"remote_created" yaml:"remote_created" toml:"remote_created"`
	TimestampPrecision   int            `json:"timestamp_precision" yaml:"timestamp_precision" toml:"timestamp_precision"`
	ShowChannelNames     bool           `json:"show_channel_names" yaml:"show_channel_names" toml:"show_channel_names"`
	ShowReceiptTimestamp bool           `json:"show_receipt_timestamp" yaml:"show_receipt_timestamp" toml:"show_receipt_timestamp"`
}

// ToMap projects Data into the encoding-neutral map[string]any form used by the imex
// envelope and the Log storage layer. Keys must mirror the json tags on Data; the
// data_test.go drift test enforces that every tagged field appears here.
func (d Data) ToMap() map[string]any {
	return map[string]any{
		"channels":               d.Channels,
		"remote_created":         d.RemoteCreated,
		"timestamp_precision":    d.TimestampPrecision,
		"show_channel_names":     d.ShowChannelNames,
		"show_receipt_timestamp": d.ShowReceiptTimestamp,
	}
}

// Schema validates the wire shape of a v1 log payload.
var Schema = zyn.Object(map[string]zyn.Schema{
	"channels": zyn.Array(zyn.Object(map[string]zyn.Schema{
		"channel":   zyn.Number().Int().Coerce(),
		"color":     zyn.String().Optional(),
		"notation":  zyn.String().Optional(),
		"precision": zyn.Number().Int().Coerce().Optional(),
		"alias":     zyn.String().Optional(),
	})),
	"remote_created":         zyn.Bool().Optional(),
	"timestamp_precision":    zyn.Number().Int().Coerce().Optional(),
	"show_channel_names":     zyn.Bool().Optional(),
	"show_receipt_timestamp": zyn.Bool().Optional(),
})
