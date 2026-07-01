// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, log } from "@synnaxlabs/client";
import { color, migrate, notation, telem } from "@synnaxlabs/x";
import { z } from "zod";

const STATE_MIGRATION_NAME = "log.state";

const V0_VERSION = "0.0.0";

const v0StateZ = z.object({
  key: z.string(),
  version: z.literal(V0_VERSION),
  channels: channel.keyZ.array(),
  remoteCreated: z.boolean(),
});
interface V0State extends z.infer<typeof v0StateZ> {}

const V1_VERSION = "1.0.0";

const v1TimestampConfigZ = z.object({
  format: telem.timestampFormatZ.default("preciseDate"),
  tz: telem.timeZoneZ.default("local"),
});

const v1ChannelEntryZ = z.object({
  channel: channel.keyZ,
  color: z.string().default(""),
  notation: notation.notationZ.default("standard"),
  precision: z.number().min(-1).max(17).default(-1),
  alias: z.string().default(""),
  timestamp: v1TimestampConfigZ.default({ format: "preciseDate", tz: "local" }),
});

const v1ToolbarStateZ = z.object({
  activeTab: z.enum(["channels", "properties"]).default("channels"),
});
const V1_ZERO_TOOLBAR_STATE = { activeTab: "channels" as const };

const v1StateZ = z.object({
  key: z.string(),
  version: z.literal(V1_VERSION),
  channels: z.array(v1ChannelEntryZ).default([]),
  remoteCreated: z.boolean(),
  timestampPrecision: z.number().min(0).max(3).default(0),
  showChannelNames: z.boolean().default(true),
  showReceiptTimestamp: z.boolean().default(true),
  toolbar: v1ToolbarStateZ.default(V1_ZERO_TOOLBAR_STATE),
});
interface V1State extends z.infer<typeof v1StateZ> {}

const v1StateMigration = migrate.createMigration<V0State, V1State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    key: state.key,
    version: V1_VERSION,
    remoteCreated: state.remoteCreated,
    timestampPrecision: 0,
    showChannelNames: true,
    showReceiptTimestamp: true,
    toolbar: V1_ZERO_TOOLBAR_STATE,
    channels: state.channels.map((key) => v1ChannelEntryZ.parse({ channel: key })),
  }),
});

const V2_VERSION = "2.0.0";

const v2PendingUploadZ = log.logZ.omit({ name: true });
interface PendingUpload extends z.infer<typeof v2PendingUploadZ> {}

const v2StateZ = z.object({
  key: z.string(),
  version: z.literal(V2_VERSION),
  toolbar: v1ToolbarStateZ.default(V1_ZERO_TOOLBAR_STATE),
  pendingUpload: v2PendingUploadZ.optional(),
});
interface V2State extends z.infer<typeof v2StateZ> {}

const v2ZeroState: V2State = {
  key: "",
  version: V2_VERSION,
  toolbar: V1_ZERO_TOOLBAR_STATE,
  pendingUpload: undefined,
};

// migrateColor converts the loose v1 color representation - a possibly-empty hex string
// - into a strongly-typed color. The empty string (the v1 "unset" sentinel) and any
// value that does not parse fall back to color.ZERO.
const migrateColor = (crude: string): color.Color => {
  const res = color.colorZ.safeParse(crude);
  return res.success ? res.data : color.ZERO;
};

const buildPendingUpload = (state: V1State): PendingUpload => ({
  key: state.key,
  channels: state.channels.map((entry) => ({
    ...entry,
    color: migrateColor(entry.color),
  })),
  hideChannelNames: !state.showChannelNames,
  hideReceiptTimestamp: !state.showReceiptTimestamp,
  timestampPrecision: state.timestampPrecision,
});

// Parks v1's body in pendingUpload when the log is not yet remoteCreated. remoteCreated
// logs already have authoritative data on the server, so pendingUpload stays undefined
// for those.
const v2StateMigration = migrate.createMigration<V1State, V2State>({
  name: STATE_MIGRATION_NAME,
  migrate: (state) => ({
    key: state.key,
    version: V2_VERSION,
    toolbar: state.toolbar,
    pendingUpload: state.remoteCreated ? undefined : buildPendingUpload(state),
  }),
});

type AnyState = V0State | V1State | V2State;

const STATE_MIGRATIONS: migrate.Migrations = {
  [V0_VERSION]: v1StateMigration,
  [V1_VERSION]: v2StateMigration,
};

const migrateState = migrate.migrator<AnyState, V2State>({
  name: STATE_MIGRATION_NAME,
  migrations: STATE_MIGRATIONS,
  def: v2ZeroState,
});

export const anyStateZ = z
  .union([v2StateZ, v1StateZ, v0StateZ])
  .transform((state) => migrateState(state));

export const parseImport = (
  data: unknown,
  fallbackName: string | undefined,
): log.New => {
  // Legacy console-state exports are tried first: forcing remoteCreated false makes the
  // migration ladder park the body in pendingUpload regardless of the source's sync
  // state. A current typed export carries a name and parses as logZ; it falls through
  // because its body fields are stripped by the console-state schema, leaving no
  // pendingUpload.
  if (typeof data === "object" && data != null) {
    const legacy = anyStateZ.safeParse({ ...data, remoteCreated: false });
    if (legacy.success && legacy.data.pendingUpload != null) {
      const { key: _key, ...body } = legacy.data.pendingUpload;
      return { ...body, name: fallbackName ?? "Log" };
    }
  }
  const { key: _key, ...rest } = log.logZ.parse(data);
  return { ...rest, name: fallbackName ?? rest.name };
};
