// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { measure } from "@synnaxlabs/pluto/ether";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/lineplot/types/v1";
import * as v3 from "@/lineplot/types/v3";

export const VERSION = "4.0.0";

const measureStateZ = z.object({
  mode: measure.modeZ,
});

export interface MeasureState extends z.infer<typeof measureStateZ> {}

export const ZERO_MEASURE_STATE: MeasureState = {
  mode: "one",
};

export const stateZ = v3.stateZ.omit({ version: true }).extend({
  version: z.literal(VERSION),
  measure: measureStateZ,
});
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...v3.ZERO_STATE,
  version: VERSION,
  measure: ZERO_MEASURE_STATE,
};

export const sliceStateZ = v3.sliceStateZ
  .omit({ plots: true, version: true })
  .extend({ version: z.literal(VERSION), plots: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  plots: {},
};

export const stateMigration = migrate.createMigration<v3.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: VERSION,
    measure: ZERO_MEASURE_STATE,
  }),
});

export const sliceMigration = migrate.createMigration<v3.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ plots }) => ({
    version: VERSION,
    plots: Object.fromEntries(
      Object.entries(plots).map(([key, plot]) => [key, stateMigration(plot)]),
    ),
  }),
});
