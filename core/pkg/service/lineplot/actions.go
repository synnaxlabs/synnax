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

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ScopedAction wraps an action sequence with the targeted line plot key, the
// dispatch-batch identifier supplied by the originating client, and a
// monotonic sequence number assigned by the node that handled the Dispatch.
// Subscribers compare Seq against the last value they applied for the same
// Key to decide whether the frame is fresh or a stale echo. DispatchKey is
// carried verbatim so the originating client can match the echo against the
// outstanding local replays it registered before sending, and skip a
// redundant reduce when no foreign action interleaved. The sequence is
// monotonic per originating node: in multi-node deployments two nodes may
// emit overlapping Seq values for the same Key, so cross-node ordering is
// best-effort until a cluster-wide ordering primitive lands as part of the
// broader server-side undo work.
type ScopedAction struct {
	Key         Key      `json:"key" msgpack:"key"`
	DispatchKey string   `json:"dispatch_key" msgpack:"dispatch_key"`
	Seq         uint64   `json:"seq" msgpack:"seq"`
	Actions     []Action `json:"actions" msgpack:"actions"`
}

// xAxisKeys is the closed set of axis keys that bind a single channel and an
// array of ranges. Used by the Add/Remove channel and range handlers to
// reject misrouted payloads at apply time.
var xAxisKeys = []AxisKey{AxisKeyX1, AxisKeyX2}

// yAxisKeys is the closed set of axis keys that bind a channel array. Used
// by the Add/Remove channel handlers to reject misrouted payloads at apply
// time.
var yAxisKeys = []AxisKey{AxisKeyY1, AxisKeyY2, AxisKeyY3, AxisKeyY4}

func isXAxis(k AxisKey) bool { return slices.Contains(xAxisKeys, k) }
func isYAxis(k AxisKey) bool { return slices.Contains(yAxisKeys, k) }

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
// by AxisKey. No-op when the channel is already bound to that axis.
func (p AddChannelPayload) Handle(state LinePlot) (LinePlot, error) {
	if !isYAxis(p.AxisKey) {
		return state, errors.Wrapf(
			validate.ErrValidation,
			"[LinePlot] - add_channel: axis_key %q must be a y-axis",
			p.AxisKey,
		)
	}
	slice := yAxisSlice(&state.Channels, p.AxisKey)
	if slices.Contains(*slice, p.Channel) {
		return state, nil
	}
	*slice = append(*slice, p.Channel)
	return state, nil
}

// Handle removes the channel from the y-axis named by AxisKey. No-op when
// the channel is not present.
func (p RemoveChannelPayload) Handle(state LinePlot) (LinePlot, error) {
	if !isYAxis(p.AxisKey) {
		return state, errors.Wrapf(
			validate.ErrValidation,
			"[LinePlot] - remove_channel: axis_key %q must be a y-axis",
			p.AxisKey,
		)
	}
	slice := yAxisSlice(&state.Channels, p.AxisKey)
	*slice = slices.DeleteFunc(*slice, func(c uint32) bool { return c == p.Channel })
	return state, nil
}

// Handle replaces the single channel bound to the x-axis named by AxisKey.
func (p SetXChannelPayload) Handle(state LinePlot) (LinePlot, error) {
	if !isXAxis(p.AxisKey) {
		return state, errors.Wrapf(
			validate.ErrValidation,
			"[LinePlot] - set_xchannel: axis_key %q must be an x-axis",
			p.AxisKey,
		)
	}
	switch p.AxisKey {
	case AxisKeyX1:
		state.Channels.X1 = p.Channel
	case AxisKeyX2:
		state.Channels.X2 = p.Channel
	}
	return state, nil
}

// Handle appends the range key to the ranges slice bound to the x-axis named
// by AxisKey. No-op when the range is already bound to that axis.
func (p AddRangePayload) Handle(state LinePlot) (LinePlot, error) {
	if !isXAxis(p.AxisKey) {
		return state, errors.Wrapf(
			validate.ErrValidation,
			"[LinePlot] - add_range: axis_key %q must be an x-axis",
			p.AxisKey,
		)
	}
	slice := xAxisRangeSlice(&state.Ranges, p.AxisKey)
	if slices.Contains(*slice, p.Range) {
		return state, nil
	}
	*slice = append(*slice, p.Range)
	return state, nil
}

// Handle removes the range key from the x-axis named by AxisKey. No-op when
// the range is not present.
func (p RemoveRangePayload) Handle(state LinePlot) (LinePlot, error) {
	if !isXAxis(p.AxisKey) {
		return state, errors.Wrapf(
			validate.ErrValidation,
			"[LinePlot] - remove_range: axis_key %q must be an x-axis",
			p.AxisKey,
		)
	}
	slice := xAxisRangeSlice(&state.Ranges, p.AxisKey)
	*slice = slices.DeleteFunc(*slice, func(r uuid.UUID) bool { return r == p.Range })
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
		return state, errors.Wrapf(
			validate.ErrValidation,
			"[LinePlot] - set_axis: unknown axis_key %q",
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
// key so handlers can append or filter in place.
func yAxisSlice(c *Channels, k AxisKey) *[]uint32 {
	switch k {
	case AxisKeyY1:
		return &c.Y1
	case AxisKeyY2:
		return &c.Y2
	case AxisKeyY3:
		return &c.Y3
	case AxisKeyY4:
		return &c.Y4
	}
	return nil
}

// xAxisRangeSlice returns a pointer to the ranges slice for the given
// x-axis key so handlers can append or filter in place.
func xAxisRangeSlice(r *Ranges, k AxisKey) *[]uuid.UUID {
	switch k {
	case AxisKeyX1:
		return &r.X1
	case AxisKeyX2:
		return &r.X2
	}
	return nil
}
