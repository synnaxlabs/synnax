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

import * as v1 from "@/schematic/types/v1";
import * as v3 from "@/schematic/types/v3";

export const VERSION = "4.0.0";

export const stateZ = v3.stateZ
  .omit({ version: true })
  .extend({ version: z.literal(VERSION), authority: z.number() });
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = { ...v3.ZERO_STATE, version: VERSION, authority: 1 };

export const sliceStateZ = v3.sliceStateZ
  .omit({ version: true, schematics: true })
  .extend({ version: z.literal(VERSION), schematics: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  ...v3.ZERO_SLICE_STATE,
  version: VERSION,
  schematics: {},
};

export const stateMigration = migrate.createMigration<v3.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({ ...state, version: VERSION, authority: 1 }),
});

export const sliceMigration = migrate.createMigration<v3.SliceState, SliceState>({
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
