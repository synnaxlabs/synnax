// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";
import { v4 as uuid } from "uuid";
import { z } from "zod";

import * as v1 from "@/schematic/types/v1";

export const VERSION = "2.0.0";
export type Version = typeof VERSION;

const TYPE = "schematic";
type Type = typeof TYPE;

export const stateZ = v1.stateZ.omit({ version: true }).extend({
  version: z.literal(VERSION),
  key: z.string(),
  type: z.literal(TYPE),
});

export interface State extends Omit<v1.State, "version"> {
  version: Version;
  key: string;
  type: Type;
}

export const ZERO_STATE: State = {
  ...v1.ZERO_STATE,
  version: VERSION,
  key: "",
  type: TYPE,
};

export const sliceStateZ = v1.sliceStateZ
  .omit({ version: true, schematics: true })
  .extend({ version: z.literal(VERSION), schematics: z.record(z.string(), stateZ) });

export interface SliceState extends Omit<v1.SliceState, "version" | "schematics"> {
  schematics: Record<string, State>;
  version: Version;
}

export const stateMigration = migrate.createMigration<v1.State, State>({
  name: "schematic.state",
  migrate: (state) => ({ ...state, version: VERSION, key: uuid(), type: TYPE }),
});

export const sliceMigration = migrate.createMigration<v1.SliceState, SliceState>({
  name: "schematic.slice",
  migrate: (sliceState) => ({
    ...sliceState,
    schematics: Object.fromEntries(
      Object.entries(sliceState.schematics).map(([key, state]) => [
        key,
        { ...stateMigration(state), key },
      ]),
    ),
    version: VERSION,
  }),
});

export const ZERO_SLICE_STATE: SliceState = {
  ...v1.ZERO_SLICE_STATE,
  version: VERSION,
  schematics: {},
};
