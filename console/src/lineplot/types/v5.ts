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
import * as v4 from "@/lineplot/types/v4";

export const VERSION = "5.0.0";

export const annotationsStateZ = z.object({
  visible: z.boolean(),
});
export interface AnnotationsState extends z.infer<typeof annotationsStateZ> {}
export const ZERO_ANNOTATIONS_STATE: AnnotationsState = { visible: true };

export const stateZ = v4.stateZ.omit({ version: true }).extend({
  version: z.literal(VERSION),
  annotations: annotationsStateZ,
});
export interface State extends z.infer<typeof stateZ> {}
export const ZERO_STATE: State = {
  ...v4.ZERO_STATE,
  version: VERSION,
  annotations: ZERO_ANNOTATIONS_STATE,
};

export const sliceStateZ = v4.sliceStateZ
  .omit({ plots: true, version: true })
  .extend({ version: z.literal(VERSION), plots: z.record(z.string(), stateZ) });
export interface SliceState extends z.infer<typeof sliceStateZ> {}
export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  plots: {},
};

export const stateMigration = migrate.createMigration<v4.State, State>({
  name: v1.STATE_MIGRATION_NAME,
  migrate: (state) => ({
    ...state,
    version: VERSION,
    annotations: ZERO_ANNOTATIONS_STATE,
  }),
});

export const sliceMigration = migrate.createMigration<v4.SliceState, SliceState>({
  name: v1.SLICE_MIGRATION_NAME,
  migrate: ({ plots }) => ({
    version: VERSION,
    plots: Object.fromEntries(
      Object.entries(plots).map(([key, plot]) => [key, stateMigration(plot)]),
    ),
  }),
});
