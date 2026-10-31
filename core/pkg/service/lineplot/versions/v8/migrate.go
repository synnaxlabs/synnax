// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v8

import (
	"context"
	"crypto/sha1"
	"slices"
	"strconv"
	"strings"
	"uuid"

	"github.com/synnaxlabs/gorp/migrate"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v7 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v7"
	ranger "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

// lineNamespace is the UUID namespace that line keys are derived in.
var lineNamespace = uuid.MustParse("a708aa30-81f8-4cf6-82fd-146c35663a6b")

// LineKey returns the key of the plot's line that plots yChannel on yAxis against
// xChannel. The key is a name-based (version 5) UUID, so the Core and the Console
// derive the same key for the same line.
func LineKey(plot Key, yAxis YAxisKey, yChannel, xChannel channel.Key) string {
	name := plot.String() + "---" + string(yAxis) + "---" + yChannel.String() + "---" +
		xChannel.String()
	h := sha1.New()
	h.Write(lineNamespace[:])
	h.Write([]byte(name))
	var id uuid.UUID
	copy(id[:], h.Sum(nil))
	id[6] = (id[6] & 0x0f) | 0x50
	id[8] = (id[8] & 0x3f) | 0x80
	return id.String()
}

var yAxisKeys = []YAxisKey{YAxisKeyY1, YAxisKeyY2, YAxisKeyY3, YAxisKeyY4}

// ReconcileLines returns the plot's line set: one line per y-channel bound to a
// y-axis, plotted against the x1 channel. A line styles its channel across every
// range, so the set does not depend on the plot's ranges. An existing line with the
// same y-axis and channels keeps its styling and is re-keyed to the plot, so styling
// survives a plot getting a new key. Missing lines get default styling, with label
// and color nil so they resolve from the channel name and palette at render time.
func ReconcileLines(plot Key, channels Channels, existing []Line) []Line {
	type identity struct {
		yAxis              YAxisKey
		yChannel, xChannel channel.Key
	}
	byIdentity := make(map[identity]Line, len(existing))
	for _, l := range existing {
		id := identity{l.YAxis, l.YChannel, l.XChannel}
		if _, ok := byIdentity[id]; !ok {
			byIdentity[id] = l
		}
	}
	kept := set.New[identity]()
	lines := make([]Line, 0, len(existing))
	for _, yAxis := range yAxisKeys {
		for _, yChannel := range yAxisChannels(channels, yAxis) {
			id := identity{yAxis, yChannel, channels.X1}
			if kept.Contains(id) {
				continue
			}
			kept.Add(id)
			l, ok := byIdentity[id]
			if !ok {
				l = Line{YAxis: yAxis, YChannel: yChannel, XChannel: channels.X1}
				l.ApplyDefaults()
			}
			l.Key = LineKey(plot, yAxis, yChannel, channels.X1)
			lines = append(lines, l)
		}
	}
	return lines
}

func yAxisChannels(c Channels, k YAxisKey) []channel.Key {
	switch k {
	case YAxisKeyY1:
		return c.Y1
	case YAxisKeyY2:
		return c.Y2
	case YAxisKeyY3:
		return c.Y3
	case YAxisKeyY4:
		return c.Y4
	}
	panic(errors.Newf("lineplot: unknown y-axis %q", k))
}

// builtInSpans are the rolling windows the Console offered as range keys before v8.
var builtInSpans = map[string]telem.TimeSpan{
	"recent":     telem.TimeSpan(30 * telem.Second),
	"rolling1m":  telem.TimeSpan(telem.Minute),
	"rolling5m":  telem.TimeSpan(5 * telem.Minute),
	"rolling15m": telem.TimeSpan(15 * telem.Minute),
	"rolling30m": telem.TimeSpan(30 * telem.Minute),
	"rolling1h":  telem.TimeSpan(telem.Hour),
	"rolling6h":  telem.TimeSpan(6 * telem.Hour),
	"rolling12h": telem.TimeSpan(12 * telem.Hour),
	"rolling1d":  telem.TimeSpan(telem.Day),
	"rolling7d":  telem.TimeSpan(7 * telem.Day),
	"rolling30d": telem.TimeSpan(30 * telem.Day),
}

// customKey is the v7 range key that resolved to the plot's custom window.
const customKey = "custom"

// MigrateLinePlot lifts a v7 line plot to v8. The x1 and x2 range keys merge into
// one list of references. A built-in rolling key becomes a window with its span, and
// "custom" becomes the custom window, or is dropped when the plot has none. Other keys
// become key references, except keys that are not UUIDs, which name no saved range;
// the lineplot service later drops the ones the Core does not hold.
//
// v7 kept one line per range. They merge into one line per y-channel, which keeps the
// styling of the first v7 line in key order. The plot then holds exactly the lines its
// channel bindings imply: a v7 line whose channel is no longer bound, or whose key does
// not parse, is dropped.
func MigrateLinePlot(ctx context.Context, old v7.LinePlot) (LinePlot, error) {
	oldLines := old.Lines
	old.Lines = nil
	out, err := autoMigrateLinePlot(ctx, old)
	if err != nil {
		return LinePlot{}, err
	}
	out.Ranges = migrateRanges(old.Ranges)
	slices.SortFunc(
		oldLines,
		func(a, b v7.Line) int { return strings.Compare(a.Key, b.Key) },
	)
	merged := make([]Line, 0, len(oldLines))
	for _, ol := range oldLines {
		if _, ok := parseV7LineKey(ol.Key); !ok {
			continue
		}
		l, err := MigrateLine(ctx, ol)
		if err != nil {
			return LinePlot{}, err
		}
		merged = append(merged, l)
	}
	out.Lines = ReconcileLines(out.Key, out.Channels, merged)
	return out, nil
}

func migrateRanges(old v7.Ranges) []ranger.Ref {
	refs := make([]ranger.Ref, 0, len(old.X1)+len(old.X2))
	seen := set.New[string]()
	for _, key := range slices.Concat(old.X1, old.X2) {
		if seen.Contains(key) {
			continue
		}
		seen.Add(key)
		if span, ok := builtInSpans[key]; ok {
			refs = append(refs, windowRef(telem.Window{
				Variant: telem.DynamicWindow{TimeSpan: span},
			}))
			continue
		}
		if key == customKey {
			if old.Custom != nil {
				refs = append(refs, windowRef(migrateCustom(*old.Custom)))
			}
			continue
		}
		// A saved range's key is a UUID, so a key that does not parse names none.
		if k, err := uuid.Parse(key); err == nil {
			refs = append(refs, ranger.Ref{Variant: ranger.KeyRef{Key: k}})
		}
	}
	return refs
}

func windowRef(w telem.Window) ranger.Ref {
	return ranger.Ref{Variant: ranger.WindowRef{Window: w}}
}

func migrateCustom(c v7.CustomRange) telem.Window {
	if s, ok := c.Variant.(v7.StaticCustomRange); ok {
		return telem.Window{Variant: telem.StaticWindow{
			TimeRange: telem.TimeRange{Start: s.Start, End: s.End},
		}}
	}
	return telem.Window{
		Variant: telem.DynamicWindow{TimeSpan: c.Variant.(v7.DynamicCustomRange).Span},
	}
}

// v7LineKey is the decoded form of a v7 line key.
type v7LineKey struct {
	yAxis              YAxisKey
	xChannel, yChannel channel.Key
}

// parseV7LineKey decodes a v7 line key, which has the form
// "<y-axis>---<x-axis>---<range>---<x-channel>---<y-channel>".
func parseV7LineKey(key string) (v7LineKey, bool) {
	parts := strings.Split(key, "---")
	if len(parts) != 5 || !YAxisKey(parts[0]).IsValid() {
		return v7LineKey{}, false
	}
	xChannel, err := strconv.ParseUint(parts[3], 10, 32)
	if err != nil {
		return v7LineKey{}, false
	}
	yChannel, err := strconv.ParseUint(parts[4], 10, 32)
	if err != nil {
		return v7LineKey{}, false
	}
	return v7LineKey{
		yAxis:    YAxisKey(parts[0]),
		xChannel: channel.Key(xChannel),
		yChannel: channel.Key(yChannel),
	}, true
}

// MigrateLine lifts a v7 line to v8, decoding its y-axis and channels from the v7 key.
// It leaves the key empty, because a v8 line key depends on the plot's key, which
// MigrateLinePlot assigns. It returns an error when the v7 key does not parse.
func MigrateLine(ctx context.Context, old v7.Line) (Line, error) {
	k, ok := parseV7LineKey(old.Key)
	if !ok {
		return Line{}, errors.Newf("lineplot: malformed v7 line key %q", old.Key)
	}
	out, err := autoMigrateLine(ctx, old)
	if err != nil {
		return Line{}, err
	}
	out.Key = ""
	out.YAxis = k.yAxis
	out.YChannel = k.yChannel
	out.XChannel = k.xChannel
	return out, nil
}

// Migration lifts stored v7 line plots to v8.
var Migration = migrate.NewEntry("v61_lineplot_ranger_refs", MigrateLinePlot)

// RaiseStrokeWidths raises each line stroke width below 1 to 1, the smallest width
// Validate accepts.
func RaiseStrokeWidths(_ context.Context, lp LinePlot) (LinePlot, error) {
	for i := range lp.Lines {
		lp.Lines[i].StrokeWidth = max(lp.Lines[i].StrokeWidth, 1)
	}
	return lp, nil
}

// StrokeWidthMigration raises stored line stroke widths below 1 to 1.
var StrokeWidthMigration = migrate.NewEntry(
	"v61_min_line_stroke_width",
	RaiseStrokeWidths,
)
