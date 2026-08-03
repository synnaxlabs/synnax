// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

import { actions } from "@/actions";
import {
  addChannel,
  createReduceAll,
  type Handlers,
  removeChannel,
  rename,
  setChannelAlias,
  setChannelColor,
  setChannelEntry,
  setChannelNotation,
  setChannelPrecision,
  setChannels,
  setChannelTimestampFormat,
  setChannelTimestampTz,
  setHideChannelNames,
  setHideReceiptTimestamp,
  setTimestampPrecision,
  swapChannel,
} from "@/log/actions.gen";
import { channelEntryZ } from "@/log/types.gen";

// Handlers report the narrowest resource each action touches as its target. Log-level
// config (name, timestamp precision, display flags) targets the log key itself; channel
// mutations target the specific channel so concurrent edits to distinct channels
// neither coalesce nor invalidate each other's undoables.
const handlers: Handlers = {
  rename: (state, payload) => {
    const oldName = state.name;
    state.name = payload.name;
    return { inverse: [rename({ name: oldName })], targets: [state.key] };
  },

  addChannel: (state, payload) => {
    if (state.channels.some((e) => e.channel === payload.channel))
      return actions.NO_OP_RESULT;
    state.channels.push(
      channelEntryZ.parse({ channel: payload.channel, color: color.ZERO }),
    );
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
        format: entry.timestamp.format,
      }),
    ];
    entry.timestamp.format = payload.format;
    return { inverse, targets: [`channel:${payload.channel}`] };
  },

  setChannelTimestampTz: (state, payload) => {
    const entry = state.channels.find((e) => e.channel === payload.channel);
    if (entry == null) return actions.NO_OP_RESULT;
    const inverse = [
      setChannelTimestampTz({ channel: payload.channel, tz: entry.timestamp.tz }),
    ];
    entry.timestamp.tz = payload.tz;
    return { inverse, targets: [`channel:${payload.channel}`] };
  },

  setChannels: (state, payload) => {
    const oldChannels = actions.snapshotDraft(state.channels);
    const targets = new Set<string>();
    oldChannels.forEach((e) => targets.add(`channel:${e.channel}`));
    payload.channels.forEach((e) => targets.add(`channel:${e.channel}`));
    state.channels = payload.channels;
    return { inverse: [setChannels({ channels: oldChannels })], targets: [...targets] };
  },

  swapChannel: (state, payload) => {
    const idx = state.channels.findIndex((e) => e.channel === payload.from);
    if (idx === -1) return actions.NO_OP_RESULT;
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

  setHideChannelNames: (state, payload) => {
    const oldValue = state.hideChannelNames;
    state.hideChannelNames = payload.hideChannelNames;
    return {
      inverse: [setHideChannelNames({ hideChannelNames: oldValue })],
      targets: [state.key],
    };
  },

  setHideReceiptTimestamp: (state, payload) => {
    const oldValue = state.hideReceiptTimestamp;
    state.hideReceiptTimestamp = payload.hideReceiptTimestamp;
    return {
      inverse: [setHideReceiptTimestamp({ hideReceiptTimestamp: oldValue })],
      targets: [state.key],
    };
  },
};

export const reduceAll = createReduceAll(handlers);
