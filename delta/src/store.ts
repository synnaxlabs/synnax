import { combineReducers } from "@reduxjs/toolkit";
import {
  reducer as driftReducer,
  TauriRuntime,
  configureStore,
} from "@synnaxlabs/drift";
import { appWindow } from "@tauri-apps/api/window";

import { clusterReducer } from "@/features/cluster";
import { layoutReducer } from "@/features/layout";
import { visualizationReducer } from "@/features/visualization";
import { workspaceReducer } from "@/features/workspace";

const reducer = combineReducers({
  drift: driftReducer,
  cluster: clusterReducer,
  layout: layoutReducer,
  visualization: visualizationReducer,
  workspace: workspaceReducer,
});

export const store = configureStore<ReturnType<typeof reducer>>({
  runtime: new TauriRuntime(appWindow),
  reducer,
});
