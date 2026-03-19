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

import * as v1 from "@/lineplot/types/v1";
import * as v5 from "@/lineplot/types/v5";

export const VERSION = "6.0.0";

export const stateZ = v5.stateZ.omit({ version: true }).extend({
  version: z.literal(VERSION),
  align: z.boolean().default(false),
});
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...v5.ZERO_STATE,
  version: VERSION,
  align: false,
};

export const sliceStateZ = v5.sliceStateZ
  .omit({ plots: true, version: true })
  .extend({ version: z.literal(VERSION), plots: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  plots: {},
};

export const stateMigration = migrate.createMigration<v5.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: VERSION,
    align: false,
  }),
});

export const sliceMigration = migrate.createMigration<v5.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ plots }) => ({
    version: VERSION,
    plots: Object.fromEntries(
      Object.entries(plots).map(([key, plot]) => [key, stateMigration(plot)]),
    ),
  }),
});
