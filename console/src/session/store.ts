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
  type Store as BaseStore,
  Tuple,
} from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { type deep, type record } from "@synnaxlabs/x";
import { useDispatch as baseUseDispatch, useStore as baseUseStore } from "react-redux";

import { Arc } from "@/session/arc";
import { Cluster } from "@/session/cluster";
import { Color } from "@/session/color";
import { Docs } from "@/session/docs";
import { Haul } from "@/session/haul";
import { LinePlot } from "@/session/lineplot";
import { Log } from "@/session/log";
import { Nav } from "@/session/nav";
import { Panel } from "@/session/panel";
import { Persist } from "@/session/persist";
import { Project } from "@/session/project";
import { Range } from "@/session/range";
import { Runtime } from "@/session/runtime";
import { Schematic } from "@/session/schematic";
import { Status } from "@/session/status";
import { Table } from "@/session/table";
import { Theme } from "@/session/theme";

const PERSIST_EXCLUDE: Array<deep.Key<State> | ((func: State) => State)> = [
  ...Panel.PERSIST_EXCLUDE,
  Haul.PERSIST_EXCLUDE,
  ...Arc.PERSIST_EXCLUDE,
  ...LinePlot.PERSIST_EXCLUDE,
  ...Log.PERSIST_EXCLUDE,
  ...Schematic.PERSIST_EXCLUDE,
  ...Table.PERSIST_EXCLUDE,
];

export const ZERO_STATE: State = {
  [Arc.SLICE_NAME]: Arc.ZERO_SLICE_STATE,
  [Cluster.SLICE_NAME]: Cluster.ZERO_SLICE_STATE,
  [Color.SLICE_NAME]: Color.ZERO_SLICE_STATE,
  [Haul.SLICE_NAME]: Haul.ZERO_SLICE_STATE,
  [Docs.SLICE_NAME]: Docs.ZERO_SLICE_STATE,
  [Drift.SLICE_NAME]: Drift.ZERO_SLICE_STATE,
  [Nav.SLICE_NAME]: Nav.ZERO_SLICE_STATE,
  [Panel.SLICE_NAME]: Panel.ZERO_SLICE_STATE,
  [Log.SLICE_NAME]: Log.ZERO_SLICE_STATE,
  [LinePlot.SLICE_NAME]: LinePlot.ZERO_SLICE_STATE,
  [Project.SLICE_NAME]: Project.ZERO_SLICE_STATE,
  [Range.SLICE_NAME]: Range.ZERO_SLICE_STATE,
  [Schematic.SLICE_NAME]: Schematic.ZERO_SLICE_STATE,
  [Status.SLICE_NAME]: Status.ZERO_SLICE_STATE,
  [Table.SLICE_NAME]: Table.ZERO_SLICE_STATE,
  [Theme.SLICE_NAME]: Theme.ZERO_SLICE_STATE,
};

export const reducer = combineReducers({
  [Arc.SLICE_NAME]: Arc.reducer,
  [Cluster.SLICE_NAME]: Cluster.reducer,
  [Color.SLICE_NAME]: Color.reducer,
  [Haul.SLICE_NAME]: Haul.reducer,
  [Docs.SLICE_NAME]: Docs.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
  [Nav.SLICE_NAME]: Nav.reducer,
  [Panel.SLICE_NAME]: Panel.reducer,
  [Log.SLICE_NAME]: Log.reducer,
  [LinePlot.SLICE_NAME]: LinePlot.reducer,
  [Project.SLICE_NAME]: Project.reducer,
  [Range.SLICE_NAME]: Range.reducer,
  [Schematic.SLICE_NAME]: Schematic.reducer,
  [Status.SLICE_NAME]: Status.reducer,
  [Table.SLICE_NAME]: Table.reducer,
  [Theme.SLICE_NAME]: Theme.reducer,
}) as unknown as Reducer<State, Action>;

export interface State {
  [Arc.SLICE_NAME]: Arc.SliceState;
  [Cluster.SLICE_NAME]: Cluster.SliceState;
  [Color.SLICE_NAME]: Color.SliceState;
  [Haul.SLICE_NAME]: Haul.SliceState;
  [Docs.SLICE_NAME]: Docs.SliceState;
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Log.SLICE_NAME]: Log.SliceState;
  [LinePlot.SLICE_NAME]: LinePlot.SliceState;
  [Project.SLICE_NAME]: Project.SliceState;
  [Nav.SLICE_NAME]: Nav.SliceState;
  [Panel.SLICE_NAME]: Panel.SliceState;
  [Range.SLICE_NAME]: Range.SliceState;
  [Schematic.SLICE_NAME]: Schematic.SliceState;
  [Status.SLICE_NAME]: Status.SliceState;
  [Table.SLICE_NAME]: Table.SliceState;
  [Theme.SLICE_NAME]: Theme.SliceState;
}

export type Action =
  | Arc.Action
  | Cluster.Action
  | Color.Action
  | Haul.Action
  | Docs.Action
  | Drift.Action
  | Log.Action
  | LinePlot.Action
  | Nav.Action
  | Panel.Action
  | Project.Action
  | Range.Action
  | Schematic.Action
  | Status.Action
  | Table.Action
  | Theme.Action
  | Persist.Action;

export type Store = BaseStore<State, Action>;

const DEFAULT_WINDOW_PROPS: Omit<Drift.WindowProps, "key"> = {
  visible: IS_DEV,
  minSize: { width: 625, height: 375 },
};

interface OpenPersistReturn {
  initialState?: State;
  persistMiddleware: Middleware<record.Unknown, State, Dispatch<Action>>;
}

const openPersist = async (): Promise<OpenPersistReturn> => {
  if (!Runtime.isMainWindow())
    return {
      initialState: undefined,
      persistMiddleware: () => (next) => (action) => next(action),
    };
  const engine = await Persist.open<State>({
    initial: ZERO_STATE,
    exclude: PERSIST_EXCLUDE,
  });
  return {
    initialState: engine.initialState,
    persistMiddleware: Persist.middleware(engine),
  };
};

export const BASE_MIDDLEWARE = [...Nav.MIDDLEWARE, ...Panel.MIDDLEWARE];

export interface CreateStoreOptions extends Partial<
  Pick<
    Drift.ConfigureStoreOptions<State, Action>,
    "enablePrerender" | "debug" | "runtime" | "preloadedState"
  >
> {
  enablePersistence?: boolean;
}

export const configureStore = async (opts: CreateStoreOptions = {}): Promise<Store> => {
  const {
    runtime = new Runtime.Drift<State, Action>(),
    enablePrerender = !IS_DEV,
    debug = false,
    enablePersistence = true,
  } = opts;
  let { preloadedState } = opts;
  const middleware: Middleware[] = [...BASE_MIDDLEWARE];
  if (enablePersistence) {
    const { initialState, persistMiddleware } = await openPersist();
    preloadedState ??= initialState;
    middleware.push(persistMiddleware);
  }
  return await Drift.configureStore<State, Action>({
    runtime,
    preloadedState,
    middleware: (def) => new Tuple(...def(), ...middleware),
    reducer,
    enablePrerender,
    debug,
    defaultWindowProps: DEFAULT_WINDOW_PROPS,
  });
};

export const useStore = baseUseStore.withTypes<Store>();
export const useDispatch = baseUseDispatch.withTypes<Dispatch<Action>>();
