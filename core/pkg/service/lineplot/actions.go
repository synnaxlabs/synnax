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
	"reflect"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Handle replaces the document with its created state.
func (p CreatePayload) Handle(LinePlot) (LinePlot, error) {
	return p.LinePlot, nil
}

// Handle replaces the line plot's name.
func (p RenamePayload) Handle(state LinePlot) (LinePlot, error) {
	state.Name = p.Name
	return state, nil
}

// Handle sets whether the plot title is shown above the plot.
func (p SetTitleVisiblePayload) Handle(state LinePlot) (LinePlot, error) {
	state.Title.Visible = p.Visible
	return state, nil
}

// Handle sets the typography level of the plot title.
func (p SetTitleLevelPayload) Handle(state LinePlot) (LinePlot, error) {
	state.Title.Level = p.Level
	return state, nil
}

// Handle sets whether the plot legend is hidden.
func (p SetLegendHiddenPayload) Handle(state LinePlot) (LinePlot, error) {
	state.Legend.Hidden = p.Hidden
	return state, nil
}

// Handle sets the anchor position of the plot legend.
func (p SetLegendPositionPayload) Handle(state LinePlot) (LinePlot, error) {
	state.Legend.Position = p.Position
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
	state.Lines = v8.ReconcileLines(state.Key, state.Channels, state.Lines)
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
	*slice = slices.DeleteFunc(
		*slice,
		func(c channel.Key) bool { return c == p.Channel },
	)
	state.Lines = v8.ReconcileLines(state.Key, state.Channels, state.Lines)
	return state, nil
}

// Handle replaces the entire set of channels bound to the y-axis named by
// AxisKey, reconciling the line set. Channels dropped from the set lose their
// lines; channels new to it gain default-styled lines. Surviving channels keep
// their existing line styling.
func (p SetChannelsPayload) Handle(state LinePlot) (LinePlot, error) {
	if !p.AxisKey.IsValid() {
		return LinePlot{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown y-axis %q",
			p.AxisKey,
		)
	}
	slice := yAxisSlice(&state.Channels, p.AxisKey)
	*slice = slices.Clone(p.Channels)
	state.Lines = v8.ReconcileLines(state.Key, state.Channels, state.Lines)
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
	state.Lines = v8.ReconcileLines(state.Key, state.Channels, state.Lines)
	return state, nil
}

// Handle appends the range reference to the plot's ranges. No-op when an equal
// reference is already present.
func (p AddRangePayload) Handle(state LinePlot) (LinePlot, error) {
	if slices.ContainsFunc(state.Ranges, func(r ranger.Ref) bool {
		return reflect.DeepEqual(r, p.Range)
	}) {
		return state, nil
	}
	state.Ranges = append(slices.Clone(state.Ranges), p.Range)
	return state, nil
}

// Handle removes the reference equal to Range from the plot's ranges. No-op when no
// reference is equal.
func (p RemoveRangePayload) Handle(state LinePlot) (LinePlot, error) {
	state.Ranges = slices.DeleteFunc(
		slices.Clone(state.Ranges),
		func(r ranger.Ref) bool { return reflect.DeepEqual(r, p.Range) },
	)
	return state, nil
}

// Handle replaces the plot's range references. Lines style channels, so they are
// unaffected.
func (p SetRangesPayload) Handle(state LinePlot) (LinePlot, error) {
	state.Ranges = slices.Clone(p.Ranges)
	return state, nil
}

// Handle sets the label rendered along the axis named by key.
func (p SetAxisLabelPayload) Handle(state LinePlot) (LinePlot, error) {
	axis := axisPointer(&state.Axes, p.Key)
	if axis == nil {
		return LinePlot{}, unknownAxisKey(p.Key)
	}
	axis.Label = p.Label
	return state, nil
}

// Handle sets the orientation in which the axis label is laid out.
func (p SetAxisLabelDirectionPayload) Handle(state LinePlot) (LinePlot, error) {
	axis := axisPointer(&state.Axes, p.Key)
	if axis == nil {
		return LinePlot{}, unknownAxisKey(p.Key)
	}
	axis.LabelDirection = p.LabelDirection
	return state, nil
}

// Handle sets the typography level of the axis label.
func (p SetAxisLabelLevelPayload) Handle(state LinePlot) (LinePlot, error) {
	axis := axisPointer(&state.Axes, p.Key)
	if axis == nil {
		return LinePlot{}, unknownAxisKey(p.Key)
	}
	axis.LabelLevel = p.LabelLevel
	return state, nil
}

// Handle sets the axis value-space window together with its per-edge auto flags.
func (p SetAxisBoundsPayload) Handle(state LinePlot) (LinePlot, error) {
	axis := axisPointer(&state.Axes, p.Key)
	if axis == nil {
		return LinePlot{}, unknownAxisKey(p.Key)
	}
	axis.Bounds = p.Bounds
	axis.ManualBounds = p.ManualBounds
	return state, nil
}

// Handle sets the target pixel distance between adjacent tick marks.
func (p SetAxisTickSpacingPayload) Handle(state LinePlot) (LinePlot, error) {
	axis := axisPointer(&state.Axes, p.Key)
	if axis == nil {
		return LinePlot{}, unknownAxisKey(p.Key)
	}
	axis.TickSpacing = p.TickSpacing
	return state, nil
}

