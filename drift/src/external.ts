// Copyright 2024 Synnax Labs, Inc.
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
  initialState,
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
  setWindowResizable,
  setWindowSize,
  setWindowSkipTaskbar,
  setWindowStage,
  setWindowTitle,
  setWindowVisible,
  SLICE_NAME,
} from "@/state";
export type { WindowProps, WindowStage, WindowState } from "@/window";
export { MAIN_WINDOW } from "@/window";
