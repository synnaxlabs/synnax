// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export { configureStore } from "@/configureStore";
export { NoopRuntime } from "@/noop";
export { type Runtime } from "@/runtime";
export {
  selectSliceState,
  selectWindow,
  selectWindowAttribute,
  selectWindowKey,
  selectWindowLabel,
  selectWindows,
} from "@/selectors";
export {
  type Action,
  closeWindow,
  type CloseWindowPayload,
  completeProcess,
  createWindow,
  type CreateWindowPayload,
  focusWindow,
  type FocusWindowPayload,
  reducer,
  registerProcess,
  reloadWindow,
  type ReloadWindowPayload,
  setWindowAlwaysOnTop,
  type SetWindowAlwaysOnTopPayload,
  setWindowDecorations,
  type SetWindowDecorationsPayload,
  setWindowFullscreen,
  type SetWindowFullScreenPayload,
  setWindowMaximized,
  type SetWindowMaximizedPayload,
  setWindowMaxSize,
  type SetWindowMaxSizePayload,
  setWindowMinimized,
  type SetWindowMinimizedPayload,
  setWindowMinSize,
  type SetWindowMinSizePayload,
  setWindowPosition,
  type SetWindowPositionPayload,
  setWindowProps,
  type SetWindowPropsPayload,
  setWindowResizable,
  type SetWindowResizablePayload,
  setWindowSize,
  type SetWindowSizePayload,
  setWindowSkipTaskbar,
  type SetWindowSkipTaskbarPayload,
  setWindowStage,
  type SetWindowStagePayload,
  setWindowTitle,
  type SetWindowTitlePayload,
  setWindowVisible,
  type SetWindowVisiblePayload,
  SLICE_NAME,
  type SliceState,
  type StoreState,
  ZERO_SLICE_STATE,
} from "@/state";
export {
  type WindowProps,
  windowPropsZ,
  type WindowStage,
  type WindowState,
} from "@/window";
export { MAIN_WINDOW } from "@/window";
