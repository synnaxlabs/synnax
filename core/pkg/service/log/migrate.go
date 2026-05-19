// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log/migrations"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
	v55 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v55"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// migrateData parses a versioned wire-format log payload and lifts it into a typed
// Log via the strict migration chain. A malformed color hex or unknown notation
// value surfaces as a fatal error so the imex import path can reject the envelope
// at the API boundary. Key and Name are envelope-level and left zero; callers
// populate them.
func migrateData(version imex.Version, data map[string]any) (Log, error) {
	d, err := migrations.Migrate(version, data)
	if err != nil {
		return Log{}, err
	}
	return logFromV1(d), nil
}

// MigrateLog lifts the previous log snapshot (v55, {Key, Name, Data}) into the
// strongly-typed Log. AutoMigrateLog copies the gorp-entry fields (Key, Name); the
// body is parsed from the per-log JSON blob via migrations.MigrateLenient, which
// substitutes documented defaults for per-field parse failures rather than
// erroring. As a final guardrail, if the lenient chain still fails (e.g. a channel
// key that cannot coerce to uint32), the body is dropped and only Key+Name are
// returned. The gorp boot migration therefore never fails on a single corrupt row.
// UI-only fields (toolbar, version) are dropped because they are not declared in
// the v1 schema. v55 is the last snapshot in which Log.Data is untyped; future
// migrations transform one typed snapshot into another and never need this blob
// handling.
func MigrateLog(
	ctx context.Context,
	old v55.Log,
	ins alamos.Instrumentation,
) (Log, error) {
	out, err := AutoMigrateLog(ctx, old)
	if err != nil {
		return Log{}, err
	}
	if len(old.Data) == 0 {
		return out, nil
	}
	d, err := migrations.MigrateLenient(v1.Version, map[string]any(old.Data))
	if err != nil {
		ins.L.Warn(
			"log gorp migration: failed to parse Data body; dropping body and keeping Key+Name only",
			zap.String("key", out.Key.String()),
			zap.Error(err),
		)
		return out, nil
	}
	body := logFromV1(d)
	body.Key, body.Name = out.Key, out.Name
	return body, nil
}

// logFromV1 lifts the latest typed wire-format Data into the runtime Log shape.
// v1.Parse produces typed values; the lift is mostly a struct copy with closed-
// set substitution for the typed enums. In the strict path v1.Validate has
// already rejected anything outside the closed sets, so the substitution is a
// no-op; in the lenient path v1.ParseLenient lets bad enum strings flow through
// and substitution here turns them into the documented default.
func logFromV1(d v1.Data) Log {
	channels := make([]ChannelEntry, len(d.Channels))
	for i, c := range d.Channels {
		channels[i] = ChannelEntry{
			Channel:   c.Channel,
			Color:     c.Color,
			Notation:  defaultNotation(c.Notation),
			Precision: c.Precision,
			Alias:     c.Alias,
			Timestamp: TimestampConfig{
				Format: defaultTimestampFormat(c.Timestamp.Format),
				Tz:     defaultTimeZone(c.Timestamp.Tz),
			},
		}
	}
	return Log{
		Channels:             channels,
		RemoteCreated:        d.RemoteCreated,
		TimestampPrecision:   d.TimestampPrecision,
		ShowChannelNames:     d.ShowChannelNames,
		ShowReceiptTimestamp: d.ShowReceiptTimestamp,
	}
}

func defaultNotation(n notation.Notation) notation.Notation {
	switch n {
	case notation.NotationStandard,
		notation.NotationScientific,
		notation.NotationEngineering:
		return n
	}
	return notation.NotationStandard
}

func defaultTimestampFormat(f telem.TimestampFormat) telem.TimestampFormat {
	switch f {
	case telem.TimestampFormatPreciseTime,
		telem.TimestampFormatPreciseDate,
		telem.TimestampFormatISO:
		return f
	}
	return telem.TimestampFormatPreciseDate
}

func defaultTimeZone(tz telem.TimeZone) telem.TimeZone {
	switch tz {
	case telem.TimeZoneLocal, telem.TimeZoneUTC:
		return tz
	}
	return telem.TimeZoneLocal
}
