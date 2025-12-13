// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate, record } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/schematic/types/v1";
import * as v5 from "@/schematic/types/v5";

export const VERSION = "6.0.0";

// Remove controlAcquireTrigger - control is now managed via RPC methods
export const stateZ = v5.stateZ
  .omit({ version: true, controlAcquireTrigger: true })
  .extend({ version: z.literal(VERSION) });
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...record.omit(v5.ZERO_STATE, "controlAcquireTrigger"),
  version: VERSION,
};

export const sliceStateZ = v5.sliceStateZ
  .omit({ version: true, schematics: true })
  .extend({ version: z.literal(VERSION), schematics: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  copy: v5.ZERO_SLICE_STATE.copy,
  version: VERSION,
  schematics: {},
};

export const stateMigration = migrate.createMigration<v5.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...record.omit(state, "controlAcquireTrigger"),
    version: VERSION,
  }),
});

export const sliceMigration = migrate.createMigration<v5.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ schematics, ...rest }) => ({
    ...rest,
    schematics: Object.fromEntries(
      Object.entries(schematics).map(([key, schematic]) => [
        key,
        stateMigration(schematic),
      ]),
    ),
    version: VERSION,
  }),
});
