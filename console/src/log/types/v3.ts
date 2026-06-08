// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/log/types/v1";
import * as v2 from "@/log/types/v2";

export const VERSION = "3.0.0";

export const toolbarTabZ = v2.toolbarTabZ;
export type ToolbarTab = v2.ToolbarTab;
export const toolbarStateZ = v2.toolbarStateZ;
export type ToolbarState = v2.ToolbarState;
export const ZERO_TOOLBAR_STATE = v2.ZERO_TOOLBAR_STATE;
export const ZERO_CHANNEL_ENTRY = v2.ZERO_CHANNEL_ENTRY;

// pendingUploadZ is the log body needed to upload a not-yet-synced log on first
// render. name is omitted because the live name lives in Layout.
export const pendingUploadZ = log.logZ.omit({ name: true });
export interface PendingUpload extends z.infer<typeof pendingUploadZ> {}

// v3 removes the document body (channels, timestamp precision, display flags) from
// Console state; those fields are owned by the Pluto flux store and round-trip to
// the server. The slice keeps only UI state (the toolbar tab). Legacy state that
// was never synced to the server has its body parked in pendingUpload so it can be
// re-created on first render.
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

const buildPendingUpload = (state: v2.State): PendingUpload => ({
  key: state.key,
  channels: state.channels,
  remoteCreated: state.remoteCreated,
  timestampPrecision: state.timestampPrecision,
  showChannelNames: state.showChannelNames,
  showReceiptTimestamp: state.showReceiptTimestamp,
});

// Parks v2's body in pendingUpload when the log is not yet remoteCreated.
// remoteCreated logs already have authoritative data on the server, so
// pendingUpload stays undefined for those.
export const stateMigration = migrate.createMigration<v2.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    key: state.key,
    version: VERSION,
    toolbar: state.toolbar,
    pendingUpload: state.remoteCreated ? undefined : buildPendingUpload(state),
  }),
});

export const sliceMigration = migrate.createMigration<v2.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ logs }) => ({
    version: VERSION,
    logs: Object.fromEntries(
      Object.entries(logs).map(([k, l]) => [k, stateMigration(l)]),
    ),
  }),
});
