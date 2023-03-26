// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Store } from "@reduxjs/toolkit";
import { combineReducers } from "@reduxjs/toolkit";
import {
  reducer as driftReducer,
  TauriRuntime,
  configureStore,
  DRIFT_SLICE_NAME,
  DriftState,
} from "@synnaxlabs/drift";
import { DeepKey } from "@synnaxlabs/x";
import { appWindow } from "@tauri-apps/api/window";

import { dispatchEffect, effectMiddleware } from "./middleware";

import {
  ClusterAction,
  clusterReducer,
  ClusterState,
  CLUSTER_SLICE_NAME,
} from "@/cluster";
import { DocsAction, docsReducer, DocsState, DOCS_SLICE_NAME } from "@/docs";
import {
  LayoutAction,
  layoutReducer,
  LayoutState,
  LAYOUT_PERSIST_EXCLUDE,
  LAYOUT_SLICE_NAME,
  removeLayout,
} from "@/layout";
import { openPersist } from "@/persist";
import { versionReducer, VersionState, VERSION_SLICE_NAME } from "@/version";
import {
  VIS_SLICE_NAME,
  visReducer,
  removeVis,
  purgeRanges,
  VisAction,
  VisState,
} from "@/vis";
import {
  removeRange,
  WorkspaceAction,
  workspaceReducer,
  WorkspaceState,
  WORKSPACE_SLICE_NAME,
} from "@/workspace";

const PERSIST_EXCLUDE: Array<DeepKey<RootState>> = [
  DRIFT_SLICE_NAME,
  ...LAYOUT_PERSIST_EXCLUDE,
];

const reducer = combineReducers({
  [DRIFT_SLICE_NAME]: driftReducer,
  [CLUSTER_SLICE_NAME]: clusterReducer,
  [LAYOUT_SLICE_NAME]: layoutReducer,
  [VIS_SLICE_NAME]: visReducer,
  [WORKSPACE_SLICE_NAME]: workspaceReducer,
  [VERSION_SLICE_NAME]: versionReducer,
  [DOCS_SLICE_NAME]: docsReducer,
});

export interface RootState {
  [DRIFT_SLICE_NAME]: DriftState;
  [CLUSTER_SLICE_NAME]: ClusterState;
  [LAYOUT_SLICE_NAME]: LayoutState;
  [VIS_SLICE_NAME]: VisState;
  [WORKSPACE_SLICE_NAME]: WorkspaceState;
  [VERSION_SLICE_NAME]: VersionState;
  [DOCS_SLICE_NAME]: DocsState;
}

export type Action =
  | VisAction
  | LayoutAction
  | WorkspaceAction
  | DocsAction
  | ClusterAction;

export type Payload = Action["payload"];

export type RootStore = Store<RootState, Action>;

const newStore = async (): Promise<RootStore> => {
  const [preloadedState, persistMiddleware] = await openPersist<RootState>({
    exclude: PERSIST_EXCLUDE,
  });
  return (await configureStore<RootState, Action>({
    runtime: new TauriRuntime(appWindow),
    preloadedState,
    middleware: (def) => [
      ...def(),
      effectMiddleware([removeLayout.type], [dispatchEffect(removeVis)]),
      effectMiddleware([removeRange.type], [dispatchEffect(purgeRanges)]),
      persistMiddleware,
    ],
    reducer,
    enablePrerender: true,
  })) as RootStore;
};

export const store = newStore();
