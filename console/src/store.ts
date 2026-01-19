// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  combineReducers,
  type Dispatch,
  type Middleware,
  type Reducer,
  type Store,
  Tuple,
} from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { type deep, type record } from "@synnaxlabs/x";

import { Arc } from "@/arc";
import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Persist } from "@/persist";
import { Range } from "@/range";
import { Runtime } from "@/runtime";
import { Schematic } from "@/schematic";
import { Status } from "@/status";
import { Table } from "@/table";
import { Version } from "@/version";
import { Workspace } from "@/workspace";

const PERSIST_EXCLUDE: Array<deep.Key<RootState> | ((func: RootState) => RootState)> = [
  ...Layout.PERSIST_EXCLUDE,
  ...Schematic.PERSIST_EXCLUDE,
];

const ZERO_STATE: RootState = {
  [Cluster.SLICE_NAME]: Cluster.ZERO_SLICE_STATE,
  [Docs.SLICE_NAME]: Docs.ZERO_SLICE_STATE,
  [Drift.SLICE_NAME]: Drift.ZERO_SLICE_STATE,
  [Layout.SLICE_NAME]: Layout.ZERO_SLICE_STATE,
  [LinePlot.SLICE_NAME]: LinePlot.ZERO_SLICE_STATE,
  [Log.SLICE_NAME]: Log.ZERO_SLICE_STATE,
  [Range.SLICE_NAME]: Range.ZERO_SLICE_STATE,
  [Schematic.SLICE_NAME]: Schematic.ZERO_SLICE_STATE,
  [Status.SLICE_NAME]: Status.ZERO_SLICE_STATE,
  [Table.SLICE_NAME]: Table.ZERO_SLICE_STATE,
  [Workspace.SLICE_NAME]: Workspace.ZERO_SLICE_STATE,
  [Version.SLICE_NAME]: Version.ZERO_SLICE_STATE,
  [Arc.SLICE_NAME]: Arc.ZERO_SLICE_STATE,
};

const reducer = combineReducers({
  [Cluster.SLICE_NAME]: Cluster.reducer,
  [Docs.SLICE_NAME]: Docs.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
  [Layout.SLICE_NAME]: Layout.reducer,
  [LinePlot.SLICE_NAME]: LinePlot.reducer,
  [Log.SLICE_NAME]: Log.reducer,
  [Range.SLICE_NAME]: Range.reducer,
  [Schematic.SLICE_NAME]: Schematic.reducer,
  [Status.SLICE_NAME]: Status.reducer,
  [Table.SLICE_NAME]: Table.reducer,
  [Version.SLICE_NAME]: Version.reducer,
  [Workspace.SLICE_NAME]: Workspace.reducer,
  [Arc.SLICE_NAME]: Arc.reducer,
}) as unknown as Reducer<RootState, RootAction>;

export interface RootState {
  [Cluster.SLICE_NAME]: Cluster.SliceState;
  [Docs.SLICE_NAME]: Docs.SliceState;
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Layout.SLICE_NAME]: Layout.SliceState;
  [LinePlot.SLICE_NAME]: LinePlot.SliceState;
  [Log.SLICE_NAME]: Log.SliceState;
  [Range.SLICE_NAME]: Range.SliceState;
  [Schematic.SLICE_NAME]: Schematic.SliceState;
  [Status.SLICE_NAME]: Status.SliceState;
  [Table.SLICE_NAME]: Table.SliceState;
  [Version.SLICE_NAME]: Version.SliceState;
  [Workspace.SLICE_NAME]: Workspace.SliceState;
  [Arc.SLICE_NAME]: Arc.SliceState;
}

export type RootAction =
  | Cluster.Action
  | Docs.Action
  | Drift.Action
  | Layout.Action
  | LinePlot.Action
  | Log.Action
  | Range.Action
  | Schematic.Action
  | Status.Action
  | Table.Action
  | Version.Action
  | Workspace.Action
  | Arc.Action;

export type RootStore = Store<RootState, RootAction>;

const DEFAULT_WINDOW_PROPS: Omit<Drift.WindowProps, "key"> = {
  visible: IS_DEV,
  minSize: { width: 625, height: 375 },
};

export const migrateState = (prev: RootState): RootState => {
  console.group("Migrating State");
  console.log(`Previous Console Version: ${prev[Version.SLICE_NAME].version}`);
  const layout = Layout.migrateSlice(prev.layout);
  const schematic = Schematic.migrateSlice(prev.schematic);
  const line = LinePlot.migrateSlice(prev.line);
  const version = Version.migrateSlice(prev.version);
  const workspace = Workspace.migrateSlice(prev.workspace);
  const range = Range.migrateSlice(prev.range);
  const docs = Docs.migrateSlice(prev.docs);
  const cluster = Cluster.migrateSlice(prev.cluster);
  const arc = Arc.migrateSlice(prev.arc);
  const status = Status.migrateSlice(prev.status);
  console.log("Migrated State");
  console.groupEnd();
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
    arc,
    status,
  };
};

interface OpenPersistReturn {
  initialState?: RootState;
  persistMiddleware: Middleware<record.Unknown, RootState, Dispatch<RootAction>>;
}

const openPersist = async (): Promise<OpenPersistReturn> => {
  if (!Runtime.isMainWindow())
    return {
      initialState: undefined,
      persistMiddleware: () => (next) => (action) => next(action),
    };
  const engine = await Persist.open<RootState>({
    initial: ZERO_STATE,
    migrator: migrateState,
    exclude: PERSIST_EXCLUDE,
  });
  return {
    initialState: engine.initialState,
    persistMiddleware: Persist.middleware(engine),
  };
};

const BASE_MIDDLEWARE = [
  ...Layout.MIDDLEWARE,
  ...LinePlot.MIDDLEWARE,
  ...Schematic.MIDDLEWARE,
  ...Arc.MIDDLEWARE,
];

const createStore = async (): Promise<RootStore> => {
  const { initialState, persistMiddleware } = await openPersist();
  return await Drift.configureStore<RootState, RootAction>({
    runtime: new Runtime.Drift(),
    preloadedState: initialState,
    middleware: (def) => new Tuple(...def(), ...BASE_MIDDLEWARE, persistMiddleware),
    reducer,
    enablePrerender: !IS_DEV,
    debug: false,
    defaultWindowProps: DEFAULT_WINDOW_PROPS,
  });
};
export const store = createStore();
