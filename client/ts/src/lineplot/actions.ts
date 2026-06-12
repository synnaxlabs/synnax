// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { actions } from "@/actions";
import {
  addChannel,
  addRange,
  createReduceAll,
  type Handlers,
  removeChannel,
  removeRange,
  removeRule,
  rename,
  setAxisBounds,
  setAxisLabel,
  setAxisLabelDirection,
  setAxisLabelLevel,
  setAxisTickSpacing,
  setAxisType,
  setChannels,
  setLegendPosition,
  setLegendVisible,
  setLine,
  setLineColor,
  setLineDownsample,
  setLineDownsampleMode,
  setLineLabel,
  setLineStrokeWidth,
  setRanges,
  setRule,
  setRuleAxis,
  setRuleColor,
  setRuleLabel,
  setRuleLineDash,
  setRuleLineWidth,
  setRulePosition,
  setRuleUnits,
  setTitle,
  setXChannel,
} from "@/lineplot/actions.gen";
import { reconcileLines } from "@/lineplot/line";

// Handlers report the narrowest resource each action touches as its target,
// type-prefixed to keep the key spaces (numeric channels, "y1" axes, uuid
// lines/rules) from colliding within a plot's undo state. Plot-level config
// (name, title, legend) targets the plot key itself; everything else targets
// the specific channel/range/axis/line/rule so concurrent edits to distinct
// resources neither coalesce nor invalidate each other's undoables.
const handlers: Handlers = {
  rename: (state, payload) => {
    const oldName = state.name;
    state.name = payload.name;
    return { inverse: [rename({ name: oldName })], targets: [state.key] };
  },

  setTitle: (state, payload) => {
    const oldTitle = actions.snapshotDraft(state.title);
    state.title = payload.title;
    return { inverse: [setTitle({ title: oldTitle })], targets: [state.key] };
  },

  setLegendVisible: (state, payload) => {
    const oldVisible = state.legend.visible;
    state.legend.visible = payload.visible;
    return {
      inverse: [setLegendVisible({ visible: oldVisible })],
      targets: [state.key],
    };
  },

  setLegendPosition: (state, payload) => {
    const oldPosition = actions.snapshotDraft(state.legend.position);
    state.legend.position = payload.position;
    return {
      inverse: [setLegendPosition({ position: oldPosition })],
      targets: [state.key],
    };
  },

  // addChannel appends the y-channel and reconciles the line set, materializing
  // a line for every (x-axis, range) combination it now participates in.
  // removeChannel is its inverse and tears those lines back down.
  addChannel: (state, payload) => {
    const axis = payload.axisKey;
    const slice = state.channels[axis];
    if (slice.includes(payload.channel)) return actions.NO_OP_RESULT;
    slice.push(payload.channel);
    state.lines = reconcileLines(state.channels, state.ranges, state.lines).lines;
    return {
      inverse: [removeChannel({ axisKey: axis, channel: payload.channel })],
      targets: [`channel:${payload.channel}`],
    };
  },

  // removeChannel drops the y-channel and reconciles the line set; every line it
  // produced is dropped. Dropped lines are restored verbatim on undo so user
  // styling survives a remove/add cycle.
  removeChannel: (state, payload) => {
    const axis = payload.axisKey;
    const slice = state.channels[axis];
    const idx = slice.indexOf(payload.channel);
    if (idx === -1) return actions.NO_OP_RESULT;
    slice.splice(idx, 1);
    const { lines, dropped } = reconcileLines(
      state.channels,
      state.ranges,
      state.lines,
    );
    state.lines = lines;
    return {
      inverse: [
        addChannel({ axisKey: axis, channel: payload.channel }),
        ...dropped.map((l) => setLine({ line: actions.snapshotDraft(l) })),
      ],
      targets: [`channel:${payload.channel}`],
    };
  },

  // setChannels replaces a y-axis's whole channel set in one edit, reconciling
  // the line set once. It is the bulk form of addChannel/removeChannel for the
  // multi-select toolbar: channels new to the set add lines, channels dropped
  // from it tear theirs down. Dropped lines are restored verbatim on undo so
  // styling survives, and targets name only the channels that actually changed
  // so the undoable invalidation stays per-channel.
  setChannels: (state, payload) => {
    const axis = payload.axisKey;
    const current = state.channels[axis];
    const nextSet = new Set(payload.channels);
    const currentSet = new Set(current);
    const removed = current.filter((channel) => !nextSet.has(channel));
    const added = payload.channels.filter((channel) => !currentSet.has(channel));
    if (added.length === 0 && removed.length === 0) return actions.NO_OP_RESULT;
    const oldChannels = actions.snapshotDraft(current);
    state.channels[axis] = [...payload.channels];
    const { lines, dropped } = reconcileLines(
      state.channels,
      state.ranges,
      state.lines,
    );
    state.lines = lines;
    return {
      inverse: [
        setChannels({ axisKey: axis, channels: oldChannels }),
        ...dropped.map((l) => setLine({ line: actions.snapshotDraft(l) })),
      ],
      targets: [...removed, ...added].map((channel) => `channel:${channel}`),
    };
  },

  // setXChannel swaps the single channel on an x-axis. Reconciliation rekeys
  // every line on that axis: the old lines are dropped and fresh ones
  // materialized. Undo restores both the previous channel and the previous
  // line styling.
  setXChannel: (state, payload) => {
    const axis = payload.axisKey;
    const oldChannel = state.channels[axis];
    if (oldChannel === payload.channel) return actions.NO_OP_RESULT;
    state.channels[axis] = payload.channel;
    const { lines, dropped } = reconcileLines(
      state.channels,
      state.ranges,
      state.lines,
    );
    state.lines = lines;
    return {
      inverse: [
        setXChannel({ axisKey: axis, channel: oldChannel }),
        ...dropped.map((l) => setLine({ line: actions.snapshotDraft(l) })),
      ],
      targets: [`axis:${axis}`],
    };
  },

  // addRange appends the range to an x-axis and reconciles the line set,
  // materializing a line for every y-channel plotted against it. removeRange is
  // its inverse.
  addRange: (state, payload) => {
    const axis = payload.axisKey;
    const slice = state.ranges[axis];
    if (slice.includes(payload.range)) return actions.NO_OP_RESULT;
    slice.push(payload.range);
    state.lines = reconcileLines(state.channels, state.ranges, state.lines).lines;
    return {
      inverse: [removeRange({ axisKey: axis, range: payload.range })],
      targets: [`range:${payload.range}`],
    };
  },

  // removeRange drops the range and reconciles the line set, dropping every line
  // it produced and restoring their styling on undo.
  removeRange: (state, payload) => {
    const axis = payload.axisKey;
    const slice = state.ranges[axis];
    const idx = slice.indexOf(payload.range);
    if (idx === -1) return actions.NO_OP_RESULT;
    slice.splice(idx, 1);
    const { lines, dropped } = reconcileLines(
      state.channels,
      state.ranges,
      state.lines,
    );
    state.lines = lines;
    return {
      inverse: [
        addRange({ axisKey: axis, range: payload.range }),
        ...dropped.map((l) => setLine({ line: actions.snapshotDraft(l) })),
      ],
      targets: [`range:${payload.range}`],
    };
  },

  // setRanges replaces an x-axis's whole range set in one edit, reconciling the
  // line set once. It is the bulk form of addRange/removeRange for the
  // multi-select toolbar; dropped lines are restored verbatim on undo and
  // targets name only the ranges that actually changed.
  setRanges: (state, payload) => {
    const axis = payload.axisKey;
    const current = state.ranges[axis];
    const nextSet = new Set(payload.ranges);
    const currentSet = new Set(current);
    const removed = current.filter((range) => !nextSet.has(range));
    const added = payload.ranges.filter((range) => !currentSet.has(range));
    if (added.length === 0 && removed.length === 0) return actions.NO_OP_RESULT;
    const oldRanges = actions.snapshotDraft(current);
    state.ranges[axis] = [...payload.ranges];
    const { lines, dropped } = reconcileLines(
      state.channels,
      state.ranges,
      state.lines,
    );
    state.lines = lines;
    return {
      inverse: [
        setRanges({ axisKey: axis, ranges: oldRanges }),
        ...dropped.map((l) => setLine({ line: actions.snapshotDraft(l) })),
      ],
      targets: [...removed, ...added].map((range) => `range:${range}`),
    };
  },

  // Each setAxis* action sets one field of the named axis in place.
  setAxisLabel: (state, payload) => {
    const axis = state.axes[payload.key];
    const inverse = [setAxisLabel({ key: payload.key, label: axis.label })];
    axis.label = payload.label;
    return { inverse, targets: [`axis:${payload.key}`] };
  },

  setAxisLabelDirection: (state, payload) => {
    const axis = state.axes[payload.key];
    const inverse = [
      setAxisLabelDirection({ key: payload.key, labelDirection: axis.labelDirection }),
    ];
    axis.labelDirection = payload.labelDirection;
    return { inverse, targets: [`axis:${payload.key}`] };
  },

  setAxisLabelLevel: (state, payload) => {
    const axis = state.axes[payload.key];
    const inverse = [
      setAxisLabelLevel({ key: payload.key, labelLevel: axis.labelLevel }),
    ];
    axis.labelLevel = payload.labelLevel;
    return { inverse, targets: [`axis:${payload.key}`] };
  },

  setAxisBounds: (state, payload) => {
    const axis = state.axes[payload.key];
    const inverse = [
      setAxisBounds({
        key: payload.key,
        bounds: actions.snapshotDraft(axis.bounds),
        autoBounds: actions.snapshotDraft(axis.autoBounds),
      }),
    ];
    axis.bounds = payload.bounds;
    axis.autoBounds = payload.autoBounds;
    return { inverse, targets: [`axis:${payload.key}`] };
  },

  setAxisTickSpacing: (state, payload) => {
    const axis = state.axes[payload.key];
    const inverse = [
      setAxisTickSpacing({ key: payload.key, tickSpacing: axis.tickSpacing }),
    ];
    axis.tickSpacing = payload.tickSpacing;
    return { inverse, targets: [`axis:${payload.key}`] };
  },

  setAxisType: (state, payload) => {
    const axis = state.axes[payload.key];
    const inverse = [setAxisType({ key: payload.key, type: axis.type })];
    axis.type = payload.type;
    return { inverse, targets: [`axis:${payload.key}`] };
  },

  setLineLabel: (state, payload) => {
    const line = state.lines.find((l) => l.key === payload.key);
    if (line == null) return actions.NO_OP_RESULT;
    const inverse = [setLineLabel({ key: payload.key, label: line.label })];
    line.label = payload.label;
    return { inverse, targets: [`line:${payload.key}`] };
  },

  setLineColor: (state, payload) => {
    const line = state.lines.find((l) => l.key === payload.key);
    if (line == null) return actions.NO_OP_RESULT;
    const inverse = [
      setLineColor({
        key: payload.key,
        color: line.color === undefined ? undefined : actions.snapshotDraft(line.color),
      }),
    ];
    line.color = payload.color;
    return { inverse, targets: [`line:${payload.key}`] };
  },

  setLineStrokeWidth: (state, payload) => {
    const line = state.lines.find((l) => l.key === payload.key);
    if (line == null) return actions.NO_OP_RESULT;
    const inverse = [
      setLineStrokeWidth({ key: payload.key, strokeWidth: line.strokeWidth }),
    ];
    line.strokeWidth = payload.strokeWidth;
    return { inverse, targets: [`line:${payload.key}`] };
  },

  setLineDownsample: (state, payload) => {
    const line = state.lines.find((l) => l.key === payload.key);
    if (line == null) return actions.NO_OP_RESULT;
    const inverse = [
      setLineDownsample({ key: payload.key, downsample: line.downsample }),
    ];
    line.downsample = payload.downsample;
    return { inverse, targets: [`line:${payload.key}`] };
  },

  setLineDownsampleMode: (state, payload) => {
    const line = state.lines.find((l) => l.key === payload.key);
    if (line == null) return actions.NO_OP_RESULT;
    const inverse = [
      setLineDownsampleMode({ key: payload.key, downsampleMode: line.downsampleMode }),
    ];
    line.downsampleMode = payload.downsampleMode;
    return { inverse, targets: [`line:${payload.key}`] };
  },

  // setLine replaces a line's full configuration in place, or inserts it when its
  // key is new. The fine-grained setLine* actions cover per-field toolbar edits;
  // this full-object form restores a line dropped by reconciliation, so a fresh
  // insert is not independently undoable (it only re-materializes a dropped line).
  setLine: (state, payload) => {
    const idx = state.lines.findIndex((l) => l.key === payload.line.key);
    if (idx === -1) {
      state.lines.push(payload.line);
      return { inverse: [], targets: [] };
    }
    const oldLine = actions.snapshotDraft(state.lines[idx]);
    state.lines[idx] = payload.line;
    return {
      inverse: [setLine({ line: oldLine })],
      targets: [`line:${payload.line.key}`],
    };
  },

  setRule: (state, payload) => {
    const idx = state.rules.findIndex((r) => r.key === payload.rule.key);
    if (idx === -1) {
      state.rules.push(payload.rule);
      return {
        inverse: [removeRule({ key: payload.rule.key })],
        targets: [`rule:${payload.rule.key}`],
      };
    }
    const oldRule = actions.snapshotDraft(state.rules[idx]);
    state.rules[idx] = payload.rule;
    return {
      inverse: [setRule({ rule: oldRule })],
      targets: [`rule:${payload.rule.key}`],
    };
  },

  removeRule: (state, payload) => {
    const idx = state.rules.findIndex((r) => r.key === payload.key);
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldRule = actions.snapshotDraft(state.rules[idx]);
    state.rules.splice(idx, 1);
    return { inverse: [setRule({ rule: oldRule })], targets: [`rule:${payload.key}`] };
  },

  // Each setRule* action sets one field of the rule identified by key.
  setRuleLabel: (state, payload) => {
    const rule = state.rules.find((r) => r.key === payload.key);
    if (rule == null) return actions.NO_OP_RESULT;
    const inverse = [setRuleLabel({ key: payload.key, label: rule.label })];
    rule.label = payload.label;
    return { inverse, targets: [`rule:${payload.key}`] };
  },

  setRuleColor: (state, payload) => {
    const rule = state.rules.find((r) => r.key === payload.key);
    if (rule == null) return actions.NO_OP_RESULT;
    const inverse = [
      setRuleColor({
        key: payload.key,
        color: rule.color === undefined ? undefined : actions.snapshotDraft(rule.color),
      }),
    ];
    rule.color = payload.color;
    return { inverse, targets: [`rule:${payload.key}`] };
  },

  setRuleAxis: (state, payload) => {
    const rule = state.rules.find((r) => r.key === payload.key);
    if (rule == null) return actions.NO_OP_RESULT;
    const inverse = [setRuleAxis({ key: payload.key, axis: rule.axis })];
    rule.axis = payload.axis;
    return { inverse, targets: [`rule:${payload.key}`] };
  },

  setRuleLineWidth: (state, payload) => {
    const rule = state.rules.find((r) => r.key === payload.key);
    if (rule == null) return actions.NO_OP_RESULT;
    const inverse = [setRuleLineWidth({ key: payload.key, lineWidth: rule.lineWidth })];
    rule.lineWidth = payload.lineWidth;
    return { inverse, targets: [`rule:${payload.key}`] };
  },

  setRuleLineDash: (state, payload) => {
    const rule = state.rules.find((r) => r.key === payload.key);
    if (rule == null) return actions.NO_OP_RESULT;
    const inverse = [setRuleLineDash({ key: payload.key, lineDash: rule.lineDash })];
    rule.lineDash = payload.lineDash;
    return { inverse, targets: [`rule:${payload.key}`] };
  },

  setRuleUnits: (state, payload) => {
    const rule = state.rules.find((r) => r.key === payload.key);
    if (rule == null) return actions.NO_OP_RESULT;
    const inverse = [setRuleUnits({ key: payload.key, units: rule.units })];
    rule.units = payload.units;
    return { inverse, targets: [`rule:${payload.key}`] };
  },

  setRulePosition: (state, payload) => {
    const rule = state.rules.find((r) => r.key === payload.key);
    if (rule == null) return actions.NO_OP_RESULT;
    const inverse = [setRulePosition({ key: payload.key, position: rule.position })];
    rule.position = payload.position;
    return { inverse, targets: [`rule:${payload.key}`] };
  },
};

export const reduceAll = createReduceAll(handlers);
