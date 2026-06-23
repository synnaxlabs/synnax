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
import { Session } from "@/layered/session";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Persist } from "@/persist";
import { Project } from "@/project";
import { Range } from "@/range";
import { Runtime } from "@/runtime";
import { Status } from "@/status";
import { Table } from "@/table";

const PERSIST_EXCLUDE: Array<deep.Key<RootState> | ((func: RootState) => RootState)> = [
  ...Layout.PERSIST_EXCLUDE,
  ...Session.Schematic.PERSIST_EXCLUDE,
  ...LinePlot.PERSIST_EXCLUDE,
];

const ZERO_STATE: RootState = {
  [Cluster.SLICE_NAME]: Cluster.ZERO_SLICE_STATE,
  [Docs.SLICE_NAME]: Docs.ZERO_SLICE_STATE,
  [Drift.SLICE_NAME]: Drift.ZERO_SLICE_STATE,
  [Layout.SLICE_NAME]: Layout.ZERO_SLICE_STATE,
  [LinePlot.SLICE_NAME]: LinePlot.ZERO_SLICE_STATE,
  [Log.SLICE_NAME]: Log.ZERO_SLICE_STATE,
  [Range.SLICE_NAME]: Range.ZERO_SLICE_STATE,
  [Session.Schematic.SLICE_NAME]: Session.Schematic.ZERO_SLICE_STATE,
  [Status.SLICE_NAME]: Status.ZERO_SLICE_STATE,
  [Table.SLICE_NAME]: Table.ZERO_SLICE_STATE,
  [Project.SLICE_NAME]: Project.ZERO_SLICE_STATE,
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
  [Session.Schematic.SLICE_NAME]: Session.Schematic.reducer,
  [Status.SLICE_NAME]: Status.reducer,
  [Table.SLICE_NAME]: Table.reducer,
  [Project.SLICE_NAME]: Project.reducer,
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
  [Session.Schematic.SLICE_NAME]: Session.Schematic.SliceState;
  [Status.SLICE_NAME]: Status.SliceState;
  [Table.SLICE_NAME]: Table.SliceState;
  [Project.SLICE_NAME]: Project.SliceState;
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
  | Session.Schematic.Action
  | Status.Action
  | Table.Action
  | Project.Action
  | Arc.Action;

export type RootStore = Store<RootState, RootAction>;

const DEFAULT_WINDOW_PROPS: Omit<Drift.WindowProps, "key"> = {
  visible: IS_DEV,
  minSize: { width: 625, height: 375 },
};

export const migrateState = (prev: RootState): RootState => {
  console.group("Migrating State");
  const layout = Layout.migrateSlice(prev.layout);
  const line = LinePlot.migrateSlice(prev.line);
  const log = Log.migrateSlice(prev.log);
  // The project slice was persisted under "workspace" before the rename;
  // migrateLegacySlice reads the legacy key so an upgrading user keeps their saved
  // active project.
  const project = Project.migrateLegacySlice(prev);
  const range = Range.migrateSlice(prev.range);
  const docs = Docs.migrateSlice(prev.docs);
  const cluster = Cluster.migrateSlice(prev.cluster);
  const arc = Arc.migrateSlice(prev.arc);
  const status = Status.migrateSlice(prev.status);
  const table = Table.migrateSlice(prev.table);
  console.log("Migrated State");
  console.groupEnd();
  return {
    ...prev,
    layout,
    line,
    log,
    project,
    range,
    docs,
    cluster,
    arc,
    status,
    table,
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
