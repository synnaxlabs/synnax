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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v0"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
)

// MigrateLog lifts the previous log snapshot (v0, {Key, Name, Data}) into the typed
// Log. autoMigrateLog copies the gorp-entry fields (Key, Name); the rest is decoded
// from the per-log JSON blob by legacy.MigrateData. The lenient legacy decode admits
// enum strings outside their closed sets, so each is replaced with its standard default
// here, and the raw color string is parsed into the typed color.Color. A blob that
// fails to decode (e.g. a channel key that cannot coerce to uint32) is dropped, leaving
// Key and Name alone, so the Gorp boot migration never fails on a single corrupt row.
// v0 is the last snapshot in which Log.Data is untyped; future migrations transform one
// typed snapshot into another and never need this blob handling.
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
	out.Channels = lo.Map(d.Channels, func(c legacy.ChannelEntry, _ int) ChannelEntry {
		return ChannelEntry{
			Channel:   c.Channel,
			Color:     parseColor(c.Color),
			Notation:  orDefault(c.Notation, notation.NotationStandard),
			Precision: c.Precision,
			Alias:     c.Alias,
			Timestamp: TimestampConfig{
				Format: orDefault(c.Timestamp.Format, telem.TimestampFormatPreciseDate),
				Tz:     orDefault(c.Timestamp.TimeZone, telem.TimeZoneLocal),
			},
		}
	})
	out.TimestampPrecision = d.TimestampPrecision
	out.HideChannelNames = !d.ShowChannelNames
	out.HideReceiptTimestamp = !d.ShowReceiptTimestamp
	return out, nil
}

type validator interface{ IsValid() bool }

func orDefault[T validator](v, def T) T {
	if v.IsValid() {
		return v
	}
	return def
}

func parseColor(hex string) color.Color {
	c, err := color.FromHex(hex)
	if err != nil {
		return color.Color{}
	}
	return c
}

// Migration lifts stored logs from the v2 blob layout to the typed v3 shape.
var Migration = gorp.NewEntryMigration("v55_lift_typed_log", MigrateLog)
