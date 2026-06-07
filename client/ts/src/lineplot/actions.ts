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

  setLegend: (state, payload) => {
    const oldLegend = actions.snapshotDraft(state.legend);
    state.legend = payload.legend;
    return { inverse: [setLegend({ legend: oldLegend })], targets: [state.key] };
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

  setAxis: (state, payload) => {
    const oldAxis = actions.snapshotDraft(state.axes[payload.axis.key]);
    state.axes[payload.axis.key] = payload.axis;
    return {
      inverse: [setAxis({ axis: oldAxis })],
      targets: [`axis:${payload.axis.key}`],
    };
  },

  // setLine edits the styling of an existing line. Lines are materialized by
  // addChannel/addRange/setXChannel, so the insert branch exists only to
  // restore a line removed by those handlers' inverses; such inserts are not
  // independently undoable (empty inverse and targets).
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
