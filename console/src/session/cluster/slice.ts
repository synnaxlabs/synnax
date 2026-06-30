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

export const clusterZ = synnaxParamsZ
  .extend({ key: z.string(), name: z.string().min(1, { message: "Name is required" }) })
  .omit({
    connectivityPollFrequency: true,
    retry: true,
    clockSkewThreshold: true,
  });
export type Cluster = z.infer<typeof clusterZ>;

const LOCAL_KEY = "LOCAL";
const LOCAL: Cluster = {
  key: LOCAL_KEY,
  name: "Local",
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
};

const DEMO_KEY = "DEMO";
const DEMO: Cluster = {
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
  activeCluster: z.string().nullable().default(null),
  clusters: z
    .record(z.string(), clusterZ)
    .default({ [LOCAL_KEY]: LOCAL, [DEMO_KEY]: DEMO }),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "cluster";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type SetPayload = Cluster;

export type SetActivePayload = string | null;

export type RemovePayload = string | string[];

export interface RenamePayload {
  key: string;
  name: string;
}

export interface ChangeKeyPayload {
  oldKey: string;
  newKey: string;
}

const checkName = (state: SliceState, name: string, key?: string) => {
  if (Object.values(state.clusters).some((c) => c.name === name && c.key !== key))
    throw new Error(`A cluster with the name ${name} already exists.`);
};

/**
 *  Purges any duplicate clusters with the exact same host, port, secure, username, and
 *  password, while keeping the cluster with the given key.
 */
const purgeDuplicateClusters = (state: SliceState, keep?: string) => {
  const clusters = Object.values(state.clusters);
  for (const cluster of clusters) {
    const duplicate = clusters.find(
      (c) =>
        (keep == null || c.key !== keep) &&
        c.key !== cluster.key &&
        c.host === cluster.host &&
        c.port === cluster.port &&
        c.secure === cluster.secure,
    );
    if (duplicate) delete state.clusters[duplicate.key];
  }
};

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    set: (state, { payload: cluster }: PayloadAction<SetPayload>) => {
      state.clusters[cluster.key] = cluster;
      purgeDuplicateClusters(state, cluster.key);
    },
    remove: ({ clusters }, { payload: keys }: PayloadAction<RemovePayload>) =>
      array.toArray(keys).forEach((key) => delete clusters[key]),
    setActive: (state, { payload: key }: PayloadAction<SetActivePayload>) => {
      state.activeCluster = key;
    },
    rename: (state, { payload: { key, name } }: PayloadAction<RenamePayload>) => {
      checkName(state, name);
      state.clusters[key].name = name;
    },
    changeKey: (
      state,
      { payload: { oldKey, newKey } }: PayloadAction<ChangeKeyPayload>,
    ) => {
      const cluster = state.clusters[oldKey];
      delete state.clusters[oldKey];
      state.clusters[newKey] = { ...cluster, key: newKey };
      if (state.activeCluster === oldKey) state.activeCluster = newKey;
    },
  },
});

export const { set, setActive, remove, rename, changeKey } = actions;

export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
