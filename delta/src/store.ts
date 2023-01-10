// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
import { versionReducer, VERSION_SLICE_NAME } from "@/features/version";
import { VISUALIZATION_SLICE_NAME, visualizationReducer } from "@/features/vis";
import { workspaceReducer, WORKSPACE_SLICE_NAME } from "@/features/workspace";

const kv = new TauriKV();

const reducer = combineReducers({
  [DRIFT_SLICE_NAME]: driftReducer,
  [CLUSTER_SLICE_NAME]: clusterReducer,
  [LAYOUT_SLICE_NAME]: layoutReducer,
  [VISUALIZATION_SLICE_NAME]: visualizationReducer,
  [WORKSPACE_SLICE_NAME]: workspaceReducer,
  [VERSION_SLICE_NAME]: versionReducer,
});

/**
 * The root state of the application.
 */
export type RootState = ReturnType<typeof reducer>;

export const store = configureStore<ReturnType<typeof reducer>>({
  runtime: new TauriRuntime(appWindow),
  preloadedState: newPreloadState(kv),
  middleware: (def: CurriedGetDefaultMiddleware<ReturnType<typeof reducer>>) => [
    ...def(),
    newPersistStateMiddleware(kv),
  ],
  reducer,
});
