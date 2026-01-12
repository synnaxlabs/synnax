// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Viewport } from "@synnaxlabs/pluto";
import { migrate, uuid } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/schematic/types/v1";

export const VERSION = "2.0.0";

export const stateZ = v1.stateZ.omit({ version: true }).extend({
  version: z.literal(VERSION),
  key: z.string(),
  type: z.literal("schematic"),
  viewportMode: Viewport.modeZ.default("select"),
});
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...v1.ZERO_STATE,
  version: VERSION,
  key: "",
  type: "schematic",
  viewportMode: "select",
};

export const sliceStateZ = v1.sliceStateZ
  .omit({ version: true, schematics: true })
  .extend({ version: z.literal(VERSION), schematics: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  ...v1.ZERO_SLICE_STATE,
  version: VERSION,
  schematics: {},
};

export const stateMigration = migrate.createMigration<v1.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: VERSION,
    key: uuid.create(),
    type: "schematic",
    viewportMode: "select",
  }),
});

export const sliceMigration = migrate.createMigration<v1.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ schematics, ...rest }) => ({
    ...rest,
    schematics: Object.fromEntries(
      Object.entries(schematics).map(([key, schematic]) => [
        key,
        { ...stateMigration(schematic), key },
      ]),
    ),
    version: VERSION,
  }),
});
