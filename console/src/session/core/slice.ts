// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { synnaxParamsZ } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

export const coreZ = synnaxParamsZ
  .extend({
    key: z.string(),
    name: z.string().min(1, { message: "Name is required" }),
  })
  .omit({
    cache: true,
    connectivityPollFrequency: true,
    retry: true,
    clockSkewThreshold: true,
  });
export interface Core extends z.infer<typeof coreZ> {}

/**
 * The key a Core is stored and partitioned under. A Core is its address: the same
 * host and port is the same Core no matter what the user named it, and an address
 * never changes underneath a running session the way a generated key did.
 */
export const key = ({ host, port }: Pick<Core, "host" | "port">): string =>
  `${host}:${port}`;

/** Stamps a Core with the key its address implies. */
export const keyed = (core: Omit<Core, "key">): Core => ({ ...core, key: key(core) });

const LOCAL: Core = keyed({
  name: "Local",
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
});

const DEMO: Core = keyed({
  name: "Demo",
  host: "demo.synnaxlabs.com",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: true,
});

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  selected: z.string().optional(),
  cores: z.record(z.string(), coreZ).default({ [LOCAL.key]: LOCAL, [DEMO.key]: DEMO }),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "core";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface SetPayload extends Omit<Core, "key"> {
  /** The key the Core is stored under today, when its address is being changed. */
  prevKey?: string;
}

export type SelectPayload = string;

export type RemovePayload = string | string[];

export interface RenamePayload {
  key: string;
  name: string;
}

const checkName = (state: SliceState, name: string, key?: string) => {
  if (Object.entries(state.cores).some(([k, c]) => c.name === name && k !== key))
    throw new Error(`A Core with the name ${name} already exists.`);
};

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    set: (state, { payload: { prevKey, ...core } }: PayloadAction<SetPayload>) => {
      const next = keyed(core);
      // Editing an address moves the entry rather than leaving a stale twin behind.
      if (prevKey != null && prevKey !== next.key) delete state.cores[prevKey];
      state.cores[next.key] = next;
      if (state.selected === prevKey) state.selected = next.key;
    },
    remove: (state, { payload: keys }: PayloadAction<RemovePayload>) => {
      const removed = array.toArray(keys);
      removed.forEach((key) => delete state.cores[key]);
      if (state.selected != null && removed.includes(state.selected))
        state.selected = undefined;
    },
    select: (state, { payload: key }: PayloadAction<SelectPayload>) => {
      state.selected = key;
    },
    clearSelected: (state) => {
      state.selected = undefined;
    },
    rename: (state, { payload: { key, name } }: PayloadAction<RenamePayload>) => {
      checkName(state, name, key);
      state.cores[key].name = name;
    },
  },
});

export const { set, select, clearSelected, remove, rename } = actions;

export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
