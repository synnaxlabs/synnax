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
import { useDispatch as baseUseDispatch, useStore as baseUseStore } from "react-redux";

import { Arc } from "@/session/arc";
import { Color } from "@/session/color";
import { Core } from "@/session/core";
import { Haul } from "@/session/haul";
import { Legacy } from "@/session/legacy";
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
import { Window } from "@/session/window";

const PERSIST_EXCLUDE: Array<Persist.ExcludeFn<State>> = [
  ...Arc.PERSIST_EXCLUDE,
  ...LinePlot.PERSIST_EXCLUDE,
  ...Panel.PERSIST_EXCLUDE,
  ...Schematic.PERSIST_EXCLUDE,
  ...Table.PERSIST_EXCLUDE,
];

// Every slice lives in exactly one partition scope, under the schema its stored
// bytes are parsed through, or in transient. Persist.open throws when one is
// missing from all four.
const PERSIST_SCOPES: Persist.Scopes<State> = {
  global: {
    [Core.SLICE_NAME]: Core.sliceStateZ,
    [Color.SLICE_NAME]: Color.sliceStateZ,
    [Theme.SLICE_NAME]: Theme.sliceStateZ,
  },
  // Statuses and ranges belong to the cluster, not to a project under it.
  core: {
    [Project.SLICE_NAME]: Project.sliceStateZ,
    [Range.SLICE_NAME]: Range.sliceStateZ,
    [Status.SLICE_NAME]: Status.sliceStateZ,
  },
  project: {
    [Drift.SLICE_NAME]: Drift.sliceStateZ,
    [Panel.ORDER_SLICE_NAME]: Panel.orderSliceStateZ,
  },
  // A window is a viewport, so its view of every document is stored under the window
  // rather than mixed into the project's.
  window: {
    [Arc.SLICE_NAME]: Arc.sliceStateZ,
    [LinePlot.SLICE_NAME]: LinePlot.sliceStateZ,
    [Log.SLICE_NAME]: Log.sliceStateZ,
    [Nav.SLICE_NAME]: Nav.sliceStateZ,
    [Panel.SLICE_NAME]: Panel.sliceStateZ,
    [Schematic.SLICE_NAME]: Schematic.sliceStateZ,
    [Table.SLICE_NAME]: Table.sliceStateZ,
  },
  transient: [Haul.SLICE_NAME, Persist.SLICE_NAME],
};

// Drift keys its windows by label; the key each label maps to is what the window-keyed
// slices store their state under.
const getWindows = (state: State): string[] =>
  Object.values(state[Drift.SLICE_NAME].labelKeys);

// A Core's state is partitioned by the cluster it connects to, not by the record the
// user picked: two records aimed at one cluster share a partition, and a Core that has
// never connected has no partition to open.
const getPersistContext = (state: State): Persist.Context => ({
  core: Core.selectClusterKey(state),
  project: state[Project.SLICE_NAME].selected,
});

export const ZERO_STATE: State = {
  [Arc.SLICE_NAME]: Arc.ZERO_SLICE_STATE,
  [Core.SLICE_NAME]: Core.ZERO_SLICE_STATE,
  [Color.SLICE_NAME]: Color.ZERO_SLICE_STATE,
  [Drift.SLICE_NAME]: Drift.ZERO_SLICE_STATE,
  [Haul.SLICE_NAME]: Haul.ZERO_SLICE_STATE,
  [Nav.SLICE_NAME]: Nav.ZERO_SLICE_STATE,
  [Panel.SLICE_NAME]: Panel.ZERO_SLICE_STATE,
  [Panel.ORDER_SLICE_NAME]: Panel.ZERO_ORDER_SLICE_STATE,
  [Log.SLICE_NAME]: Log.ZERO_SLICE_STATE,
  [LinePlot.SLICE_NAME]: LinePlot.ZERO_SLICE_STATE,
  [Persist.SLICE_NAME]: Persist.ZERO_SLICE_STATE,
  [Project.SLICE_NAME]: Project.ZERO_SLICE_STATE,
  [Range.SLICE_NAME]: Range.ZERO_SLICE_STATE,
  [Schematic.SLICE_NAME]: Schematic.ZERO_SLICE_STATE,
  [Status.SLICE_NAME]: Status.ZERO_SLICE_STATE,
  [Table.SLICE_NAME]: Table.ZERO_SLICE_STATE,
  [Theme.SLICE_NAME]: Theme.ZERO_SLICE_STATE,
};

