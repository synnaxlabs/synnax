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
  setAxis,
  setLegend,
  setLine,
  setRule,
  setTitle,
  setXChannel,
} from "@/lineplot/actions.gen";
import { type AxisKey } from "@/lineplot/types.gen";

type YAxisKey = "y1" | "y2" | "y3" | "y4";
type XAxisKey = "x1" | "x2";

// requireYAxis narrows AxisKey to the y-axis subset so the channel-array
// handlers can index state.channels[narrowed] type-safely. Throws when a
// client constructs an action targeting the wrong axis; the server rejects
// the same payload with validate.ErrValidation.
const requireYAxis = (key: AxisKey): YAxisKey => {
  if (key === "x1" || key === "x2")
    throw new Error(`expected a y-axis key, got ${key}`);
  return key;
};

const requireXAxis = (key: AxisKey): XAxisKey => {
  if (key === "x1" || key === "x2") return key;
  throw new Error(`expected an x-axis key, got ${key}`);
};

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

  setLegend: (state, payload) => {
    const oldLegend = actions.snapshotDraft(state.legend);
    state.legend = payload.legend;
    return { inverse: [setLegend({ legend: oldLegend })], targets: [state.key] };
  },

  addChannel: (state, payload) => {
    const axis = requireYAxis(payload.axisKey);
    const slice = state.channels[axis];
    if (slice.includes(payload.channel)) return actions.NO_OP_RESULT;
    slice.push(payload.channel);
    return {
      inverse: [removeChannel({ axisKey: axis, channel: payload.channel })],
      targets: [`channel:${payload.channel}`],
    };
  },

  removeChannel: (state, payload) => {
    const axis = requireYAxis(payload.axisKey);
    const slice = state.channels[axis];
    const idx = slice.indexOf(payload.channel);
    if (idx === -1) return actions.NO_OP_RESULT;
    slice.splice(idx, 1);
    return {
      inverse: [addChannel({ axisKey: axis, channel: payload.channel })],
      targets: [`channel:${payload.channel}`],
    };
  },

  setXChannel: (state, payload) => {
    const axis = requireXAxis(payload.axisKey);
    const oldChannel = state.channels[axis];
    state.channels[axis] = payload.channel;
    return {
      inverse: [setXChannel({ axisKey: axis, channel: oldChannel })],
      targets: [`axis:${axis}`],
    };
  },

  addRange: (state, payload) => {
    const axis = requireXAxis(payload.axisKey);
    const slice = state.ranges[axis];
    if (slice.includes(payload.range)) return actions.NO_OP_RESULT;
    slice.push(payload.range);
    return {
      inverse: [removeRange({ axisKey: axis, range: payload.range })],
      targets: [`range:${payload.range}`],
    };
  },

  removeRange: (state, payload) => {
    const axis = requireXAxis(payload.axisKey);
    const slice = state.ranges[axis];
    const idx = slice.indexOf(payload.range);
    if (idx === -1) return actions.NO_OP_RESULT;
    slice.splice(idx, 1);
    return {
      inverse: [addRange({ axisKey: axis, range: payload.range })],
      targets: [`range:${payload.range}`],
    };
  },

  setAxis: (state, payload) => {
    const oldAxis = actions.snapshotDraft(state.axes[payload.axis.key]);
    state.axes[payload.axis.key] = payload.axis;
    return {
      inverse: [setAxis({ axis: oldAxis })],
      targets: [`axis:${payload.axis.key}`],
    };
  },

  // setLine upserts: edits to an existing line are undoable, fresh inserts
  // are not (there is no removeLine inverse to record). Empty targets keep
  // create out of the undo stack so Cmd+Z never silently no-ops past it.
  // Lines are typically derived from addChannel/addRange, so direct creation
  // is the unusual path.
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
};

export const reduceAll = createReduceAll(handlers);
