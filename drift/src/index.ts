// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type { Runtime } from "@/runtime";
export { configureStore } from "@/configureStore";
export {
  reducer,
  createWindow,
  closeWindow,
  setWindowMinimized,
  setWindowMaximized,
  setWindowFullscreen,
  setWindowVisible,
  setWindowPosition,
  setWindowSize,
  setWindowAlwaysOnTop,
  setWindowMinSize,
  setWindowMaxSize,
  setWindowResizable,
  setWindowTitle,
  setWindowSkipTaskbar,
  registerProcess,
  completeProcess,
  initialState,
  setWindowDecorations,
  DRIFT_SLICE_NAME,
} from "@/state";
export type { StoreState as DriftStoreState, DriftState, DriftAction } from "@/state";
export type {
  WindowProps,
  WindowStage as WindowState,
  WindowState as Window,
} from "@/window";
export { MAIN_WINDOW } from "@/window";
export {
  Provider,
  useWindowLifecycle,
  useSelectWindow,
  useSelectWindowKey,
  useSelectWindowAttribute,
} from "@/react";
export { TauriRuntime } from "@/tauri";
