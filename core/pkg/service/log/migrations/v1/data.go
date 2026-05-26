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
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
)

const Version imex.Version = 1

// TimestampConfig is per-channel timestamp display configuration. v1 was originally
// frozen without this struct; the field was added later by console clients and
// persisted through the v55 gorp blob without bumping the wire-format version, so v1
// captures it here for round-trip fidelity.
type TimestampConfig struct {
	Format telem.TimestampFormat `json:"format"`
	Tz     telem.TimeZone        `json:"tz"`
}

// ChannelEntry is a channel reference with display configuration. Fields are typed so a
// successful Parse produces a directly-usable value; color.Color.UnmarshalJSON converts
// the wire-format hex string into the typed Color.
type ChannelEntry struct {
	Channel   channel.Key       `json:"channel"`
	Color     color.Color       `json:"color"`
	Notation  notation.Notation `json:"notation"`
	Precision int32             `json:"precision"`
	Alias     string            `json:"alias"`
	Timestamp TimestampConfig   `json:"timestamp"`
}

// Data is the frozen type for log data at version 1. Channels are stored as config
// entries with display options. Key, Name, Type, and Version are envelope-level fields
// and are not part of Data.
type Data struct {
	Channels             []ChannelEntry `json:"channels"`
	RemoteCreated        bool           `json:"remoteCreated"`
	TimestampPrecision   int32          `json:"timestampPrecision"`
	ShowChannelNames     bool           `json:"showChannelNames"`
	ShowReceiptTimestamp bool           `json:"showReceiptTimestamp"`
}

// Parse marshals the imex map back to JSON and unmarshals it into a typed Data.
// The typed fields (color.Color, notation.Notation, telem.TimestampFormat,
// telem.TimeZone) handle their own conversion via the standard library's
// json.Unmarshaler interface where applicable, so the wire format flows directly
// into the typed shape without per-field parser helpers. An empty per-channel
// color string is scrubbed pre-unmarshal so it doesn't trip
// color.Color.UnmarshalJSON, which has no opinion on the empty case; closed-set
// membership for the enum fields is enforced by Validate, not by this function.
// Callers that need strict validation chain the two together.
func Parse(raw map[string]any) (Data, error) {
	scrubChannels(raw, func(s string) bool { return s == "" })
	b, err := json.Marshal(raw)
	if err != nil {
		return Data{}, err
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return Data{}, err
	}
	return d, nil
}

// ParseLenient is the gorp-boot variant of Parse. Before unmarshalling it also
// scrubs any per-channel color hex that would fail color.Color.UnmarshalJSON so
// a single corrupted field never blocks the whole row from loading. Enum strings
// (notation, timestamp format, timezone) flow through unchanged — they assign
// into named string types so unmarshal accepts any string; the latest-Log lift
// substitutes defaults for values outside the closed sets. The returned Data is
// always usable; an underlying json.Marshal failure produces a zero Data plus
// the error, which the catch-all in MigrateLog then drops.
func ParseLenient(raw map[string]any) (Data, error) {
	scrubChannels(raw, func(s string) bool {
		if s == "" {
			return true
		}
		_, err := color.FromHex(s)
		return err != nil
	})
	return Parse(raw)
}

// scrubChannels walks the per-channel color field on the raw map and removes any
// value for which drop returns true, so the subsequent json.Unmarshal sees the
// field as absent and color.Color stays at its zero value. The raw map is
// mutated in place; both Parse and ParseLenient deliberately accept the side
// effect because the map is owned by the imex envelope and isn't read again
// after migration.
func scrubChannels(raw map[string]any, drop func(string) bool) {
	list, ok := raw["channels"].([]any)
	if !ok {
		return
	}
	for _, e := range list {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		s, ok := m["color"].(string)
		if !ok {
			continue
		}
		if drop(s) {
			delete(m, "color")
		}
	}
}

// Validate enforces that every channel key is valid and that the typed enum fields hold
// closed-set values. Used by the imex import path so a malformed envelope is rejected at
// the API boundary; the lenient gorp-boot path skips this and lets the latest-Log lift
// substitute defaults for any invalid value that slipped past json.Unmarshal.
func (d Data) Validate() error {
	for i, c := range d.Channels {
		if err := c.Channel.Validate(); err != nil {
			return errors.Wrapf(err, "channels[%d].channel", i)
		}
		switch c.Notation {
		case "",
			notation.NotationStandard,
			notation.NotationScientific,
			notation.NotationEngineering:
		default:
			return errors.Newf("channels[%d].notation: invalid value %q", i, c.Notation)
		}
		switch c.Timestamp.Format {
		case "",
			telem.TimestampFormatPreciseTime,
			telem.TimestampFormatPreciseDate,
			telem.TimestampFormatISO:
		default:
			return errors.Newf(
				"channels[%d].timestamp.format: invalid value %q",
				i, c.Timestamp.Format,
			)
		}
		switch c.Timestamp.Tz {
		case "", telem.TimeZoneLocal, telem.TimeZoneUTC:
		default:
			return errors.Newf(
				"channels[%d].timestamp.tz: invalid value %q",
				i, c.Timestamp.Tz,
			)
		}
	}
	return nil
}
