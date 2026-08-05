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
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	notation "github.com/synnaxlabs/x/notation/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

// Version is the ImEx schema version of log data at this state. The Console stamped it
// on the wire as the semver string "1.0.0", which legacy.MigrateData decodes onto this
// numeric version.
const Version imex.Version = 1

// TimestampConfig is per-channel timestamp display configuration. v1 was originally
// frozen without this struct; the field was added later by Console clients and
// persisted through the types/v0 gorp blob without bumping the wire-format version, so
// v1 captures it here for round-trip fidelity.
type TimestampConfig struct {
	// Format is the timestamp display format.
	Format telem.TimestampFormat `json:"format"`
	// Tz is the timestamp display time zone.
	Tz telem.TimeZone `json:"tz"`
}

// ChannelEntry is a channel reference with display configuration. Color is the raw
// wire-format color string the Console persisted (a hex string such as "#ff0000", or
// empty for none); the latest-Log lift parses it into the typed color.Color, defaulting
// a malformed or empty value to the zero color. The enum fields (notation, timestamp
// format, timezone) are named string types that accept any string on decode; the lift
// substitutes defaults for values outside their closed sets.
type ChannelEntry struct {
	// Channel is the key of the channel this entry displays.
	Channel channel.Key `json:"channel"`
	// Color is the raw hex color string; empty for none.
	Color string `json:"color"`
	// Notation is the numeric display notation.
	Notation notation.Notation `json:"notation"`
	// Precision is the number of decimal places displayed.
	Precision int32 `json:"precision"`
	// Alias overrides the channel name; empty for none.
	Alias string `json:"alias"`
	// Timestamp is the timestamp display configuration.
	Timestamp TimestampConfig `json:"timestamp"`
}

// Data is the frozen type for log data at version 1. Channels are stored as config
// entries with display options. Key, Name, Type, and Version are envelope-level fields
// and are not part of Data.
type Data struct {
	// Channels are the displayed channels with display options.
	Channels []ChannelEntry `json:"channels"`
	// RemoteCreated is UI-only sync bookkeeping; dropped on lift.
	RemoteCreated bool `json:"remoteCreated"`
	// TimestampPrecision is the timestamp display precision.
	TimestampPrecision int32 `json:"timestampPrecision"`
	// ShowChannelNames toggles channel name display.
	ShowChannelNames bool `json:"showChannelNames"`
	// ShowReceiptTimestamp toggles receipt timestamp display.
	ShowReceiptTimestamp bool `json:"showReceiptTimestamp"`
}

// Normalize replaces every per-channel enum that fails its Validate with the standard
// default (NotationStandard, TimestampFormatPreciseDate, TimeZoneLocal), so a value
// outside the closed set that slipped through the lenient decode never reaches the
// lifted Log. The channel entries are normalized in place and d is returned.
func (d Data) Normalize() Data {
	for i := range d.Channels {
		c := &d.Channels[i]
		if !c.Notation.IsValid() {
			c.Notation = notation.NotationStandard
		}
		if !c.Timestamp.Format.IsValid() {
			c.Timestamp.Format = telem.TimestampFormatPreciseDate
		}
		if !c.Timestamp.Tz.IsValid() {
			c.Timestamp.Tz = telem.TimeZoneLocal
		}
	}
	return d
}
