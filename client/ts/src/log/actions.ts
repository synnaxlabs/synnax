// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type patch } from "@synnaxlabs/x";
import { type Draft } from "immer";

import { actions } from "@/actions";
import {
  type Action,
  addChannel,
  createReduceAll,
  type HandlerResult,
  type Handlers,
  removeChannel,
  setChannelAlias,
  setChannelColor,
  setChannelEntry,
  setChannelNamesHidden,
  setChannelNotation,
  setChannelPrecision,
  setChannelTimestampFormat,
  setChannelTimestampTimeZone,
  setTimestampPrecision,
  setTimestampsHidden,
  setTimeZone,
  swapChannel,
} from "@/log/actions.gen";
import { channelEntryZ, type Log, logZ } from "@/log/types.gen";

const applyPath = (state: Draft<Log>, action: patch.Action): HandlerResult =>
  actions.applyPath(state, action, logZ);

// Handlers report the narrowest resource each action touches as its target. Log-level
// config (timestamp precision, display flags) targets the log key itself, and a path
// action targets its path. Channel mutations target the specific channel so concurrent
// edits to distinct channels neither coalesce nor invalidate each other's undoables.
const handlers: Handlers = {
  create: (state, payload) => {
    Object.assign(state, payload.log);
    return { inverse: [], targets: [payload.log.key] };
  },

  addChannel: (state, payload) => {
    if (state.channels.some((e) => e.channel === payload.channel))
      return actions.NO_OP_RESULT;
    state.channels.push(channelEntryZ.parse({ channel: payload.channel }));
    return {
      inverse: [removeChannel({ channel: payload.channel })],
      targets: [`channel:${payload.channel}`],
    };
  },

  removeChannel: (state, payload) => {
    const idx = state.channels.findIndex((e) => e.channel === payload.channel);
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldEntry = actions.snapshotDraft(state.channels[idx]);
    state.channels.splice(idx, 1);
    return {
      inverse: [
        addChannel({ channel: payload.channel }),
        setChannelEntry({ entry: oldEntry }),
      ],
      targets: [`channel:${payload.channel}`],
    };
  },

  setChannelEntry: (state, payload) => {
    const idx = state.channels.findIndex((e) => e.channel === payload.entry.channel);
    if (idx === -1) {
      state.channels.push(payload.entry);
      return {
        inverse: [removeChannel({ channel: payload.entry.channel })],
        targets: [`channel:${payload.entry.channel}`],
      };
    }
    const oldEntry = actions.snapshotDraft(state.channels[idx]);
    state.channels[idx] = payload.entry;
    return {
      inverse: [setChannelEntry({ entry: oldEntry })],
      targets: [`channel:${payload.entry.channel}`],
    };
  },

  setChannelColor: (state, payload) => {
    const entry = state.channels.find((e) => e.channel === payload.channel);
    if (entry == null) return actions.NO_OP_RESULT;
    const inverse = [
      setChannelColor({
        channel: payload.channel,
        color: actions.snapshotDraft(entry.color),
      }),
    ];
    entry.color = payload.color;
    return { inverse, targets: [`channel:${payload.channel}`] };
  },

  setChannelNotation: (state, payload) => {
    const entry = state.channels.find((e) => e.channel === payload.channel);
    if (entry == null) return actions.NO_OP_RESULT;
    const inverse = [
      setChannelNotation({ channel: payload.channel, notation: entry.notation }),
    ];
    entry.notation = payload.notation;
    return { inverse, targets: [`channel:${payload.channel}`] };
  },

  setChannelPrecision: (state, payload) => {
    const entry = state.channels.find((e) => e.channel === payload.channel);
    if (entry == null) return actions.NO_OP_RESULT;
    const inverse = [
      setChannelPrecision({ channel: payload.channel, precision: entry.precision }),
    ];
    entry.precision = payload.precision;
    return { inverse, targets: [`channel:${payload.channel}`] };
  },

  setChannelAlias: (state, payload) => {
    const entry = state.channels.find((e) => e.channel === payload.channel);
    if (entry == null) return actions.NO_OP_RESULT;
    const inverse = [setChannelAlias({ channel: payload.channel, alias: entry.alias })];
    entry.alias = payload.alias;
    return { inverse, targets: [`channel:${payload.channel}`] };
  },

  setChannelTimestampFormat: (state, payload) => {
    const entry = state.channels.find((e) => e.channel === payload.channel);
    if (entry == null) return actions.NO_OP_RESULT;
    const inverse = [
      setChannelTimestampFormat({
        channel: payload.channel,
        format: entry.timestampConfig.format,
      }),
    ];
    entry.timestampConfig.format = payload.format;
    return { inverse, targets: [`channel:${payload.channel}`] };
  },

  setChannelTimestampTimeZone: (state, payload) => {
    const entry = state.channels.find((e) => e.channel === payload.channel);
    if (entry == null) return actions.NO_OP_RESULT;
    const inverse = [
      setChannelTimestampTimeZone({
        channel: payload.channel,
        timeZone: entry.timestampConfig.timeZone,
      }),
    ];
    entry.timestampConfig.timeZone = payload.timeZone;
    return { inverse, targets: [`channel:${payload.channel}`] };
  },

  swapChannel: (state, payload) => {
    const idx = state.channels.findIndex((e) => e.channel === payload.from);
    if (idx === -1 || state.channels.some((e) => e.channel === payload.to))
      return actions.NO_OP_RESULT;
    state.channels[idx].channel = payload.to;
    return {
      inverse: [swapChannel({ from: payload.to, to: payload.from })],
      targets: [`channel:${payload.from}`, `channel:${payload.to}`],
    };
  },

  setTimestampPrecision: (state, payload) => {
    const oldPrecision = state.timestampPrecision;
    state.timestampPrecision = payload.timestampPrecision;
    return {
      inverse: [setTimestampPrecision({ timestampPrecision: oldPrecision })],
      targets: [state.key],
    };
  },

  setChannelNamesHidden: (state, payload) => {
    const oldValue = state.channelNamesHidden;
    state.channelNamesHidden = payload.channelNamesHidden;
    return {
      inverse: [setChannelNamesHidden({ channelNamesHidden: oldValue })],
      targets: [state.key],
    };
  },

  setTimestampsHidden: (state, payload) => {
    const oldValue = state.timestampsHidden;
    state.timestampsHidden = payload.timestampsHidden;
    return {
      inverse: [setTimestampsHidden({ timestampsHidden: oldValue })],
      targets: [state.key],
    };
  },

  setTimeZone: (state, payload) => {
    const oldTimeZone = state.timeZone;
    state.timeZone = payload.timeZone;
    return {
      inverse: [setTimeZone({ timeZone: oldTimeZone })],
      targets: [state.key],
    };
  },
  ...actions.createPathHandlers(applyPath),
};

