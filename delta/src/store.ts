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

import { lineMiddleware } from "./line/store/middleware";
import {
  LINE_SLICE_NAME,
  LineAction,
  LineSliceState,
  lineReducer,
} from "./line/store/slice";
import {
  PIDAction,
  PIDSliceState,
  PID_SLICE_NAME,
  pidReducer,
} from "./pid/store/slice";

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
  LAYOUT_PERSIST_EXCLUDE,
  LAYOUT_SLICE_NAME,
  LayoutSliceState,
} from "@/layout";
import { openPersist } from "@/persist";
import { versionReducer, VersionState, VERSION_SLICE_NAME } from "@/version";
import {
  WorkspaceAction,
  workspaceReducer,
  WorkspaceSliceState,
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
  [PID_SLICE_NAME]: pidReducer,
  [WORKSPACE_SLICE_NAME]: workspaceReducer,
  [VERSION_SLICE_NAME]: versionReducer,
  [DOCS_SLICE_NAME]: docsReducer,
  [LINE_SLICE_NAME]: lineReducer,
});

export interface RootState {
  [DRIFT_SLICE_NAME]: DriftState;
  [CLUSTER_SLICE_NAME]: ClusterState;
  [LAYOUT_SLICE_NAME]: LayoutSliceState;
  [WORKSPACE_SLICE_NAME]: WorkspaceSliceState;
  [VERSION_SLICE_NAME]: VersionState;
  [DOCS_SLICE_NAME]: DocsState;
  [PID_SLICE_NAME]: PIDSliceState;
  [LINE_SLICE_NAME]: LineSliceState;
}

export type Action =
  | LayoutAction
  | WorkspaceAction
  | DocsAction
  | ClusterAction
  | LineAction
  | PIDAction;

export type Payload = Action["payload"];

export type RootStore = Store<RootState, Action>;

const newStore = async (): Promise<RootStore> => {
  const [preloadedState, persistMiddleware] = await openPersist<RootState>({
    exclude: PERSIST_EXCLUDE,
  });
  return (await configureStore<RootState, Action>({
    runtime: new TauriRuntime(appWindow),
    preloadedState,
    middleware: (def) => [...def(), ...lineMiddleware, persistMiddleware],
    reducer,
    enablePrerender: false,
  })) as RootStore;
};

export const store = newStore();
