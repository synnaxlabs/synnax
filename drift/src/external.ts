// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export { configureStore } from "@/configureStore";
export type { Runtime } from "@/runtime";
export {
  selectSliceState,
  selectWindow,
  selectWindowAttribute,
  selectWindowKey,
  selectWindowLabel,
  selectWindows,
} from "@/selectors";
export type {
  Action,
  CloseWindowPayload,
  CreateWindowPayload,
  SetWindowAlwaysOnTopPayload,
  SetWindowMaximizedPayload,
  SetWindowMaxSizePayload,
  SetWindowMinimizedPayload,
  SetWindowMinSizePayload,
  SetWindowPositionPayload,
  SetWindowPropsPayload,
  SetWindowResizablePayload,
  SetWindowSizePayload,
  SetWindowSkipTaskbarPayload,
  SetWindowTitlePayload,
  SetWindowVisiblePayload,
  SliceState,
  StoreState,
} from "@/state";
export {
  closeWindow,
  completeProcess,
  createWindow,
  focusWindow,
  reducer,
  registerProcess,
  reloadWindow,
  setWindowAlwaysOnTop,
  setWindowDecorations,
  setWindowFullscreen,
  setWindowMaximized,
  setWindowMaxSize,
  setWindowMinimized,
  setWindowMinSize,
  setWindowPosition,
  setWindowProps,
  setWindowResizable,
  setWindowSize,
  setWindowSkipTaskbar,
  setWindowStage,
  setWindowTitle,
  setWindowVisible,
  SLICE_NAME,
  ZERO_SLICE_STATE,
} from "@/state";
export {
  type WindowProps,
  windowPropsZ,
  type WindowStage,
  type WindowState,
} from "@/window";
export { MAIN_WINDOW } from "@/window";
