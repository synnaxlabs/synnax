// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Log } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import type * as v0 from "@/log/types/v0";

export const VERSION = "1.0.0";

export const { channelConfigZ } = Log;
export type ChannelConfig = z.infer<typeof Log.channelConfigZ>;

export const ZERO_CHANNEL_CONFIG: ChannelConfig = {
  color: "",
  notation: "standard",
  precision: -1,
  alias: "",
};

export const channelEntryZ = channelConfigZ.extend({
  channel: channel.keyZ,
});
export type ChannelEntry = z.infer<typeof channelEntryZ>;

export const ZERO_CHANNEL_ENTRY: ChannelEntry = {
  ...ZERO_CHANNEL_CONFIG,
  channel: 0,
};

export const stateZ = z.object({
  key: z.string(),
  version: z.literal(VERSION),
  channels: z.array(channelEntryZ).default([]),
  remoteCreated: z.boolean(),
  timestampPrecision: z.number().min(0).max(3).default(0),
  showChannelNames: z.boolean().default(true),
  showReceiptTimestamp: z.boolean().default(true),
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
};

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
    channels: state.channels.map((key) => ({
      channel: key,
      ...ZERO_CHANNEL_CONFIG,
    })),
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
