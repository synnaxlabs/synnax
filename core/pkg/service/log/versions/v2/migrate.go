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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v0"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// MigrateLog lifts the previous log snapshot (v0, {Key, Name, Data}) into the typed
// Log. autoMigrateLog copies the gorp-entry fields (Key, Name); the rest is decoded
// from the per-log JSON blob by legacy.MigrateData. The lenient legacy decode admits
// enum strings outside their closed sets, so each is replaced with its standard default
// here, and the raw color string is parsed into the typed color.Color. Fractional
// precisions are truncated, as the Console's formatter did. A blob that fails to decode
// (e.g. a channel key that cannot coerce to uint32) is an error. v0 is the last
// snapshot in which Log.Data is untyped; future migrations transform one typed snapshot
// into another and never need this blob handling.
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
		return Log{}, err
	}
	out.Channels = lo.Map(d.Channels, func(c legacy.ChannelEntry, _ int) ChannelEntry {
		return ChannelEntry{
			Channel:   c.Channel,
			Color:     parseColor(c.Color),
			Notation:  orDefault(c.Notation, notation.NotationStandard),
			Precision: int32(c.Precision),
			Alias:     c.Alias,
			Timestamp: TimestampConfig{
				Format: orDefault(c.Timestamp.Format, telem.TimestampFormatPreciseDate),
				Tz:     orDefault(c.Timestamp.TimeZone, telem.TimeZoneLocal),
			},
		}
	})
	out.TimestampPrecision = int32(d.TimestampPrecision)
	out.ChannelNamesHidden = !d.ShowChannelNames
	out.ReceiptTimestampHidden = !d.ShowReceiptTimestamp
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

// Migration lifts stored logs from the v0 blob layout to the typed v2 shape.
// migrateStored is MigrateLog for the stored-data migration. It keeps a log whose body
// does not decode with only its Key and Name, and logs the loss, so one corrupt log
// cannot stop the Core from starting.
func migrateStored(
	ctx context.Context,
	old v0.Log,
	ins alamos.Instrumentation,
) (Log, error) {
	out, err := MigrateLog(ctx, old)
	if err == nil {
		return out, nil
	}
	ins.L.Warn(
		"dropped a log body that does not decode",
		zap.Stringer("log", old.Key),
		zap.Error(err),
	)
	return autoMigrateLog(ctx, old)
}

var Migration = gorp.NewInstrumentedEntryMigration("v55_lift_typed_log", migrateStored)
