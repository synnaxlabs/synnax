// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Reducer, Store } from "@reduxjs/toolkit";
import { combineReducers, Tuple } from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { TauriRuntime } from "@synnaxlabs/drift/tauri";
import { type deep } from "@synnaxlabs/x";
import { appWindow } from "@tauri-apps/api/window";

import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Persist } from "@/persist";
import { PID } from "@/pid";
import { Range } from "@/range";
import { Version } from "@/version";
import { Workspace } from "@/workspace";

const PERSIST_EXCLUDE: Array<deep.Key<RootState>> = [
  Drift.SLICE_NAME,
  ...Layout.PERSIST_EXCLUDE,
  Cluster.PERSIST_EXCLUDE,
];

const reducer = combineReducers({
  [Drift.SLICE_NAME]: Drift.reducer,
  [Cluster.SLICE_NAME]: Cluster.reducer,
  [Layout.SLICE_NAME]: Layout.reducer,
  [PID.SLICE_NAME]: PID.reducer,
  [Range.SLICE_NAME]: Range.reducer,
  [Version.SLICE_NAME]: Version.reducer,
  [Docs.SLICE_NAME]: Docs.reducer,
  [LinePlot.SLICE_NAME]: LinePlot.reducer,
  [Workspace.SLICE_NAME]: Workspace.reducer,
}) as unknown as Reducer<RootState, RootAction>;

export interface RootState {
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Cluster.SLICE_NAME]: Cluster.SliceState;
  [Layout.SLICE_NAME]: Layout.SliceState;
  [Range.SLICE_NAME]: Range.SliceState;
  [Version.SLICE_NAME]: Version.SliceState;
  [Docs.SLICE_NAME]: Docs.SliceState;
  [PID.SLICE_NAME]: PID.SliceState;
  [LinePlot.SLICE_NAME]: LinePlot.SliceState;
  [Workspace.SLICE_NAME]: Workspace.SliceState;
}

export type RootAction =
  | Drift.Action
  | Layout.Action
  | Range.Action
  | Docs.Action
  | Cluster.Action
  | LinePlot.Action
  | PID.Action
  | Range.Action
  | Workspace.Action;

export type Payload = RootAction["payload"];

export type RootStore = Store<RootState, RootAction>;

const DEFAULT_WINDOW_PROPS: Omit<Drift.WindowProps, "key"> = {
  fileDropEnabled: false,
};

export const migrateState = (prev: RootState): RootState => ({
  ...prev,
  layout: Layout.migrateSlice(prev.layout),
  pid: PID.migrateSlice(prev.pid),
  line: LinePlot.migrateSlice(prev.line),
  version: Version.migrateSlice(prev.version),
  workspace: Workspace.migrateSlice(prev.workspace),
  range: Range.migrateSlice(prev.range),
  docs: Docs.migrateSlice(prev.docs),
  cluster: Cluster.migrateSlice(prev.cluster),
});

const newStore = async (): Promise<RootStore> => {
  const [preloadedState, persistMiddleware] = await Persist.open<RootState>({
    migrator: migrateState,
    exclude: PERSIST_EXCLUDE,
  });
  return await Drift.configureStore<RootState, RootAction>({
    runtime: new TauriRuntime(appWindow),
    preloadedState,
    middleware: (def) =>
      new Tuple(
        ...def(),
        ...LinePlot.MIDDLEWARE,
        ...Layout.MIDDLEWARE,
        ...PID.MIDDLEWARE,
        persistMiddleware,
      ),
    reducer,
    enablePrerender: true,
    defaultWindowProps: DEFAULT_WINDOW_PROPS,
  });
};

export const store = newStore();
