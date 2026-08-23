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
import { array, uuid } from "@synnaxlabs/x";
import { z } from "zod";

export const coreZ = synnaxParamsZ
  .extend({
    key: z.string(),
    name: z.string().min(1, { message: "Name is required" }),
    /**
     * The cluster the Core last connected to. Cached so a session opens the cluster's
     * stored state before a connection is up; absent until the first connection.
     */
    clusterKey: z.string().optional(),
  })
  .omit({
    cache: true,
    connectivityPollFrequency: true,
    retry: true,
    clockSkewThreshold: true,
  });
export interface Core extends z.infer<typeof coreZ> {}

/** The local Core a desktop session starts with. */
export const LOCAL_KEY = "LOCAL";

/** The public demo Core. */
export const DEMO_KEY = "DEMO";

/**
 * The Core serving a browser Console. Stored state is scoped to the page's origin and
 * a served Console's address is that origin, so a session sees one served Core.
 */
export const SERVED_KEY = "SERVED";

const LOCAL: Core = {
  key: LOCAL_KEY,
  name: "Local",
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
};

const DEMO: Core = {
  key: DEMO_KEY,
  name: "Demo",
  host: "demo.synnaxlabs.com",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: true,
};

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  selected: z.string().optional(),
  cores: z.record(z.string(), coreZ).default({ [LOCAL_KEY]: LOCAL, [DEMO_KEY]: DEMO }),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "core";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface SetPayload extends Omit<Core, "key"> {
  /** The key to store under. A Core the user has just added gets a generated one. */
  key?: string;
}

export type SelectPayload = string;

export type RemovePayload = string | string[];

export interface RenamePayload {
  key: string;
  name: string;
}

export interface SetClusterKeyPayload {
  key: string;
  clusterKey: string;
}

const nameTaken = (state: SliceState, name: string, key: string): boolean =>
  Object.values(state.cores).some((c) => c.name === name && c.key !== key);

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    set: {
      prepare: ({ key, ...core }: SetPayload) => ({
        payload: { ...core, key: key ?? uuid.create() },
      }),
      reducer: (state, { payload }: PayloadAction<Core>) => {
        // A cluster key is learned from a connection, so an edit that carries none
        // keeps what the record already cached.
        const prev = state.cores[payload.key];
        state.cores[payload.key] = {
          ...payload,
          clusterKey: payload.clusterKey ?? prev?.clusterKey,
        };
      },
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
    // Duplicate names are user input, so the reducer drops them instead of throwing;
    // the list surfaces the error before dispatching.
    rename: (state, { payload: { key, name } }: PayloadAction<RenamePayload>) => {
      const core = state.cores[key];
      if (core == null || nameTaken(state, name, key)) return;
      core.name = name;
    },
    setClusterKey: (
      state,
      { payload: { key, clusterKey } }: PayloadAction<SetClusterKeyPayload>,
    ) => {
      const core = state.cores[key];
      if (core != null) core.clusterKey = clusterKey;
    },
  },
});

export const { set, select, clearSelected, remove, rename, setClusterKey } = actions;

export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
