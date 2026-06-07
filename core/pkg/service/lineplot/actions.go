// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"slices"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Handle replaces the line plot's name.
func (p RenamePayload) Handle(state LinePlot) (LinePlot, error) {
	state.Name = p.Name
	return state, nil
}

// Handle replaces the plot title configuration.
func (p SetTitlePayload) Handle(state LinePlot) (LinePlot, error) {
	state.Title = p.Title
	return state, nil
}

// Handle replaces the plot legend configuration.
func (p SetLegendPayload) Handle(state LinePlot) (LinePlot, error) {
	state.Legend = p.Legend
	return state, nil
}

// Handle appends the channel to the channels slice bound to the y-axis named
// by AxisKey. No-op when the channel is already bound to that axis. The
// inverse RemoveChannel undoes the append without preserving the original
// slice position, so a remove-then-undo cycle moves the channel to the end
// of the list. Channel order does not affect rendering, so this is accepted.
func (p AddChannelPayload) Handle(state LinePlot) (LinePlot, error) {
	if !p.AxisKey.IsValid() {
		return LinePlot{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown y-axis %q",
			p.AxisKey,
		)
	}
	slice := yAxisSlice(&state.Channels, p.AxisKey)
	if slices.Contains(*slice, p.Channel) {
		return state, nil
	}
	*slice = append(*slice, p.Channel)
	state.Lines = reconcileLines(state)
	return state, nil
}

// Handle removes the channel from the y-axis named by AxisKey. No-op when
// the channel is not present. See AddChannel for the inverse-position
// caveat that affects undo.
func (p RemoveChannelPayload) Handle(state LinePlot) (LinePlot, error) {
	if !p.AxisKey.IsValid() {
		return LinePlot{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown y-axis %q",
			p.AxisKey,
		)
	}
	slice := yAxisSlice(&state.Channels, p.AxisKey)
	*slice = slices.DeleteFunc(*slice, func(c channel.Key) bool { return c == p.Channel })
	state.Lines = reconcileLines(state)
	return state, nil
}

// Handle replaces the single channel bound to the x-axis named by AxisKey.
func (p SetXChannelPayload) Handle(state LinePlot) (LinePlot, error) {
	switch p.AxisKey {
	case XAxisKeyX1:
		state.Channels.X1 = p.Channel
	case XAxisKeyX2:
		state.Channels.X2 = p.Channel
	default:
		return LinePlot{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown x-axis %q",
			p.AxisKey,
		)
	}
	state.Lines = reconcileLines(state)
	return state, nil
}

// Handle appends the range key to the ranges slice bound to the x-axis named
// by AxisKey. No-op when the range is already bound to that axis. The
// inverse RemoveRange undoes the append without preserving original slice
// position; see AddChannel for the same caveat.
func (p AddRangePayload) Handle(state LinePlot) (LinePlot, error) {
	if !p.AxisKey.IsValid() {
		return LinePlot{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown x-axis %q",
			p.AxisKey,
		)
	}
	slice := xAxisRangeSlice(&state.Ranges, p.AxisKey)
	if slices.Contains(*slice, p.Range) {
		return state, nil
	}
	*slice = append(*slice, p.Range)
	state.Lines = reconcileLines(state)
	return state, nil
}

// Handle removes the range key from the x-axis named by AxisKey. No-op when
// the range is not present.
func (p RemoveRangePayload) Handle(state LinePlot) (LinePlot, error) {
	if !p.AxisKey.IsValid() {
		return LinePlot{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown x-axis %q",
			p.AxisKey,
		)
	}
	slice := xAxisRangeSlice(&state.Ranges, p.AxisKey)
	*slice = slices.DeleteFunc(*slice, func(r string) bool { return r == p.Range })
	state.Lines = reconcileLines(state)
	return state, nil
}

// Handle replaces the configuration for the axis named by axis.key.
func (p SetAxisPayload) Handle(state LinePlot) (LinePlot, error) {
	switch p.Axis.Key {
	case AxisKeyX1:
		state.Axes.X1 = p.Axis
	case AxisKeyX2:
		state.Axes.X2 = p.Axis
	case AxisKeyY1:
		state.Axes.Y1 = p.Axis
	case AxisKeyY2:
		state.Axes.Y2 = p.Axis
	case AxisKeyY3:
		state.Axes.Y3 = p.Axis
	case AxisKeyY4:
		state.Axes.Y4 = p.Axis
	default:
		return LinePlot{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown axis_key %q",
			p.Axis.Key,
		)
	}
	return state, nil
}

// Handle inserts the line if no line with the same key exists, otherwise
// replaces the existing entry in place.
func (p SetLinePayload) Handle(state LinePlot) (LinePlot, error) {
	for i := range state.Lines {
		if state.Lines[i].Key == p.Line.Key {
			state.Lines[i] = p.Line
			return state, nil
		}
	}
	state.Lines = append(state.Lines, p.Line)
	return state, nil
}

