import { combineReducers } from "@reduxjs/toolkit";
import {
  reducer as driftReducer,
  TauriRuntime,
  configureStore,
} from "@synnaxlabs/drift";
import { clusterReducer } from "@/features/cluster";
import { layoutReducer } from "@/features/layout";
import { visualizationReducer } from "@/features/visualization";

const reducer = combineReducers({
  drift: driftReducer,
  cluster: clusterReducer,
  layout: layoutReducer,
  visualization: visualizationReducer,
});

export const store = configureStore<ReturnType<typeof reducer>>({
  runtime: new TauriRuntime(),
  reducer,
});
