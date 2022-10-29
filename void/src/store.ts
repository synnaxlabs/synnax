import { combineReducers } from "@reduxjs/toolkit";
import {
  slice as driftSlice,
  TauriWindow,
  configureStore,
} from "@synnaxlabs/drift";
import { appWindow } from "@tauri-apps/api/window";
import { clusterReducer } from "@/features/cluster";
import { layoutReducer } from "@/features/layout";
import { visualizationReducer } from "@/features/visualization";

const reducer = combineReducers({
  drift: driftSlice.reducer,
  cluster: clusterReducer,
  layout: layoutReducer,
  visualization: visualizationReducer,
});

export const store = configureStore<ReturnType<typeof reducer>>({
  window: new TauriWindow(appWindow),
  reducer,
});
