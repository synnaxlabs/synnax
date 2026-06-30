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

import { Arc } from "@/session/arc";
import { Cluster } from "@/session/cluster";
import { Docs } from "@/session/docs";
import { Layout } from "@/session/layout";
import { LinePlot } from "@/session/lineplot";
import { Log } from "@/session/log";
import { Nav } from "@/session/nav";
import { Project } from "@/session/project";
import { Range } from "@/session/range";
import { Runtime } from "@/session/runtime";
import { Schematic } from "@/session/schematic";
import { Status } from "@/session/status";
import { Table } from "@/session/table";
import { Theme } from "@/session/theme";

import { Persist } from "./persist";

const PERSIST_EXCLUDE: Array<deep.Key<RootState> | ((func: RootState) => RootState)> = [
  ...Layout.PERSIST_EXCLUDE,
  ...Arc.PERSIST_EXCLUDE,
  ...LinePlot.PERSIST_EXCLUDE,
  ...Log.PERSIST_EXCLUDE,
  ...Schematic.PERSIST_EXCLUDE,
  ...Table.PERSIST_EXCLUDE,
];

const ZERO_STATE: RootState = {
  [Arc.SLICE_NAME]: Arc.ZERO_SLICE_STATE,
  [Cluster.SLICE_NAME]: Cluster.ZERO_SLICE_STATE,
  [Docs.SLICE_NAME]: Docs.ZERO_SLICE_STATE,
  [Drift.SLICE_NAME]: Drift.ZERO_SLICE_STATE,
  [Layout.SLICE_NAME]: Layout.ZERO_SLICE_STATE,
  [Nav.SLICE_NAME]: Nav.ZERO_SLICE_STATE,
  [Log.SLICE_NAME]: Log.ZERO_SLICE_STATE,
  [LinePlot.SLICE_NAME]: LinePlot.ZERO_SLICE_STATE,
  [Project.SLICE_NAME]: Project.ZERO_SLICE_STATE,
  [Range.SLICE_NAME]: Range.ZERO_SLICE_STATE,
  [Schematic.SLICE_NAME]: Schematic.ZERO_SLICE_STATE,
  [Status.SLICE_NAME]: Status.ZERO_SLICE_STATE,
  [Table.SLICE_NAME]: Table.ZERO_SLICE_STATE,
  [Theme.SLICE_NAME]: Theme.ZERO_SLICE_STATE,
};

const reducer = combineReducers({
  [Arc.SLICE_NAME]: Arc.reducer,
  [Cluster.SLICE_NAME]: Cluster.reducer,
  [Docs.SLICE_NAME]: Docs.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
  [Layout.SLICE_NAME]: Layout.reducer,
  [Nav.SLICE_NAME]: Nav.reducer,
  [Log.SLICE_NAME]: Log.reducer,
  [LinePlot.SLICE_NAME]: LinePlot.reducer,
  [Project.SLICE_NAME]: Project.reducer,
  [Range.SLICE_NAME]: Range.reducer,
  [Schematic.SLICE_NAME]: Schematic.reducer,
  [Status.SLICE_NAME]: Status.reducer,
  [Table.SLICE_NAME]: Table.reducer,
  [Theme.SLICE_NAME]: Theme.reducer,
}) as unknown as Reducer<RootState, RootAction>;

export interface RootState {
  [Arc.SLICE_NAME]: Arc.SliceState;
  [Cluster.SLICE_NAME]: Cluster.SliceState;
  [Docs.SLICE_NAME]: Docs.SliceState;
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Layout.SLICE_NAME]: Layout.SliceState;
  [Log.SLICE_NAME]: Log.SliceState;
  [LinePlot.SLICE_NAME]: LinePlot.SliceState;
  [Project.SLICE_NAME]: Project.SliceState;
  [Nav.SLICE_NAME]: Nav.SliceState;
  [Range.SLICE_NAME]: Range.SliceState;
  [Schematic.SLICE_NAME]: Schematic.SliceState;
  [Status.SLICE_NAME]: Status.SliceState;
  [Table.SLICE_NAME]: Table.SliceState;
  [Theme.SLICE_NAME]: Theme.SliceState;
}

export type RootAction =
  | Arc.Action
  | Cluster.Action
  | Docs.Action
  | Drift.Action
  | Layout.Action
  | Log.Action
  | LinePlot.Action
  | Nav.Action
  | Project.Action
  | Range.Action
  | Schematic.Action
  | Status.Action
  | Table.Action
  | Theme.Action;

export type RootStore = Store<RootState, RootAction>;

const DEFAULT_WINDOW_PROPS: Omit<Drift.WindowProps, "key"> = {
  visible: IS_DEV,
  minSize: { width: 625, height: 375 },
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
    exclude: PERSIST_EXCLUDE,
  });
  return {
    initialState: engine.initialState,
    persistMiddleware: Persist.middleware(engine),
  };
};

const BASE_MIDDLEWARE = [...Layout.MIDDLEWARE, ...Nav.MIDDLEWARE];

export const createStore = async (): Promise<RootStore> => {
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