// Handle sets the axis tick label style. A nil type resets it to the default.
func (p SetAxisTypePayload) Handle(state LinePlot) (LinePlot, error) {
	axis := axisPointer(&state.Axes, p.Key)
	if axis == nil {
		return LinePlot{}, unknownAxisKey(p.Key)
	}
	axis.Type = p.Type
	return state, nil
}

func unknownAxisKey(k AxisKey) error {
	return errors.Wrapf(validate.ErrValidation, "unknown axis_key %q", k)
}

// axisPointer returns a pointer to the axis configuration for the given key so
// handlers can set into it in place, or nil if the key is not a valid axis.
func axisPointer(a *Axes, k AxisKey) *Axis {
	switch k {
	case AxisKeyX1:
		return &a.X1
	case AxisKeyX2:
		return &a.X2
	case AxisKeyY1:
		return &a.Y1
	case AxisKeyY2:
		return &a.Y2
	case AxisKeyY3:
		return &a.Y3
	case AxisKeyY4:
		return &a.Y4
	}
	return nil
}

// Handle sets the label of the line identified by key. A nil label resets it.
func (p SetLineLabelPayload) Handle(state LinePlot) (LinePlot, error) {
	if l := linePointer(&state, p.Key); l != nil {
		l.Label = p.Label
	}
	return state, nil
}

// Handle sets the color of the line identified by key. A nil color resets it.
func (p SetLineColorPayload) Handle(state LinePlot) (LinePlot, error) {
	if l := linePointer(&state, p.Key); l != nil {
		l.Color = p.Color
	}
	return state, nil
}

// Handle sets the stroke width, in pixels, of the line identified by key.
func (p SetLineStrokeWidthPayload) Handle(state LinePlot) (LinePlot, error) {
	if l := linePointer(&state, p.Key); l != nil {
		l.StrokeWidth = p.StrokeWidth
	}
	return state, nil
}

// Handle sets how the line identified by key reduces a pixel group.
func (p SetLineAggregationPayload) Handle(state LinePlot) (LinePlot, error) {
	if l := linePointer(&state, p.Key); l != nil {
		l.Aggregation = p.Aggregation
	}
	return state, nil
}

// Handle sets the number of sample groups per pixel column of the line by key.
func (p SetLineDetailPayload) Handle(state LinePlot) (LinePlot, error) {
	if l := linePointer(&state, p.Key); l != nil {
		l.Detail = p.Detail
	}
	return state, nil
}

// linePointer returns a pointer to the line with the given key so handlers can
// set into it in place, or nil if no such line exists.
func linePointer(state *LinePlot, key string) *Line {
	for i := range state.Lines {
		if state.Lines[i].Key == key {
			return &state.Lines[i]
		}
	}
	return nil
}

// Handle replaces the line with the same key in place, inserting it when no such
// line exists yet. The fine-grained SetLine* actions cover per-field edits; this
// full-object form restores a line dropped by reconciliation.
func (p SetLinePayload) Handle(state LinePlot) (LinePlot, error) {
	if l := linePointer(&state, p.Line.Key); l != nil {
		*l = p.Line
		return state, nil
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
	state.Rules = slices.DeleteFunc(
		state.Rules,
		func(r Rule) bool { return r.Key == p.Key },
	)
	return state, nil
}

// rulePointer returns a pointer to the rule with the given key so handlers can
// set into it in place, or nil if no such rule exists.
func rulePointer(state *LinePlot, key string) *Rule {
	for i := range state.Rules {
		if state.Rules[i].Key == key {
			return &state.Rules[i]
		}
	}
	return nil
}

// Handle sets the label of the rule identified by key.
func (p SetRuleLabelPayload) Handle(state LinePlot) (LinePlot, error) {
	if r := rulePointer(&state, p.Key); r != nil {
		r.Label = p.Label
	}
	return state, nil
}

// Handle sets the color of the rule identified by key. A nil color resets it.
func (p SetRuleColorPayload) Handle(state LinePlot) (LinePlot, error) {
	if r := rulePointer(&state, p.Key); r != nil {
		r.Color = p.Color
	}
	return state, nil
}

// Handle sets the axis the rule identified by key is anchored to.
func (p SetRuleAxisPayload) Handle(state LinePlot) (LinePlot, error) {
	if r := rulePointer(&state, p.Key); r != nil {
		r.Axis = p.Axis
	}
	return state, nil
}

// Handle sets the line width of the rule identified by key.
func (p SetRuleLineWidthPayload) Handle(state LinePlot) (LinePlot, error) {
	if r := rulePointer(&state, p.Key); r != nil {
		r.LineWidth = p.LineWidth
	}
	return state, nil
}

// Handle sets the dash length of the rule identified by key.
func (p SetRuleLineDashPayload) Handle(state LinePlot) (LinePlot, error) {
	if r := rulePointer(&state, p.Key); r != nil {
		r.LineDash = p.LineDash
	}
	return state, nil
}

// Handle sets the unit label of the rule identified by key.
func (p SetRuleUnitsPayload) Handle(state LinePlot) (LinePlot, error) {
	if r := rulePointer(&state, p.Key); r != nil {
		r.Units = p.Units
	}
	return state, nil
}

// Handle sets the value-space position of the rule identified by key.
func (p SetRulePositionPayload) Handle(state LinePlot) (LinePlot, error) {
	if r := rulePointer(&state, p.Key); r != nil {
		r.Position = p.Position
	}
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
