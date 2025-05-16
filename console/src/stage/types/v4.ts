// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v3 from "@/stage/types/v3";

// This file is mostly pointless, as the state is exactly the same as the previous
// version. But, customers have existing stages and slices with the 'version' key
// being 3.0.0, so we need to keep this file around for compatibility.

export const VERSION = "4.0.0";
export type Version = typeof VERSION;

export const stateZ = v3.stateZ
  .omit({ version: true })
  .extend({ version: z.literal(VERSION), authority: z.number() });

export interface State extends Omit<v3.State, "version"> {
  version: Version;
  authority: number;
}

export const ZERO_STATE: State = { ...v3.ZERO_STATE, version: VERSION, authority: 1 };

export const sliceStateZ = v3.sliceStateZ
  .omit({ version: true, stages: true })
  .extend({ version: z.literal(VERSION), stages: z.record(z.string(), stateZ) });

export interface SliceState extends Omit<v3.SliceState, "version" | "stages"> {
  stages: Record<string, State>;
  version: Version;
}

export const stateMigration = migrate.createMigration<v3.State, State>({
  name: "stage.state",
  migrate: (state) => ({
    ...state,
    version: VERSION,
    authority: 1,
  }),
});

export const sliceMigration = migrate.createMigration<v3.SliceState, SliceState>({
  name: "stage.slice",
  migrate: (sliceState) => ({
    ...sliceState,
    stages: Object.fromEntries(
      Object.entries(sliceState.stages).map(([key, state]) => [
        key,
        { ...stateMigration(state) },
      ]),
    ),
    version: VERSION,
  }),
});

export const ZERO_SLICE_STATE: SliceState = {
  ...v3.ZERO_SLICE_STATE,
  version: VERSION,
  stages: {},
};
