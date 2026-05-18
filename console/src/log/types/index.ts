// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/log/types/v0";
import * as v1 from "@/log/types/v1";

export const stateZ = v1.stateZ;
export type State = v1.State;
export type SliceState = v1.SliceState;
export const ZERO_SLICE_STATE = v1.ZERO_SLICE_STATE;
export const ZERO_STATE = v1.ZERO_STATE;
export type ChannelConfig = v1.ChannelConfig;
export const ZERO_CHANNEL_CONFIG = v1.ZERO_CHANNEL_CONFIG;
export type ChannelEntry = v1.ChannelEntry;
export const ZERO_CHANNEL_ENTRY = v1.ZERO_CHANNEL_ENTRY;
export const channelEntryZ = v1.channelEntryZ;
export type ToolbarTab = v1.ToolbarTab;
export type ToolbarState = v1.ToolbarState;
export const ZERO_TOOLBAR_STATE = v1.ZERO_TOOLBAR_STATE;

export type AnyState = v0.State | v1.State;
export type AnySliceState = v0.SliceState | v1.SliceState;

export const STATE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.stateMigration,
};

export const migrateState = migrate.migrator<AnyState, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrations: STATE_MIGRATIONS,
  def: ZERO_STATE,
});

export const SLICE_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.sliceMigration,
};

export const migrateSlice = migrate.migrator<AnySliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrations: SLICE_MIGRATIONS,
  def: ZERO_SLICE_STATE,
});

export const anyStateZ = z
  .union([v1.stateZ, v0.stateZ])
  .transform((state) => migrateState(state));
