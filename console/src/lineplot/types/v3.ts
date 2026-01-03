// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Viewport } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/lineplot/types/v0";
import * as v1 from "@/lineplot/types/v1";
import * as v2 from "@/lineplot/types/v2";

export const VERSION = "3.0.0";

export const stateZ = v2.stateZ.omit({ version: true }).extend({
  version: z.literal(VERSION),
  mode: Viewport.modeZ,
  control: v0.controlStateZ,
  toolbar: v0.toolbarStateZ,
});
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...v2.ZERO_STATE,
  version: VERSION,
  mode: v2.ZERO_SLICE_STATE.mode,
  control: v2.ZERO_SLICE_STATE.control,
  toolbar: v2.ZERO_SLICE_STATE.toolbar,
};

export const sliceStateZ = v2.sliceStateZ
  .omit({ plots: true, version: true, mode: true, control: true, toolbar: true })
  .extend({ version: z.literal(VERSION), plots: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  plots: {},
};

export const stateMigration = migrate.createMigration<v2.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: VERSION,
    mode: v2.ZERO_SLICE_STATE.mode,
    control: v2.ZERO_SLICE_STATE.control,
    toolbar: v2.ZERO_SLICE_STATE.toolbar,
  }),
});

export const sliceMigration = migrate.createMigration<v2.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ plots, mode, control, toolbar }) => ({
    version: VERSION,
    plots: Object.fromEntries(
      Object.entries(plots).map(([key, plot]) => [
        key,
        { ...stateMigration(plot), mode, control, toolbar },
      ]),
    ),
  }),
});
