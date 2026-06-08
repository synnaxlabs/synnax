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

import * as v1 from "@/log/types/v1";

export const VERSION = "2.0.0";

export const toolbarTabZ = v1.toolbarTabZ;
export type ToolbarTab = v1.ToolbarTab;
export const toolbarStateZ = v1.toolbarStateZ;
export type ToolbarState = v1.ToolbarState;
export const ZERO_TOOLBAR_STATE = v1.ZERO_TOOLBAR_STATE;

export const ZERO_CHANNEL_ENTRY: log.ChannelEntry = {
  channel: 0,
  color: color.ZERO,
  notation: "standard",
  precision: -1,
  alias: "",
  timestamp: { format: "preciseDate", tz: "local" },
};

// pendingUploadZ is the typed log body needed to upload a not-yet-synced log on first
// render. name is omitted because the live name lives in Layout.
export const pendingUploadZ = log.logZ.omit({ name: true });
export interface PendingUpload extends z.infer<typeof pendingUploadZ> {}

// v2 removes the document body (channels, timestamp precision, display flags) from
// Console state; those fields are owned by the Pluto flux store and round-trip to the
// server. The slice keeps only UI state (the toolbar tab). Legacy state that was never
// synced to the server has its body parked in pendingUpload so it can be re-created on
// first render.
export const stateZ = z.object({
  key: z.string(),
  version: z.literal(VERSION),
  toolbar: toolbarStateZ.default(ZERO_TOOLBAR_STATE),
  pendingUpload: pendingUploadZ.optional(),
});
export type State = z.infer<typeof stateZ>;

export const ZERO_STATE: State = {
  key: "",
  version: VERSION,
  toolbar: ZERO_TOOLBAR_STATE,
  pendingUpload: undefined,
};

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  logs: z.record(z.string(), stateZ),
});
export type SliceState = z.infer<typeof sliceStateZ>;
export const ZERO_SLICE_STATE: SliceState = { version: VERSION, logs: {} };

// migrateColor converts the loose v1 color representation - a possibly-empty hex string
// - into a strongly-typed color. The empty string (the v1 "unset" sentinel) and any
// value that does not parse fall back to color.ZERO.
const migrateColor = (crude: string): color.Color => {
  const res = color.colorZ.safeParse(crude);
  return res.success ? res.data : color.ZERO;
};

const buildPendingUpload = (state: v1.State): PendingUpload => ({
  key: state.key,
  channels: state.channels.map((entry) => ({
    ...entry,
    color: migrateColor(entry.color),
  })),
  remoteCreated: state.remoteCreated,
  timestampPrecision: state.timestampPrecision,
  showChannelNames: state.showChannelNames,
  showReceiptTimestamp: state.showReceiptTimestamp,
});

// Parks v1's body in pendingUpload when the log is not yet remoteCreated. remoteCreated
// logs already have authoritative data on the server, so pendingUpload stays undefined
// for those.
export const stateMigration = migrate.createMigration<v1.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    key: state.key,
    version: VERSION,
    toolbar: state.toolbar,
    pendingUpload: state.remoteCreated ? undefined : buildPendingUpload(state),
  }),
});

export const sliceMigration = migrate.createMigration<v1.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ logs }) => ({
    version: VERSION,
    logs: Object.fromEntries(
      Object.entries(logs).map(([key, logState]) => [key, stateMigration(logState)]),
    ),
  }),
});