// Handle inserts the rule if no rule with the same key exists, otherwise
// replaces the existing entry in place.
func (p SetRulePayload) Handle(state LinePlot) (LinePlot, error) {
	for i := range state.Rules {
		if state.Rules[i].Key == p.Rule.Key {
			state.Rules[i] = p.Rule
			return state, nil
		}
	}
	state.Rules = append(state.Rules, p.Rule)
	return state, nil
}

// Handle removes the rule with the given key. No-op when not present.
func (p RemoveRulePayload) Handle(state LinePlot) (LinePlot, error) {
	state.Rules = slices.DeleteFunc(state.Rules, func(r Rule) bool { return r.Key == p.Key })
	return state, nil
}

// yAxisSlice returns a pointer to the channels slice for the given y-axis
// key so handlers can append or filter in place. Callers must have already
// validated k via IsValid; a non-matching key here means the y-axis enum
// gained a member that wasn't wired through, so panic loudly rather than
// returning nil for callers to dereference.
func yAxisSlice(c *Channels, k YAxisKey) *[]channel.Key {
	switch k {
	case YAxisKeyY1:
		return &c.Y1
	case YAxisKeyY2:
		return &c.Y2
	case YAxisKeyY3:
		return &c.Y3
	case YAxisKeyY4:
		return &c.Y4
	}
	panic(errors.Newf("lineplot: yAxisSlice called with non-y-axis %q", k))
}

// xAxisRangeSlice returns a pointer to the ranges slice for the given
// x-axis key so handlers can append or filter in place. Callers must have
// already validated k via IsValid; a non-matching key here means the x-axis
// enum gained a member that wasn't wired through, so panic loudly rather
// than returning nil for callers to dereference.
func xAxisRangeSlice(r *Ranges, k XAxisKey) *[]string {
	switch k {
	case XAxisKeyX1:
		return &r.X1
	case XAxisKeyX2:
		return &r.X2
	}
	panic(errors.Newf("lineplot: xAxisRangeSlice called with non-x-axis %q", k))
}

const lineKeySeparator = "---"

// Default styling for a newly materialized line. These mirror the Oracle schema
// defaults on Line. Oracle does not currently emit Go-side struct defaults, so
// they are duplicated here and must be kept in sync with schemas/lineplot.oracle.
const (
	defaultLineStrokeWidth    = 2
	defaultLineDownsample     = 1
	defaultLineDownsampleMode = DownsampleModeDecimate
)

var (
	xAxisKeys = []XAxisKey{XAxisKeyX1, XAxisKeyX2}
	yAxisKeys = []YAxisKey{YAxisKeyY1, YAxisKeyY2, YAxisKeyY3, YAxisKeyY4}
)

// lineKey encodes the identity of a line into the stable string stored as
// Line.Key. It must match the client's lineKey byte-for-byte so that lines
// reduced on the server and on the client share identity.
func lineKey(
	yAxis YAxisKey,
	xAxis XAxisKey,
	rng string,
	xChannel, yChannel channel.Key,
) string {
	return string(yAxis) + lineKeySeparator + string(xAxis) + lineKeySeparator + rng +
		lineKeySeparator + xChannel.String() + lineKeySeparator + yChannel.String()
}

// zeroLine constructs a line at key with default styling. Label and color are
// left nil so they resolve from the channel name and palette at render time.
func zeroLine(key string) Line {
	return Line{
		Key:            key,
		StrokeWidth:    defaultLineStrokeWidth,
		Downsample:     defaultLineDownsample,
		DownsampleMode: defaultLineDownsampleMode,
	}
}

// xAxisChannel returns the single channel bound to the given x-axis.
func xAxisChannel(c Channels, k XAxisKey) channel.Key {
	if k == XAxisKeyX2 {
		return c.X2
	}
	return c.X1
}

// reconcileLines rebuilds the complete set of lines implied by the channel and
// range bindings: one line per (x-axis, range, y-axis, y-channel) combination.
// Existing lines are preserved by key so user styling survives, missing ones are
// created with default styling, and lines whose combination no longer exists are
// dropped. It is the Go counterpart of the client's reconcileLines and must
// produce identical keys.
func reconcileLines(state LinePlot) []Line {
	byKey := make(map[string]Line, len(state.Lines))
	for _, l := range state.Lines {
		byKey[l.Key] = l
	}
	kept := set.New[string]()
	lines := make([]Line, 0, len(state.Lines))
	for _, xAxis := range xAxisKeys {
		xChannel := xAxisChannel(state.Channels, xAxis)
		for _, rng := range *xAxisRangeSlice(&state.Ranges, xAxis) {
			for _, yAxis := range yAxisKeys {
				for _, yChannel := range *yAxisSlice(&state.Channels, yAxis) {
					key := lineKey(yAxis, xAxis, rng, xChannel, yChannel)
					if kept.Contains(key) {
						continue
					}
					kept.Add(key)
					if existing, ok := byKey[key]; ok {
						lines = append(lines, existing)
					} else {
						lines = append(lines, zeroLine(key))
					}
				}
			}
		}
	}
	return lines
}
