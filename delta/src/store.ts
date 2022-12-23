import { combineReducers } from "@reduxjs/toolkit";
import {
  reducer as driftReducer,
  TauriRuntime,
  configureStore,
  DRIFT_SLICE_NAME,
} from "@synnaxlabs/drift";
import { JSONEncoderDecoder } from "@synnaxlabs/freighter";
import { appWindow } from "@tauri-apps/api/window";

import { clusterReducer, CLUSTER_SLICE_NAME } from "@/features/cluster";
import { layoutReducer, LAYOUT_SLICE_NAME } from "@/features/layout";
import {
  TauriKV,
  createPreloadedState,
  createPersistStateMiddleware,
} from "@/features/persist";
import { visualizationReducer } from "@/features/visualization";
import { workspaceReducer } from "@/features/workspace";

const kv = new TauriKV(new JSONEncoderDecoder());

const reducer = combineReducers({
  [DRIFT_SLICE_NAME]: driftReducer,
  [CLUSTER_SLICE_NAME]: clusterReducer,
  [LAYOUT_SLICE_NAME]: layoutReducer,
  visualization: visualizationReducer,
  workspace: workspaceReducer,
});

export const store = configureStore<ReturnType<typeof reducer>>({
  runtime: new TauriRuntime(appWindow),
  preloadedState: createPreloadedState(kv),
  middleware: (getDefaultMiddleware) => [
    ...getDefaultMiddleware(),
    createPersistStateMiddleware(kv),
  ],
  reducer,
});
