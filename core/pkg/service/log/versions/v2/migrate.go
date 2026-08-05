// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v0"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
)

// MigrateLog lifts the previous log snapshot (v2, {Key, Name, Data}) into the v3
// strongly-typed Log. autoMigrateLog copies the gorp-entry fields (Key, Name); the body
// is decoded from the per-log JSON blob via legacy.MigrateData and lifted by logFromV1,
// which normalizes out-of-set enums to their documented defaults. If legacy.MigrateData
// fails (e.g. a channel key that cannot coerce to uint32), the body is dropped and only
// Key+Name are returned, so the Gorp boot migration never fails on a single corrupt
// row. UI-only fields (toolbar, version) are dropped because they are not declared in
// the v1 schema. v0 is the last snapshot in which Log.Data is untyped; future
// migrations transform one typed snapshot into another and never need this blob
// handling.
func MigrateLog(ctx context.Context, old v0.Log) (Log, error) {
	out, err := autoMigrateLog(ctx, old)
	if err != nil {
		return Log{}, err
	}
	if len(old.Data) == 0 {
		return out, nil
	}
	d, err := legacy.MigrateData(old.Data)
	if err != nil {
		return out, nil
	}
	body := logFromV1(d)
	body.Key, body.Name = out.Key, out.Name
	return body, nil
}

// logFromV1 lifts the latest legacy Data into the runtime Log shape. The lenient
// legacy decode admits enum strings outside their closed sets, so each is replaced with
// its standard default here, and the raw color string is parsed into the typed
// color.Color (defaulting a malformed or empty value to the zero color).
func logFromV1(d legacy.Data) Log {
	channels := make([]ChannelEntry, len(d.Channels))
	for i, c := range d.Channels {
		channels[i] = ChannelEntry{
			Channel:   c.Channel,
			Color:     parseColor(c.Color),
			Notation:  orDefault(c.Notation, notation.NotationStandard),
			Precision: c.Precision,
			Alias:     c.Alias,
			Timestamp: TimestampConfig{
				Format: orDefault(c.Timestamp.Format, telem.TimestampFormatPreciseDate),
				Tz:     orDefault(c.Timestamp.Tz, telem.TimeZoneLocal),
			},
		}
	}
	return Log{
		Channels:             channels,
		TimestampPrecision:   d.TimestampPrecision,
		HideChannelNames:     !d.ShowChannelNames,
		HideReceiptTimestamp: !d.ShowReceiptTimestamp,
	}
}

// validator is an enum that can report whether it holds one of its defined values.
type validator interface{ IsValid() bool }

// orDefault returns v when it holds a defined enum value, and def otherwise.
func orDefault[T validator](v, def T) T {
	if v.IsValid() {
		return v
	}
	return def
}

// parseColor parses the raw wire-format hex color string into the typed color.Color,
// returning the zero color for an empty or malformed value so a single bad color never
// fails the lift.
func parseColor(hex string) color.Color {
	c, err := color.FromHex(hex)
	if err != nil {
		return color.Color{}
	}
	return c
}

// Migration lifts stored logs from the v2 blob layout to the typed v3 shape.
var Migration = gorp.NewEntryMigration("v55_lift_typed_log", MigrateLog)