const combinedReducer = combineReducers({
  [Arc.SLICE_NAME]: Arc.reducer,
  [Core.SLICE_NAME]: Core.reducer,
  [Color.SLICE_NAME]: Color.reducer,
  [Drift.SLICE_NAME]: Drift.reducer,
  [Haul.SLICE_NAME]: Haul.reducer,
  [Nav.SLICE_NAME]: Nav.reducer,
  [Panel.SLICE_NAME]: Panel.reducer,
  [Panel.ORDER_SLICE_NAME]: Panel.orderReducer,
  [Log.SLICE_NAME]: Log.reducer,
  [LinePlot.SLICE_NAME]: LinePlot.reducer,
  [Persist.SLICE_NAME]: Persist.reducer,
  [Project.SLICE_NAME]: Project.reducer,
  [Range.SLICE_NAME]: Range.reducer,
  [Schematic.SLICE_NAME]: Schematic.reducer,
  [Status.SLICE_NAME]: Status.reducer,
  [Table.SLICE_NAME]: Table.reducer,
  [Theme.SLICE_NAME]: Theme.reducer,
}) as unknown as Reducer<State, Action>;

// hydrate replaces the swapped slices wholesale on a session context switch. Drift
// merges instead, keeping the windows this process is actually running.
export const reducer: Reducer<State, Action> = (state, action) => {
  if (state != null && Persist.matchHydrate<State>(action)) {
    const { payload } = action;
    const stored = payload[Drift.SLICE_NAME];
    const live = state[Drift.SLICE_NAME];
    state = { ...state, ...payload };
    if (stored != null) state[Drift.SLICE_NAME] = Drift.restoreWindows(live, stored);
  }
  return combinedReducer(state, action);
};

export interface State {
  [Arc.SLICE_NAME]: Arc.SliceState;
  [Core.SLICE_NAME]: Core.SliceState;
  [Color.SLICE_NAME]: Color.SliceState;
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Haul.SLICE_NAME]: Haul.SliceState;
  [Log.SLICE_NAME]: Log.SliceState;
  [LinePlot.SLICE_NAME]: LinePlot.SliceState;
  [Persist.SLICE_NAME]: Persist.SliceState;
  [Project.SLICE_NAME]: Project.SliceState;
  [Nav.SLICE_NAME]: Nav.SliceState;
  [Panel.SLICE_NAME]: Panel.SliceState;
  [Panel.ORDER_SLICE_NAME]: Panel.OrderSliceState;
  [Range.SLICE_NAME]: Range.SliceState;
  [Schematic.SLICE_NAME]: Schematic.SliceState;
  [Status.SLICE_NAME]: Status.SliceState;
  [Table.SLICE_NAME]: Table.SliceState;
  [Theme.SLICE_NAME]: Theme.SliceState;
}

export type Action =
  | Arc.Action
  | Core.Action
  | Color.Action
  | Drift.Action
  | Haul.Action
  | Log.Action
  | LinePlot.Action
  | Nav.Action
  | Panel.Action
  | Panel.OrderAction
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
  size: { width: 800, height: 600 },
  minSize: { width: 625, height: 375 },
};

export const BASE_MIDDLEWARE = [
  Window.removalMiddleware,
  ...Arc.MIDDLEWARE,
  ...Core.MIDDLEWARE,
  ...LinePlot.MIDDLEWARE,
  ...Log.MIDDLEWARE,
  ...Nav.MIDDLEWARE,
  ...Panel.MIDDLEWARE,
  ...Schematic.MIDDLEWARE,
  ...Table.MIDDLEWARE,
];

export interface CreateStoreOptions extends Partial<
  Pick<
    Drift.ConfigureStoreOptions<State, Action>,
    "enablePrerender" | "debug" | "runtime" | "preloadedState"
  >
> {
  enablePersistence?: boolean;
  /** Overrides the persistence KV store. Tests inject an in-memory KV. */
  openKV?: Persist.KVOpener;
}

export const createStore = async (opts: CreateStoreOptions = {}): Promise<Store> => {
  const {
    runtime = new Runtime.Drift<State, Action>(),
    enablePrerender = !IS_DEV,
    debug = false,
    enablePersistence = true,
    openKV,
  } = opts;
  let { preloadedState } = opts;
  const middleware: Middleware[] = [...BASE_MIDDLEWARE];
  if (enablePersistence) {
    const persist = await Persist.open<State>({
      initial: ZERO_STATE,
      scopes: PERSIST_SCOPES,
      getContext: getPersistContext,
      getWindows,
      lens: Window.LENS,
      exclude: PERSIST_EXCLUDE,
      migrate: Legacy.migrate,
      openKV,
    });
    preloadedState ??= persist.initialState;
    middleware.push(persist.middleware);
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
