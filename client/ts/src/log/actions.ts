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
import { type channel } from "@/channel";
import {
  addChannel,
  createReduceAll,
  type Handlers,
  removeChannel,
  rename,
  setChannelEntry,
  setChannels,
  setShowChannelNames,
  setShowReceiptTimestamp,
  setTimestampPrecision,
} from "@/log/actions.gen";
import { type ChannelEntry } from "@/log/types.gen";

// defaultChannelEntry builds the entry a freshly added channel receives. It must
// stay in sync with the field defaults declared on ChannelEntry in
// schemas/log.oracle and with defaultChannelEntry in
// core/pkg/service/log/actions.go. Color is left at the zero value; the rendering
// layer resolves a palette color for entries without an explicit color.
const defaultChannelEntry = (key: channel.Key): ChannelEntry => ({
  channel: key,
  color: color.ZERO,
  notation: "standard",
  precision: -1,
  alias: "",
  timestamp: { format: "preciseDate", tz: "local" },
});

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
    state.channels.push(defaultChannelEntry(payload.channel));
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

  setChannels: (state, payload) => {
    const oldChannels = actions.snapshotDraft(state.channels);
    const targets = new Set<string>();
    oldChannels.forEach((e) => targets.add(`channel:${e.channel}`));
    payload.channels.forEach((e) => targets.add(`channel:${e.channel}`));
    state.channels = payload.channels;
    return { inverse: [setChannels({ channels: oldChannels })], targets: [...targets] };
  },

  setTimestampPrecision: (state, payload) => {
    const oldPrecision = state.timestampPrecision;
    state.timestampPrecision = payload.timestampPrecision;
    return {
      inverse: [setTimestampPrecision({ timestampPrecision: oldPrecision })],
      targets: [state.key],
    };
  },

  setShowChannelNames: (state, payload) => {
    const oldValue = state.showChannelNames;
    state.showChannelNames = payload.showChannelNames;
    return {
      inverse: [setShowChannelNames({ showChannelNames: oldValue })],
      targets: [state.key],
    };
  },

  setShowReceiptTimestamp: (state, payload) => {
    const oldValue = state.showReceiptTimestamp;
    state.showReceiptTimestamp = payload.showReceiptTimestamp;
    return {
      inverse: [setShowReceiptTimestamp({ showReceiptTimestamp: oldValue })],
      targets: [state.key],
    };
  },
};

export const reduceAll = createReduceAll(handlers);
