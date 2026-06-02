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
      targets: [state.key],
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
      targets: [state.key],
    };
  },

  setXChannel: (state, payload) => {
    const axis = requireXAxis(payload.axisKey);
    const oldChannel = state.channels[axis];
    state.channels[axis] = payload.channel;
    return {
      inverse: [setXChannel({ axisKey: axis, channel: oldChannel })],
      targets: [state.key],
    };
  },

  addRange: (state, payload) => {
    const axis = requireXAxis(payload.axisKey);
    const slice = state.ranges[axis];
    if (slice.includes(payload.range)) return actions.NO_OP_RESULT;
    slice.push(payload.range);
    return {
      inverse: [removeRange({ axisKey: axis, range: payload.range })],
      targets: [state.key],
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
      targets: [state.key],
    };
  },

  setAxis: (state, payload) => {
    const oldAxis = actions.snapshotDraft(state.axes[payload.axis.key]);
    state.axes[payload.axis.key] = payload.axis;
    return { inverse: [setAxis({ axis: oldAxis })], targets: [state.key] };
  },

  // setLine has upsert semantics. When the line is new there is no
  // removeLine action to revert with, so the inverse is empty and undo of
  // setLine on a fresh line is a no-op. Lines are derived from channel and
  // range bindings so they are typically created and removed by the channel
  // and range actions rather than directly.
  setLine: (state, payload) => {
    const idx = state.lines.findIndex((l) => l.key === payload.line.key);
    if (idx === -1) {
      state.lines.push(payload.line);
      return { inverse: [], targets: [state.key] };
    }
    const oldLine = actions.snapshotDraft(state.lines[idx]);
    state.lines[idx] = payload.line;
    return { inverse: [setLine({ line: oldLine })], targets: [state.key] };
  },

  setRule: (state, payload) => {
    const idx = state.rules.findIndex((r) => r.key === payload.rule.key);
    if (idx === -1) {
      state.rules.push(payload.rule);
      return {
        inverse: [removeRule({ key: payload.rule.key })],
        targets: [state.key],
      };
    }
    const oldRule = actions.snapshotDraft(state.rules[idx]);
    state.rules[idx] = payload.rule;
    return { inverse: [setRule({ rule: oldRule })], targets: [state.key] };
  },

  removeRule: (state, payload) => {
    const idx = state.rules.findIndex((r) => r.key === payload.key);
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldRule = actions.snapshotDraft(state.rules[idx]);
    state.rules.splice(idx, 1);
    return { inverse: [setRule({ rule: oldRule })], targets: [state.key] };
  },
};

export const reduceAll = createReduceAll(handlers);
