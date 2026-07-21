// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/log/types/legacy"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/types/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/types/v2"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// MigrateLog lifts the previous log snapshot (v0, {Key, Name, Data}) into the
// strongly-typed Log. autoMigrateLog copies the gorp-entry fields (Key, Name); the body
// is decoded from the per-log JSON blob via legacy.MigrateData and lifted by logFromV1,
// which normalizes out-of-set enums to their documented defaults. If legacy.MigrateData
// fails (e.g. a channel key that cannot coerce to uint32), the body is dropped and only
// Key+Name are returned, so the Gorp boot migration never fails on a single corrupt
// row. UI-only fields (toolbar, version) are dropped because they are not declared in
// the v1 schema. v0 is the last snapshot in which Log.Data is untyped; future
// migrations transform one typed snapshot into another and never need this blob
// handling.
func MigrateLog(ctx context.Context, old v2.Log) (Log, error) {
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

// logFromV1 lifts the latest legacy Data into the runtime Log shape. Data.Normalize
// first replaces any out-of-set enum with its standard default, and the lift parses the
// raw color string into the typed color.Color (defaulting a malformed or empty value to
// the zero color).
func logFromV1(d v1.Data) Log {
	d = d.Normalize()
	channels := make([]ChannelEntry, len(d.Channels))
	for i, c := range d.Channels {
		channels[i] = ChannelEntry{
			Channel:   c.Channel,
			Color:     parseColor(c.Color),
			Notation:  c.Notation,
			Precision: c.Precision,
			Alias:     c.Alias,
			Timestamp: TimestampConfig{Format: c.Timestamp.Format, Tz: c.Timestamp.Tz},
		}
	}
	return Log{
		Channels:             channels,
		TimestampPrecision:   d.TimestampPrecision,
		HideChannelNames:     !d.ShowChannelNames,
		HideReceiptTimestamp: !d.ShowReceiptTimestamp,
	}
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

// codecMigration re-encodes stored logs from MessagePack to Orc. It is pinned to
// the v2 shape so its output stays stable as Log evolves.
var codecMigration = gorp.CodecMigration[Key, v2.Log]("msgpack_to_orc")

// liftMigration lifts stored logs from the v2 blob layout to the typed v3 shape.
var liftMigration = gorp.NewEntryMigration(
	"v55_lift_typed_log", MigrateLog, codecMigration.Key(),
)

// Migrations is the ordered set of migrations introduced at this version.
var Migrations = []migrate.Migration{codecMigration, liftMigration}
