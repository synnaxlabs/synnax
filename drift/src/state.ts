// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type {
  Action,
  AnyAction,
  PreloadedState as BasePreloadedState,
  CombinedState,
  PayloadAction,
} from "@reduxjs/toolkit";
import { createSlice } from "@reduxjs/toolkit";
import type { NoInfer } from "@reduxjs/toolkit/dist/tsHelpers";

import { Runtime } from "@/runtime";
import { KeyedWindowProps, Window, WindowProps, WindowState } from "@/window";

/** The Slice State */
interface DriftState {
  key: string;
  windows: Record<string, Window>;
}

/** State of a store with a drift slice */
export interface StoreState {
  drift: DriftState;
}

export type PreloadedState<S extends StoreState> = BasePreloadedState<
  CombinedState<NoInfer<S>>
>;

interface MaybeKeyPayload {
  key?: string;
}
interface KeyPayload {
  key: string;
}
type CreateWindowPayload = WindowProps;
type CloseWindowPayload = MaybeKeyPayload;
type SetWindowKeyPayload = KeyPayload;
type SetWindowPayload = MaybeKeyPayload & { state: WindowState };

/** Type representing all possible actions that are drift related. */
export type DriftAction = PayloadAction<
  CreateWindowPayload | CloseWindowPayload | SetWindowPayload | MaybeKeyPayload
>;

export const initialState: DriftState = {
  key: "main",
  windows: {
    main: {
      processCount: 0,
      state: "created",
      props: {
        key: "main",
      },
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
      if (key == null) return;
      state.windows[key] = {
        state: "creating",
        processCount: 0,
        props: payload as KeyedWindowProps,
      };
    },
    setWindowState: ({ windows }, { payload }: PayloadAction<SetWindowPayload>) => {
      const { key, state } = assertKey<SetWindowPayload>(payload);
      windows[key].state = state;
    },
    closeWindow: ({ windows }, { payload }: PayloadAction<CloseWindowPayload>) => {
      const { key } = assertKey<CloseWindowPayload>(payload);
      windows[key].state = "closing";
      if (readyToClose(0, windows[key])) {
        windows[key].state = "closed";
      }
    },
    registerProcess: ({ windows }, { payload }: PayloadAction<MaybeKeyPayload>) => {
      const { key } = assertKey<MaybeKeyPayload>(payload);
      windows[key].processCount += 1;
    },
    completeProcess: ({ windows }, { payload }: PayloadAction<MaybeKeyPayload>) => {
      const { key } = assertKey<MaybeKeyPayload>(payload);
      const win = windows[key];
      win.processCount -= 1;
      // If the window is closing and there are no more processes, mark it
      // as closed.
      if (readyToClose(0, win)) win.state = "closed";
    },
  },
});

export const { reducer } = slice;

const { actions } = slice;

/**
 * Creates a new window with the given properties. Is a no-op if the window
 * already exists.
 *
 * @param props - The properties of the window to create.
 *
 * @returns an action that can be dispatched.
 */
export const createWindow = (props: WindowProps): DriftAction =>
  actions.createWindow(props);

/**
 * Sets the state of the window with the given key.
 *
 * @param state - The state to set.
 * @param key - The key of the window to set the status of.
 * If not provided, the status of the current window is set.
 * @returns an action that can be dispatched.
 */
export const setWindowState = (state: WindowState, key?: string): DriftAction =>
  actions.setWindowState({ state, key });

/**
 * Closes the window with the given key.
 *
 * @param key - The key of the window to close.
 * If not provided, the current window is closed.
 * @returns an action that can be dispatched.
 */
export const closeWindow = (key?: string): DriftAction => actions.closeWindow({ key });

/**
 * Sets the key of the current window.
 *
 * @param key - The key of the window to set as the current window.
 * @returns  an action that can be dispatched.
 */
export const setWindowKey = (key: string): DriftAction => actions.setWindowKey({ key });

/**
 * Registers a process that is running in the window with the given key.
 * The window will not be closed until the process is completed. This means
 * that completeProcess must be called for the window to be closed. This is
 * useful for shutting down processes gracefully.
 *
 * @param key - The key of the window to register the process in.
 * If not provided, the current window is used.
 * @returns an action that can be dispatched.
 */
export const registerProcess = (key?: string): DriftAction =>
  actions.registerProcess({ key });

/**
 * Completes a process that was registered in the window with the given key.
 * Should only be called after registerProcess.
 *
 * @param key - The key of the window to complete the process in.
 * If not provided, the current window is used.
 * @returns an action that can be dispatched.
 */
export const completeProcess = (key?: string): DriftAction =>
  actions.completeProcess({ key });

/**
 * @returns true if the given action type is a drift action.
 * @param type - The action type to check.
 */
export const isDrift = (type: string): boolean => type.startsWith(DRIFT_SLICE_NAME);

/** A list of actions that shouldn't be emitted to other windows. */
const EXCLUDED_ACTIONS = [actions.setWindowKey.type];

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
 * Evaluates whether a window is ready to be closed.
 *
 * @param threshold - The maximum number of processes that can be running
 * before the window is considered ready to close.
 * @param state - The current state of the window.
 * @returns true if the window is ready to be closed.
 */
const readyToClose = (threshold: number, state?: Window): boolean =>
  state == null || (state.processCount <= threshold && state.state === "closing");

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
  if (type === actions.createWindow.type)
    return `window-${Object.keys(windows).length + 1}`;
  return runtime.key();
};

/**
 * Executes a drift action on a window.
 *
 * @param runtime - The runtime of the current window.
 * @param action - The action to execute.
 * @param state - The current state of the store.
 */
export const executeAction = <S extends StoreState, A extends Action = AnyAction>(
  runtime: Runtime<S, A>,
  { type, payload }: DriftAction,
  { drift: { windows } }: S
): void => {
  const { key } = payload;
  if (key == null) throw new Error("[drift] - bug - action doesn't have a key");

  switch (type) {
    case actions.createWindow.type: {
      // If we've already created a window with this key, focus it.
      if (!runtime.exists(key)) runtime.create(payload as KeyedWindowProps);
      else runtime.focus(key);
      break;
    }
    case actions.closeWindow.type: {
      // If no processes are running, close the window immediately.
      // Execute a close request even if we can't find the window in state.
      // This is mainly to deal with redux state being out of sync with the
      // window state.
      const win = windows[key] as Window | undefined;
      console.log(win, key);
      if (win == null || win.processCount <= 0) runtime.close(key);
      break;
    }
    case actions.completeProcess.type: {
      // If no processes are running, close the window. Threshold
      // set at 1 because we haven't yet updated the state to include the last
      // closure.
      const win = windows[key];
      if (win.processCount <= 1 && win.state === "closing") runtime.close(key);
    }
  }
};
