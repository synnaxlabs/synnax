// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dispatch } from "react";

import type {
  Action,
  AnyAction,
  PreloadedState as BasePreloadedState,
  CombinedState,
  PayloadAction,
} from "@reduxjs/toolkit";
import { createSlice } from "@reduxjs/toolkit";
import type { NoInfer } from "@reduxjs/toolkit/dist/tsHelpers";
import { XY, Dimensions } from "@synnaxlabs/x";

import { log } from "./debug";

import { Runtime } from "@/runtime";
import { KeyedWindowProps, WindowState, WindowProps, WindowStage } from "@/window";

/** The Slice State */
export interface DriftState {
  key: string;
  windows: Record<string, WindowState>;
}

/** State of a store with a drift slice */
export interface StoreState {
  drift: DriftState;
}

export type PreloadedState<S extends StoreState> = BasePreloadedState<
  CombinedState<NoInfer<S>>
>;

// Disabling consistent type definitions here because 'empty' interfaces can't be named,
// which raises an error on build.
// eslint-disable-next-line @typescript-eslint/consistent-type-definitions
type MaybeKeyPayload = { key?: string };
export interface KeyPayload {
  key: string;
}
export interface MaybeBooleanPayload {
  value?: boolean;
}
type CreateWindowPayload = WindowProps;
type CloseWindowPayload = MaybeKeyPayload;
type SetWindowClosedPayload = MaybeKeyPayload;

type FocusWindowPayload = MaybeKeyPayload;

type SetWindowMinimizedPayload = MaybeKeyPayload & MaybeBooleanPayload;
type SetWindowMaximizedPayload = MaybeKeyPayload & MaybeBooleanPayload;
type SetWindowVisiblePayload = MaybeKeyPayload & MaybeBooleanPayload;
type SetWindowFullScreenPayload = MaybeKeyPayload & MaybeBooleanPayload;
type CenterWindowPayload = MaybeKeyPayload;
type SetWindowPositionPayload = MaybeKeyPayload & XY;
type SetWindowSizePayload = MaybeKeyPayload & Dimensions;
type SetWindowMinSizePayload = MaybeKeyPayload & Dimensions;
type SetWindowMaxSizePayload = MaybeKeyPayload & Dimensions;
type SetWindowResizablePayload = MaybeKeyPayload & { resizable: boolean };
type SetWindowSkipTaskbarPayload = MaybeKeyPayload & { skipTaskbar: boolean };
type SetWindowAlwaysOnTopPayload = MaybeKeyPayload & { alwaysOnTop: boolean };
type SetWindowTitlePayload = MaybeKeyPayload & { title: string };

type SetWindowKeyPayload = KeyPayload;
type SetWindowStatePayload = MaybeKeyPayload & { stage: WindowStage };
type SetWindowPropsPayload = MaybeKeyPayload & Partial<WindowProps>;
type SetWindowErrorPaylod = MaybeKeyPayload & { message: string };

/** Type representing all possible actions that are drift related. */
export type DriftAction = PayloadAction<
  | CreateWindowPayload
  | CloseWindowPayload
  | SetWindowStatePayload
  | MaybeKeyPayload
  | SetWindowPropsPayload
  | SetWindowErrorPaylod
  | SetWindowKeyPayload
  | SetWindowClosedPayload
  | SetWindowMinimizedPayload
  | SetWindowMaximizedPayload
  | SetWindowVisiblePayload
  | SetWindowFullScreenPayload
  | CenterWindowPayload
  | SetWindowPositionPayload
  | SetWindowSizePayload
  | SetWindowMinSizePayload
  | SetWindowMaxSizePayload
  | SetWindowResizablePayload
  | SetWindowSkipTaskbarPayload
  | SetWindowAlwaysOnTopPayload
  | SetWindowTitlePayload
  | FocusWindowPayload
>;

export const initialState: DriftState = {
  key: "main",
  windows: {
    main: {
      key: "main",
      processCount: 0,
      stage: "created",
    },
  },
};

const assertKey = <T extends MaybeKeyPayload>(pld: MaybeKeyPayload): T & KeyPayload => {
  if (pld.key === undefined) {
    throw new Error("drift - bug - key is undefined");
  }
  return pld as T & KeyPayload;
};

export const DRIFT_SLICE_NAME = "drift";

