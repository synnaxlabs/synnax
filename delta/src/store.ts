// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { combineReducers } from "@reduxjs/toolkit";
import {
  reducer as driftReducer,
  TauriRuntime,
  configureStore,
  DRIFT_SLICE_NAME,
} from "@synnaxlabs/drift";
import { DeepKey } from "@synnaxlabs/x";
import { appWindow } from "@tauri-apps/api/window";

import { clusterReducer, CLUSTER_SLICE_NAME } from "@/cluster";
import { docsReducer, DOCS_SLICE_NAME } from "@/features/docs";
import {
  layoutReducer,
  LAYOUT_PERSIST_EXCLUDE,
  LAYOUT_SLICE_NAME,
} from "@/features/layout";
import { openPersist } from "@/features/persist";
import { versionReducer, VERSION_SLICE_NAME } from "@/features/version";
import { VISUALIZATION_SLICE_NAME, visualizationReducer } from "@/features/vis";
import { workspaceReducer, WORKSPACE_SLICE_NAME } from "@/features/workspace";

const PERSIST_EXCLUDE: Array<DeepKey<RootState>> = [
  DRIFT_SLICE_NAME,
  ...LAYOUT_PERSIST_EXCLUDE,
];

const reducer = combineReducers({
  [DRIFT_SLICE_NAME]: driftReducer,
  [CLUSTER_SLICE_NAME]: clusterReducer,
  [LAYOUT_SLICE_NAME]: layoutReducer,
  [VISUALIZATION_SLICE_NAME]: visualizationReducer,
  [WORKSPACE_SLICE_NAME]: workspaceReducer,
  [VERSION_SLICE_NAME]: versionReducer,
  [DOCS_SLICE_NAME]: docsReducer,
});

/** The root state of the application.   */
export type RootState = ReturnType<typeof reducer>;

export type Store = Awaited<ReturnType<typeof configureStore<RootState>>>;

export const newStore = async (): Promise<Store> => {
  const [preloadedState, persistMiddleware] = await openPersist<RootState>({
    exclude: PERSIST_EXCLUDE,
  });
  return await configureStore({
    runtime: new TauriRuntime(appWindow),
    preloadedState,
    middleware: (def) => [...def(), persistMiddleware],
    reducer,
  });
};
