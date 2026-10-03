// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v7

import (
	"context"
	"strings"
	"uuid"

	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
	"github.com/synnaxlabs/x/gorp"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

const (
	// rollingLineRange is the range part of the key of a line plotted over an axis's
	// rolling window.
	rollingLineRange = "rolling"
	lineKeySeparator = "---"
	customRangeKey   = "custom"
)

// builtInSpans holds the rolling windows every released Console offered, by the range
// key v6 plots stored for them.
var builtInSpans = map[string]telem.TimeSpan{
	"recent":     30 * telem.Second,
	"rolling1m":  telem.Minute,
	"rolling5m":  5 * telem.Minute,
	"rolling15m": 15 * telem.Minute,
	"rolling30m": 30 * telem.Minute,
	"rolling1h":  telem.Hour,
	"rolling6h":  6 * telem.Hour,
	"rolling12h": 12 * telem.Hour,
	"rolling1d":  telem.Day,
	"rolling7d":  7 * telem.Day,
	"rolling30d": 30 * telem.Day,
}

// staticCustomKey keys the entry a fixed v6 custom window becomes. Keys are unique
// only within an axis, so one constant serves every plot.
var staticCustomKey = uuid.MustParse("cb056cea-fb6b-42aa-9991-d91bdf6db8ab")

// MigrateLinePlot lifts a v6 plot to v7 and renames the range part of each line key
// to match its migrated range. Lines whose range did not survive are dropped.
func MigrateLinePlot(ctx context.Context, old v6.LinePlot) (LinePlot, error) {
	lp, err := autoMigrateLinePlot(ctx, old)
	if err != nil {
		return LinePlot{}, err
	}
	_, x1 := migrateAxis(old.Ranges.X1, old.Ranges.Custom)
	_, x2 := migrateAxis(old.Ranges.X2, old.Ranges.Custom)
	renames := map[string]map[string]string{"x1": x1, "x2": x2}
	lp.Lines = make([]Line, 0, len(old.Lines))
	for _, l := range old.Lines {
		parts := strings.Split(l.Key, lineKeySeparator)
		if len(parts) != 5 {
			continue
		}
		renamed, ok := renames[parts[1]][parts[2]]
		if !ok {
			continue
		}
		parts[2] = renamed
		l.Key = strings.Join(parts, lineKeySeparator)
		lp.Lines = append(lp.Lines, l)
	}
	return lp, nil
}

// MigrateRanges splits each axis's range keys into its rolling window and its ranges.
func MigrateRanges(_ context.Context, old v6.Ranges) (Ranges, error) {
	x1, _ := migrateAxis(old.X1, old.Custom)
	x2, _ := migrateAxis(old.X2, old.Custom)
	return Ranges{X1: x1, X2: x2}, nil
}

// migrateAxis converts one axis's v6 range keys. It also returns the line key range
// part each surviving key becomes. The widest rolling key wins the axis's one rolling
// window, since v6 drew every rolling window over the bounds of the widest. Keys that
// name nothing a v7 plot can hold are dropped.
func migrateAxis(
	keys []string,
	custom *v6.CustomRange,
) (XAxisRanges, map[string]string) {
	axis := XAxisRanges{Ranges: []Range{}}
	renames := make(map[string]string, len(keys))
	var widest string
	for _, k := range keys {
		span, ok := rollingSpan(k, custom)
		if ok && (axis.Rolling == nil || span > *axis.Rolling) {
			axis.Rolling, widest = &span, k
		}
	}
	if axis.Rolling != nil {
		renames[widest] = rollingLineRange
	}
	for _, k := range keys {
		if _, seen := renames[k]; seen {
			continue
		}
		if _, ok := rollingSpan(k, custom); ok {
			continue
		}
		if k == customRangeKey {
			if custom == nil {
				continue
			}
			static, ok := custom.Variant.(v6.StaticCustomRange)
			if !ok {
				continue
			}
			axis.Ranges = append(axis.Ranges, Range{Variant: StaticRange{
				BaseRange: BaseRange{Key: staticCustomKey},
				Start:     static.Start,
				End:       static.End,
			}})
			renames[k] = staticCustomKey.String()
			continue
		}
		key, err := uuid.Parse(k)
		if err != nil {
			continue
		}
		axis.Ranges = append(axis.Ranges, Range{
			Variant: PersistedRange{BaseRange: BaseRange{Key: key}},
		})
		renames[k] = key.String()
	}
	return axis, renames
}

// rollingSpan returns the span of the rolling window k names, if it names one.
func rollingSpan(k string, custom *v6.CustomRange) (telem.TimeSpan, bool) {
	if span, ok := builtInSpans[k]; ok {
		return span, true
	}
	if k != customRangeKey || custom == nil {
		return 0, false
	}
	dynamic, ok := custom.Variant.(v6.DynamicCustomRange)
	return dynamic.Span, ok
}

var Migration = gorp.NewEntryMigration("v59_range_entries", MigrateLinePlot)