const slice = createSlice({
  name: DRIFT_SLICE_NAME,
  initialState,
  reducers: {
    setWindowKey: (state, action: PayloadAction<SetWindowKeyPayload>) => {
      state.key = assertKey<SetWindowKeyPayload>(action.payload).key;
    },
    createWindow: (state, { payload }: PayloadAction<CreateWindowPayload>) => {
      const { key } = payload;
      assertKey(payload);
      if (key == null || key in state.windows) return;
      state.windows[key] = {
        stage: "creating",
        processCount: 0,
        ...(payload as KeyedWindowProps),
      };
    },
    setWindowStage: (state, { payload }: PayloadAction<SetWindowStatePayload>) => {
      const { key, stage } = assertKey<SetWindowStatePayload>(payload);
      state.windows[key].stage = stage;
    },
    setWindowProps: (state, { payload }: PayloadAction<SetWindowPropsPayload>) => {
      const { key, ...props } = assertKey<SetWindowPropsPayload>(payload);
      state.windows[key] = { ...state.windows[key], ...props };
    },
    closeWindow: (state, { payload }: PayloadAction<CloseWindowPayload>) => {
      const { key } = assertKey<CloseWindowPayload>(payload);
      state.windows[key].stage = "closing";
    },
    setWindowClosed: (state, { payload }: PayloadAction<SetWindowClosedPayload>) => {
      const { key } = assertKey<SetWindowClosedPayload>(payload);
      state.windows[key].stage = "closed";
    },
    registerProcess: (state, { payload }: PayloadAction<MaybeKeyPayload>) => {
      const { key } = assertKey<MaybeKeyPayload>(payload);
      state.windows[key].processCount += 1;
    },
    completeProcess: (state, { payload }: PayloadAction<MaybeKeyPayload>) => {
      const { key } = assertKey<MaybeKeyPayload>(payload);
      state.windows[key].processCount -= 1;
    },
    setWindowError: (state, { payload }: PayloadAction<SetWindowErrorPaylod>) => {
      const { key, message } = assertKey<SetWindowErrorPaylod>(payload);
      state.windows[key].error = message;
    },
    focusWindow: (_s, _a: PayloadAction<FocusWindowPayload>) => {},
    setWindowMinimized: (_s, _a: PayloadAction<SetWindowMinimizedPayload>) => {},
    setWindowMaximized: (_s, _a: PayloadAction<SetWindowMaximizedPayload>) => {},
    setWindowVisible: (_s, _a: PayloadAction<SetWindowVisiblePayload>) => {},
    setWindowFullscreen: (_s, _a: PayloadAction<SetWindowFullScreenPayload>) => {},
    centerWindow: (_s, _a: PayloadAction<CenterWindowPayload>) => {},
    setWindowPosition: (_s, _a: PayloadAction<SetWindowPositionPayload>) => {},
    setWindowSize: (_s, _a: PayloadAction<SetWindowSizePayload>) => {},
    setWindowMinSize: (_s, _a: PayloadAction<SetWindowMinSizePayload>) => {},
    setWindowMaxSize: (_s, _a: PayloadAction<SetWindowMaxSizePayload>) => {},
    setWindowResizable: (_s, _a: PayloadAction<SetWindowResizablePayload>) => {},
    setWindowSkipTaskbar: (_s, _a: PayloadAction<SetWindowSkipTaskbarPayload>) => {},
    setWindowAlwaysOnTop: (_s, _a: PayloadAction<SetWindowAlwaysOnTopPayload>) => {},
    setWindowTitle: (_s, _a: PayloadAction<SetWindowTitlePayload>) => {},
  },
});

export const {
  reducer,
  actions: {
    setWindowKey,
    createWindow,
    setWindowStage,
    setWindowProps,
    closeWindow,
    setWindowClosed,
    registerProcess,
    completeProcess,
    setWindowError,
    focusWindow,
    setWindowMinimized,
    setWindowMaximized,
    setWindowVisible,
    setWindowFullscreen,
    centerWindow,
    setWindowPosition,
    setWindowSize,
    setWindowMinSize,
    setWindowMaxSize,
    setWindowResizable,
    setWindowSkipTaskbar,
    setWindowAlwaysOnTop,
    setWindowTitle,
  },
} = slice;

/**
 * @returns true if the given action type is a drift action.
 * @param type - The action type to check.
 */
export const isDrift = (type: string): boolean => type.startsWith(DRIFT_SLICE_NAME);

/** A list of actions that shouldn't be emitted to other windows. */
const EXCLUDED_ACTIONS: readonly string[] = [setWindowKey.type];

/**
 * @returns true if the action with the given type should be emitted to other
 * windows.
 * @param emitted - Boolean indicating if the action was emitted by another window.
 * @param type - The action type to check.
 *
 */
export const shouldEmit = (emitted: boolean, type: string): boolean =>
  !emitted && !EXCLUDED_ACTIONS.includes(type);

/**
 * Conditionally returns a default key for a given action.
 * @param runtime - The runtime of the current window.
 * @param action - The action to check.
 * @param state - The current state of the store.
 * @returns the correct key for the action.
 */
export const assignKey = <S extends StoreState, A extends Action>(
  runtime: Runtime<S, A>,
  { type, payload: { key } }: DriftAction,
  { drift: { windows } }: S
): string => {
  if (key != null) return key;
  if (type === createWindow.type) return `window-${Object.keys(windows).length + 1}`;
  return runtime.key();
};

/**
 * Executes a drift action on a window.
 *
 * @param runtime - The runtime of the current window.
 * @param action - The action to execute.
 * @param state - The current state of the store.
 */
