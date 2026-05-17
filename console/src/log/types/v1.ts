// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { color, migrate } from "@synnaxlabs/x";
import { z } from "zod";

import type * as v0 from "@/log/types/v0";

export const VERSION = "1.0.0";

export const ZERO_CHANNEL_ENTRY: log.ChannelEntry = {
  channel: 0,
  color: color.ZERO,
  notation: "standard",
  precision: -1,
  alias: "",
  timestamp: { format: "preciseDate", tz: "local" },
};

export const toolbarTabZ = z.enum(["channels", "properties"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export const toolbarStateZ = z.object({
  activeTab: toolbarTabZ.default("channels"),
});
export type ToolbarState = z.infer<typeof toolbarStateZ>;

export const ZERO_TOOLBAR_STATE: ToolbarState = { activeTab: "channels" };

export const stateZ = z.object({
  key: z.string(),
  version: z.literal(VERSION),
  channels: z.array(log.channelEntryZ).default([]),
  remoteCreated: z.boolean(),
  timestampPrecision: z.number().min(0).max(3).default(0),
  showChannelNames: z.boolean().default(true),
  showReceiptTimestamp: z.boolean().default(true),
  toolbar: toolbarStateZ.default(ZERO_TOOLBAR_STATE),
});
export type State = z.infer<typeof stateZ>;

export const ZERO_STATE: State = {
  key: "",
  version: VERSION,
  channels: [],
  remoteCreated: false,
  timestampPrecision: 0,
  showChannelNames: true,
  showReceiptTimestamp: true,
  toolbar: ZERO_TOOLBAR_STATE,
};

// stateFromLog projects a typed Log retrieved from the server into the Console
// state shape, layering in the version stamp and the default UI-only toolbar.
// The Log resource carries no Console-specific UI state (toolbar, persisted
// state version); those are reset to defaults on load.
export const stateFromLog = (l: log.Log): State => ({
  key: l.key,
  version: VERSION,
  channels: l.channels,
  remoteCreated: l.remoteCreated,
  timestampPrecision: l.timestampPrecision,
  showChannelNames: l.showChannelNames,
  showReceiptTimestamp: l.showReceiptTimestamp,
  toolbar: ZERO_TOOLBAR_STATE,
});

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  logs: z.record(z.string(), stateZ),
});
export type SliceState = z.infer<typeof sliceStateZ>;
export const ZERO_SLICE_STATE: SliceState = { version: VERSION, logs: {} };

export const STATE_MIGRATION_NAME = "log.state";
export const SLICE_MIGRATION_NAME = "log.slice";

export const stateMigration = migrate.createMigration<v0.State, State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    key: state.key,
    version: VERSION,
    remoteCreated: state.remoteCreated,
    timestampPrecision: 0,
    showChannelNames: true,
    showReceiptTimestamp: true,
    toolbar: ZERO_TOOLBAR_STATE,
    channels: state.channels.map((key) => ({ ...ZERO_CHANNEL_ENTRY, channel: key })),
  }),
});

export const sliceMigration = migrate.createMigration<v0.SliceState, SliceState>({
  name: SLICE_MIGRATION_NAME,
  migrate: ({ logs }) => ({
    version: VERSION,
    logs: Object.fromEntries(
      Object.entries(logs).map(([key, log]) => [key, stateMigration(log)]),
    ),
  }),
});
