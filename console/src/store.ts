// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers, type Reducer, type Store, Tuple } from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { TauriRuntime } from "@synnaxlabs/drift/tauri";
import { type deep } from "@synnaxlabs/x";

import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { isDev } from "@/isDev";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Permissions } from "@/permissions";
import { Persist } from "@/persist";
import { Range } from "@/range";
import { Schematic } from "@/schematic";
import { Table } from "@/table";
import { Version } from "@/version";
import { Workspace } from "@/workspace";

const PERSIST_EXCLUDE: Array<deep.Key<RootState>> = [
  ...Layout.PERSIST_EXCLUDE,
  Cluster.PERSIST_EXCLUDE,
  ...Schematic.PERSIST_EXCLUDE,
];

const ZERO_STATE: RootState = {
  [Drift.SLICE_NAME]: Drift.ZERO_SLICE_STATE,
  [Layout.SLICE_NAME]: Layout.ZERO_SLICE_STATE,
  [Cluster.SLICE_NAME]: Cluster.ZERO_SLICE_STATE,
  [Range.SLICE_NAME]: Range.ZERO_SLICE_STATE,
  [Version.SLICE_NAME]: Version.ZERO_SLICE_STATE,
  [Docs.SLICE_NAME]: Docs.ZERO_SLICE_STATE,
  [Schematic.SLICE_NAME]: Schematic.ZERO_SLICE_STATE,
  [LinePlot.SLICE_NAME]: LinePlot.ZERO_SLICE_STATE,
  [Workspace.SLICE_NAME]: Workspace.ZERO_SLICE_STATE,
  [Permissions.SLICE_NAME]: Permissions.ZERO_SLICE_STATE,
  [Log.SLICE_NAME]: Log.ZERO_SLICE_STATE,
  [Table.SLICE_NAME]: Table.ZERO_SLICE_STATE,
};

const reducer = combineReducers({
  [Drift.SLICE_NAME]: Drift.reducer,
  [Cluster.SLICE_NAME]: Cluster.reducer,
  [Layout.SLICE_NAME]: Layout.reducer,
  [Schematic.SLICE_NAME]: Schematic.reducer,
  [Range.SLICE_NAME]: Range.reducer,
  [Version.SLICE_NAME]: Version.reducer,
  [Docs.SLICE_NAME]: Docs.reducer,
  [LinePlot.SLICE_NAME]: LinePlot.reducer,
  [Workspace.SLICE_NAME]: Workspace.reducer,
  [Permissions.SLICE_NAME]: Permissions.reducer,
  [Log.SLICE_NAME]: Log.reducer,
  [Table.SLICE_NAME]: Table.reducer,
}) as unknown as Reducer<RootState, RootAction>;

export interface RootState {
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Cluster.SLICE_NAME]: Cluster.SliceState;
  [Layout.SLICE_NAME]: Layout.SliceState;
  [Range.SLICE_NAME]: Range.SliceState;
  [Version.SLICE_NAME]: Version.SliceState;
  [Docs.SLICE_NAME]: Docs.SliceState;
  [Schematic.SLICE_NAME]: Schematic.SliceState;
  [LinePlot.SLICE_NAME]: LinePlot.SliceState;
  [Workspace.SLICE_NAME]: Workspace.SliceState;
  [Permissions.SLICE_NAME]: Permissions.SliceState;
  [Log.SLICE_NAME]: Log.SliceState;
  [Table.SLICE_NAME]: Table.SliceState;
}

export type RootAction =
  | Drift.Action
  | Layout.Action
  | Range.Action
  | Docs.Action
  | Cluster.Action
  | LinePlot.Action
  | Schematic.Action
  | Permissions.Action
  | Version.Action
  | Workspace.Action
  | Log.Action;

export type RootStore = Store<RootState, RootAction>;

const DEFAULT_WINDOW_VISIBLE = isDev();
const DEFAULT_WINDOW_PROPS: Omit<Drift.WindowProps, "key"> = {
  visible: DEFAULT_WINDOW_VISIBLE,
};

export const migrateState = (prev: RootState): RootState => {
  console.log("--------------- Migrating State ---------------");
  console.log(`Previous Console Version: ${prev[Version.SLICE_NAME].version}`);
  const layout = Layout.migrateSlice(prev.layout);
  const schematic = Schematic.migrateSlice(prev.schematic);
  const line = LinePlot.migrateSlice(prev.line);
  const version = Version.migrateSlice(prev.version);
  const workspace = Workspace.migrateSlice(prev.workspace);
  const range = Range.migrateSlice(prev.range);
  const docs = Docs.migrateSlice(prev.docs);
  const cluster = Cluster.migrateSlice(prev.cluster);
  const permissions = Permissions.migrateSlice(prev.permissions);
  console.log("--------------- Migrated State ---------------");
  return {
    ...prev,
    layout,
    schematic,
    line,
    version,
    workspace,
    range,
    docs,
    cluster,
    permissions,
  };
};

const newStore = async (): Promise<RootStore> => {
  const [preloadedState, persistMiddleware] = await Persist.open<RootState>({
    initial: ZERO_STATE,
    migrator: migrateState,
    exclude: PERSIST_EXCLUDE,
  });
  if (preloadedState != null && Drift.SLICE_NAME in preloadedState) {
    const windows = preloadedState[Drift.SLICE_NAME].windows;
    Object.keys(windows).forEach((key) => {
      if (!windows[key].reserved) return;
      windows[key].visible = DEFAULT_WINDOW_VISIBLE;
      windows[key].focusCount = 0;
      windows[key].centerCount = 0;
    });
  }
  return await Drift.configureStore<RootState, RootAction>({
    runtime: new TauriRuntime(),
    preloadedState,
    middleware: (def) =>
      new Tuple(
        ...def(),
        ...LinePlot.MIDDLEWARE,
        ...Layout.MIDDLEWARE,
        ...Schematic.MIDDLEWARE,
        persistMiddleware,
      ),
    reducer,
    enablePrerender: false,
    debug: false,
    defaultWindowProps: DEFAULT_WINDOW_PROPS,
  });
};

export const store = newStore();