export const reduceAll = createReduceAll(handlers);

// createOf hands the dispatch controller the document carried by a create
// action so frames for never-cached documents ingest instead of drop.
export const createOf = (action: Action) =>
  action.type === "create" ? action.create.log : undefined;

// kindOf keys single-channel edits by channel so rapid edits to one
// channel's config coalesce into a single undoable, while edits to distinct
// channels stay separate. Log-level edits (name, precision, flags) key by their
// action type; multi-action batches collapse into one "transaction" undoable.
export const kindOf = (actions: Action[]): string => {
  if (actions.length === 0) return "default";
  if (actions.length > 1) return "transaction";
  const a = actions[0];
  switch (a.type) {
    case "add_channel":
      return `channel:${a.addChannel.channel}`;
    case "remove_channel":
      return `channel:${a.removeChannel.channel}`;
    case "set_channel_entry":
      return `channel:${a.setChannelEntry.entry.channel}`;
    case "set_channel_color":
      return `channel:${a.setChannelColor.channel}`;
    case "set_channel_notation":
      return `channel:${a.setChannelNotation.channel}`;
    case "set_channel_precision":
      return `channel:${a.setChannelPrecision.channel}`;
    case "set_channel_alias":
      return `channel:${a.setChannelAlias.channel}`;
    case "set_channel_timestamp_format":
      return `channel:${a.setChannelTimestampFormat.channel}`;
    case "set_channel_timestamp_time_zone":
      return `channel:${a.setChannelTimestampTimeZone.channel}`;
    default:
      return a.type;
  }
};
