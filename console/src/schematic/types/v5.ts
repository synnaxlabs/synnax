// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Viewport } from "@synnaxlabs/pluto";
import { migrate, record } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/schematic/types/v0";
import * as v1 from "@/schematic/types/v1";
import * as v4 from "@/schematic/types/v4";

export const VERSION = "5.0.0";

export const stateZ = v4.stateZ.omit({ version: true, type: true }).extend({
  version: z.literal(VERSION),
  mode: Viewport.modeZ,
  toolbar: v0.toolbarStateZ,
});
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...record.omit(v4.ZERO_STATE, "type"),
  version: VERSION,
  mode: v4.ZERO_SLICE_STATE.mode,
  toolbar: v0.ZERO_TOOLBAR_STATE,
};

export const sliceStateZ = v4.sliceStateZ
  .omit({ version: true, schematics: true, mode: true, toolbar: true })
  .extend({ version: z.literal(VERSION), schematics: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  copy: v4.ZERO_SLICE_STATE.copy,
  version: VERSION,
  schematics: {},
};

export const stateMigration = migrate.createMigration<v4.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...record.omit(state, "type"),
    version: VERSION,
    mode: v4.ZERO_SLICE_STATE.mode,
    toolbar: { ...v4.ZERO_SLICE_STATE.toolbar },
  }),
});

export const sliceMigration = migrate.createMigration<v4.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ schematics, mode, toolbar, ...rest }) => ({
    ...rest,
    schematics: Object.fromEntries(
      Object.entries(schematics).map(([key, schematic]) => [
        key,
        { ...stateMigration(schematic), mode, toolbar },
      ]),
    ),
    version: VERSION,
  }),
});
