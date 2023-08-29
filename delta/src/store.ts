// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Store } from "@reduxjs/toolkit";
import { combineReducers } from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { TauriRuntime } from "@synnaxlabs/drift/tauri";
import { DeepKey } from "@synnaxlabs/x";
import { appWindow } from "@tauri-apps/api/window";

import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { Layout } from "@/layout";
import { Line } from "@/line";
import { Persist } from "@/persist";
import { PID } from "@/pid";
import { Version } from "@/version";
import { Workspace } from "@/workspace";

const PERSIST_EXCLUDE: Array<DeepKey<RootState>> = [
  Drift.SLICE_NAME,
  ...Layout.PERSIST_EXCLUDE,
];

const reducer = combineReducers({
  [Drift.SLICE_NAME]: Drift.reducer,
  [Cluster.SLICE_NAME]: Cluster.reducer,
  [Layout.SLICE_NAME]: Layout.reducer,
  [PID.SLICE_NAME]: PID.reducer,
  [Workspace.SLICE_NAME]: Workspace.reducer,
  [Version.SLICE_NAME]: Version.reducer,
  [Docs.LSICE_NAME]: Docs.reducer,
  [Line.SLICE_NAME]: Line.reducer,
});

export interface RootState {
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Cluster.SLICE_NAME]: Cluster.SliceState;
  [Layout.SLICE_NAME]: Layout.SliceState;
  [Workspace.SLICE_NAME]: Workspace.SliceState;
  [Version.SLICE_NAME]: Version.SliceState;
  [Docs.LSICE_NAME]: Docs.SliceState;
  [PID.SLICE_NAME]: PID.SliceState;
  [Line.SLICE_NAME]: Line.SliceState;
}

export type Action =
  | Layout.Action
  | Workspace.Action
  | Docs.Action
  | Cluster.Action
  | Line.Action
  | PID.Action;

export type Payload = Action["payload"];

export type RootStore = Store<RootState, Action>;

const DEFAULT_WINDOW_PROPS: Omit<Drift.WindowProps, "key"> = {
  transparent: true,
  fileDropEnabled: false,
};

const newStore = async (): Promise<RootStore> => {
  const [preloadedState, persistMiddleware] = await Persist.open<RootState>({
    exclude: PERSIST_EXCLUDE,
  });
  return (await Drift.configureStore<RootState, Action>({
    runtime: new TauriRuntime(appWindow),
    preloadedState,
    middleware: (def) => [
      ...def(),
      ...Line.middleware,
      ...Layout.middleware,
      persistMiddleware,
    ],
    reducer,
    enablePrerender: true,
    defaultWindowProps: DEFAULT_WINDOW_PROPS,
  })) as RootStore;
};

export const store = newStore();
