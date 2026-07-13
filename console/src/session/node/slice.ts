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

export const nodeZ = synnaxParamsZ
  .extend({ key: z.string(), name: z.string().min(1, { message: "Name is required" }) })
  .omit({
    connectivityPollFrequency: true,
    retry: true,
    clockSkewThreshold: true,
  });
export type Node = z.infer<typeof nodeZ>;

const LOCAL_KEY = "LOCAL";
const LOCAL: Node = {
  key: LOCAL_KEY,
  name: "Local",
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
};

const DEMO_KEY = "DEMO";
const DEMO: Node = {
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
  nodes: z.record(z.string(), nodeZ).default({ [LOCAL_KEY]: LOCAL, [DEMO_KEY]: DEMO }),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "node";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type SetPayload = Node;

export type SelectPayload = string;

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
  if (Object.values(state.nodes).some((c) => c.name === name && c.key !== key))
    throw new Error(`A cluster with the name ${name} already exists.`);
};

/**
 *  Purges any duplicate nodes with the exact same host, port, secure, username, and
 *  password, while keeping the node with the given key.
 */
const purgeDuplicateNodes = (state: SliceState, keep?: string) => {
  const nodes = Object.values(state.nodes);
  for (const node of nodes) {
    const duplicate = nodes.find(
      (c) =>
        (keep == null || c.key !== keep) &&
        c.key !== node.key &&
        c.host === node.host &&
        c.port === node.port &&
        c.secure === node.secure,
    );
    if (duplicate) delete state.nodes[duplicate.key];
  }
};

const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    set: (state, { payload: node }: PayloadAction<SetPayload>) => {
      state.nodes[node.key] = node;
      purgeDuplicateNodes(state, node.key);
    },
    remove: ({ nodes }, { payload: keys }: PayloadAction<RemovePayload>) =>
      array.toArray(keys).forEach((key) => delete nodes[key]),
    select: (state, { payload: key }: PayloadAction<SelectPayload>) => {
      state.selected = key;
    },
    clearSelected: (state) => {
      state.selected = undefined;
    },
    rename: (state, { payload: { key, name } }: PayloadAction<RenamePayload>) => {
      checkName(state, name);
      state.nodes[key].name = name;
    },
    changeKey: (
      state,
      { payload: { oldKey, newKey } }: PayloadAction<ChangeKeyPayload>,
    ) => {
      const node = state.nodes[oldKey];
      delete state.nodes[oldKey];
      state.nodes[newKey] = { ...node, key: newKey };
      if (state.selected === oldKey) state.selected = newKey;
    },
  },
});

export const { set, select, clearSelected, remove, rename, changeKey } = actions;

export { reducer };

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
