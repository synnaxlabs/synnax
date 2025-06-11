// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";
import { z } from "zod/v4";

import * as v3 from "@/schematic/types/v3";

// This file is mostly pointless, as the state is exactly the same as the previous
// version. But, customers have existing schematics and slices with the 'version' key
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
  .omit({ version: true, schematics: true })
  .extend({ version: z.literal(VERSION), schematics: z.record(z.string(), stateZ) });

export interface SliceState extends Omit<v3.SliceState, "version" | "schematics"> {
  schematics: Record<string, State>;
  version: Version;
}

export const stateMigration = migrate.createMigration<v3.State, State>({
  name: "schematic.state",
  migrate: (state) => ({
    ...state,
    version: VERSION,
    authority: 1,
  }),
});

export const sliceMigration = migrate.createMigration<v3.SliceState, SliceState>({
  name: "schematic.slice",
  migrate: (sliceState) => ({
    ...sliceState,
    schematics: Object.fromEntries(
      Object.entries(sliceState.schematics).map(([key, state]) => [
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
  schematics: {},
};
