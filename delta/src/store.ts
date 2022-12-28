import { combineReducers } from "@reduxjs/toolkit";
import type { CurriedGetDefaultMiddleware } from "@reduxjs/toolkit/dist/getDefaultMiddleware";
import {
  reducer as driftReducer,
  TauriRuntime,
  configureStore,
  DRIFT_SLICE_NAME,
} from "@synnaxlabs/drift";
import { appWindow } from "@tauri-apps/api/window";

import { clusterReducer, CLUSTER_SLICE_NAME } from "@/features/cluster";
import { layoutReducer, LAYOUT_SLICE_NAME } from "@/features/layout";
import {
  TauriKV,
  newPreloadState,
  newPersistStateMiddleware,
} from "@/features/persist";
import {
  VISUALIZATION_SLICE_NAME,
  visualizationReducer,
} from "@/features/visualization";
import { workspaceReducer, WORKSPACE_SLICE_NAME } from "@/features/workspace";

const kv = new TauriKV();

const reducer = combineReducers({
  [DRIFT_SLICE_NAME]: driftReducer,
  [CLUSTER_SLICE_NAME]: clusterReducer,
  [LAYOUT_SLICE_NAME]: layoutReducer,
  [VISUALIZATION_SLICE_NAME]: visualizationReducer,
  [WORKSPACE_SLICE_NAME]: workspaceReducer,
});

/**
 * The root state of the application.
 */
export type RootState = ReturnType<typeof reducer>;

export const store = configureStore<ReturnType<typeof reducer>>({
  runtime: new TauriRuntime(appWindow),
  preloadedState: newPreloadState(kv),
  middleware: (
    getDefaultMiddleware: CurriedGetDefaultMiddleware<ReturnType<typeof reducer>>
  ) => [...getDefaultMiddleware(), newPersistStateMiddleware(kv)],
  reducer,
});