export const processAction = <S extends StoreState, A extends Action = AnyAction>(
  runtime: Runtime<S, A>,
  { type, payload }: DriftAction,
  { drift: { windows } }: S,
  dispatch: Dispatch<A | DriftAction>,
  debug: boolean = false
): void => {
  const { key } = payload;
  if (key == null) throw new Error("[drift] - bug - action doesn't have a key");

  if (type === createWindow.type) {
    if (!runtime.isMain()) return;
    // If we've already created a window with this key, focus it.
    const existing = windows[key];
    const shouldCreate = existing == null || existing.stage === "closed";
    log(debug, "createWindow", { key, shouldCreate });
    if (shouldCreate) runtime.create(payload as KeyedWindowProps);
    else dispatch(focusWindow({ key }));
    return;
  }

  const win = windows[key];
  if (win == null)
    return console.error(`[drift] - window with key ${key} doesn't exist`);

  switch (type) {
    case closeWindow.type: {
      if (!runtime.isMain()) return;
      // If no processes are running, close the window immediately.
      // Execute a close request even if we can't find the window in state.
      // This is mainly to deal with redux state being out of sync with the
      // window state.
      log(debug, "closeWindow", { key, win });
      if (win == null || win.processCount <= 0)
        runExec(runtime.close(key), dispatch, setWindowClosed({ key }));
      return;
    }
    case completeProcess.type: {
      if (!runtime.isMain()) return;
      // If no processes are running, close the window. Threshold
      // set at 1 because we haven't yet updated the state to include the last
      // closure.
      const win = windows[key];
      if (win.processCount <= 1 && win.stage === "closing")
        runExec(runtime.close(key), dispatch, setWindowClosed({ key }));
      return;
    }
  }

  if (runtime.key() !== key) return;

  switch (type) {
    case focusWindow.type: {
      log(debug, "focusWindow", { key });
      runExec(runtime.focus(), dispatch, setWindowProps({ key, focus: true }));
      break;
    }
    case setWindowFullscreen.type: {
      const { value } = payload as SetWindowFullScreenPayload;
      const fullscreen = value ?? !(win.fullscreen ?? false);
      log(debug, "fullscreenWindow", { key, fullscreen });
      runExec(
        runtime.setFullscreen(fullscreen),
        dispatch,
        setWindowProps({ key, fullscreen })
      );
      break;
    }
    case setWindowMinimized.type: {
      const { value } = payload as SetWindowMinimizedPayload;
      const minimized = value ?? !(win.visible ?? false);
      log(debug, "minimizeWindow", { key, minimized });
      runExec(
        runtime.setMinimized(minimized),
        dispatch,
        setWindowProps({ key, visible: minimized })
      );
      break;
    }
    case setWindowMaximized.type: {
      const { value } = payload as SetWindowMaximizedPayload;
      const maximized = value ?? !(win.maximized ?? false);
      log(debug, "maximizeWindow", { key, maximized });
      runExec(
        runtime.setMaximized(maximized),
        dispatch,
        setWindowProps({ key, maximized })
      );
      break;
    }
    case setWindowVisible.type: {
      const { value } = payload as SetWindowVisiblePayload;
      const visible = value ?? !(win.visible ?? false);
      log(debug, "showWindow", { key, visible });
      runExec(runtime.setVisible(visible), dispatch, setWindowProps({ key, visible }));
      break;
    }
    case setWindowPosition.type: {
      const pos = payload as SetWindowPositionPayload;
      log(debug, "setWindowPosition", { key, ...pos });
      runExec(runtime.setPosition(pos), dispatch, setWindowProps({ key, ...pos }));
      break;
    }
    case setWindowSize.type: {
      const dims = payload as SetWindowSizePayload;
      log(debug, "setWindowSize", { key, ...dims });
      runExec(runtime.setSize(dims), dispatch, setWindowProps({ key, ...dims }));
      break;
    }
    case setWindowMinSize.type: {
      const dims = payload as SetWindowMinSizePayload;
      log(debug, "setWindowMinSize", { key, ...dims });
      runExec(
        runtime.setMinSize(dims),
        dispatch,
        setWindowProps({ key, minWidth: dims.width, minHeight: dims.height })
      );
      break;
    }
    case setWindowMaxSize.type: {
      const dims = payload as SetWindowMaxSizePayload;
      log(debug, "setWindowMaxSize", { key, ...dims });
      runExec(
        runtime.setMaxSize(dims),
        dispatch,
        setWindowProps({ key, maxWidth: dims.width, maxHeight: dims.height })
      );
      break;
    }
    case setWindowTitle.type: {
      const { title } = payload as SetWindowTitlePayload;
      log(debug, "setWindowTitle", { key, title });
      runExec(runtime.setTitle(title), dispatch, setWindowProps({ key, title }));
      break;
    }
    case centerWindow.type: {
      log(debug, "centerWindow", { key });
      runExec(runtime.center(), dispatch, setWindowProps({ key, center: true }));
      break;
    }
  }
};

export const runExec = (
  result: Promise<void>,
  dispatch: Dispatch<DriftAction>,
  success: DriftAction
): void => {
  result.then(() => dispatch(success)).catch((e) => dispatch(setWindowError(e)));
};
