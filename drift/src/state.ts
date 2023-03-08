// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dispatch } from "react";

import {
  PreloadedState as BasePreloadedState,
  CombinedState,
  PayloadAction,
  createSlice,
  nanoid,
} from "@reduxjs/toolkit";
import type { NoInfer } from "@reduxjs/toolkit/dist/tsHelpers";
import {
  XY,
  Dimensions,
  positionInCenter,
  Box,
  ZERO_XY,
  toXYEqual,
  Deep,
} from "@synnaxlabs/x";

import { log } from "./debug";

import { Manager, Properties, MainChecker } from "@/runtime";
import {
  LabeledWindowProps,
  WindowState,
  WindowProps,
  WindowStage,
  MAIN_WINDOW,
} from "@/window";

/** The Slice State */
export interface DriftState {
  label: string;
  windows: Record<string, WindowState>;
  labelKeys: Record<string, string>;
  keyLabels: Record<string, string>;
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

export interface LabelPayload {
  label: string;
}

export interface BooleanPayload {
  value: boolean;
}
export interface MaybeBooleanPayload {
  value?: boolean;
}

export interface SizePayload {
  size: Dimensions;
}

type CreateWindowPayload = WindowProps & { prerenderLabel?: string };
type CloseWindowPayload = MaybeKeyPayload;
type SetWindowClosedPayload = MaybeKeyPayload;
type FocusWindowPayload = MaybeKeyPayload;
type SetWindowMinimizedPayload = MaybeKeyPayload & MaybeBooleanPayload;
type SetWindowMaximizedPayload = MaybeKeyPayload & MaybeBooleanPayload;
type SetWindowVisiblePayload = MaybeKeyPayload & MaybeBooleanPayload;
type SetWindowFullScreenPayload = MaybeKeyPayload & MaybeBooleanPayload;
type CenterWindowPayload = MaybeKeyPayload;
type SetWindowPositionPayload = MaybeKeyPayload & { position: XY };
type SetWindowSizePayload = MaybeKeyPayload & SizePayload;
type SetWindowMinSizePayload = MaybeKeyPayload & SizePayload;
type SetWindowMaxSizePayload = MaybeKeyPayload & SizePayload;
type SetWindowResizablePayload = MaybeKeyPayload & MaybeBooleanPayload;
type SetWindowSkipTaskbarPayload = MaybeKeyPayload & MaybeBooleanPayload;
type SetWindowAlwaysOnTopPayload = MaybeKeyPayload & MaybeBooleanPayload;
type SetWindowTitlePayload = MaybeKeyPayload & { title: string };
type SetWindowLabelPayload = LabelPayload;
type SetWindowStatePayload = MaybeKeyPayload & { stage: WindowStage };
type SetWindowPropsPayload = LabelPayload & Partial<WindowProps>;
type SetWindowErrorPaylod = KeyPayload & { message: string };

/** Type representing all possible actions that are drift related. */
export type DriftPayload =
  | CreateWindowPayload
  | CloseWindowPayload
  | SetWindowStatePayload
  | MaybeKeyPayload
  | SetWindowPropsPayload
  | SetWindowErrorPaylod
  | SetWindowLabelPayload
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
  | FocusWindowPayload;

/** Type representing all possible actions that are drift related. */
export type DriftAction = PayloadAction<DriftPayload>;

export const initialState: DriftState = {
  label: "main",
  windows: {
    main: {
      key: "main",
      label: "main",
      reserved: true,
      stage: "created",
      processCount: 0,
      focusCount: 0,
      centerCount: 0,
      visible: true,
    },
  },
  labelKeys: {
    main: "main",
  },
  keyLabels: {
    main: "main",
  },
};

export const assignLabel = <T extends MaybeKeyPayload | LabelPayload>(
  a: PayloadAction<T>,
  s: DriftState
): PayloadAction<T & LabelPayload> => {
  if (a.type === createWindow.type) {
    if (s.label !== MAIN_WINDOW) return a as PayloadAction<T & LabelPayload>;
    (a.payload as CreateWindowPayload).label = nanoid();
    (a.payload as CreateWindowPayload).prerenderLabel = nanoid();
    return a as PayloadAction<T & LabelPayload>;
  }
  if ("label" in a.payload) return a as PayloadAction<T & LabelPayload>;
  let label = s.label;
  // eslint-disable-next-line
  const pld = a.payload as MaybeKeyPayload;
  if (pld.key != null)
    if (pld.key in s.windows) label = pld.key;
    else label = s.keyLabels[pld.key];
  a.payload = { ...a.payload, label };
  return a as PayloadAction<T & LabelPayload>;
};

const assertLabel =
  <T extends DriftPayload>(
    f: (state: DriftState, action: PayloadAction<T & LabelPayload>) => void
  ): ((s: DriftState, a: PayloadAction<T>) => void) =>
  (s, a) => {
    if (!("label" in a.payload)) throw new Error("Missing label");
    f(s, a as PayloadAction<T & LabelPayload>);
  };

const assignBool = <T extends MaybeKeyPayload & MaybeBooleanPayload>(
  prop: keyof WindowProps,
  def_: boolean = false
): ((s: DriftState, a: PayloadAction<T>) => void) =>
  assertLabel<T>((s, a) => {
    let v = def_;
    const win = s.windows[a.payload.label];
    if (a.payload.value != null) v = a.payload.value;
    else {
      const existing = win[prop] as boolean | undefined;
      if (existing != null) v = !existing;
    }
    s.windows[a.payload.label] = { ...win, [prop]: v };
  });

const incrementCounter =
  (prop: keyof WindowState, decrement: boolean = false) =>
  (s: DriftState, a: PayloadAction<LabelPayload>) => {
    const win = s.windows[a.payload.label];
    s.windows[a.payload.label] = {
      ...win,
      [prop]: (win[prop] as number) + (decrement ? -1 : 1),
    };
  };

export const DRIFT_SLICE_NAME = "drift";

const slice = createSlice({
  name: DRIFT_SLICE_NAME,
  initialState,
  reducers: {
    setWindowLabel: (s: DriftState, a: PayloadAction<SetWindowLabelPayload>) => {
      s.label = a.payload.label;
    },
    createWindow: (s: DriftState, a: PayloadAction<CreateWindowPayload>) => {
      if (a.payload.label == null) throw new Error("label is required");

      // if the window already exists, just focus it
      if (a.payload.key in s.keyLabels) {
        const label = s.keyLabels[a.payload.key];
        s.windows[label].visible = true;
        s.windows[label].focusCount += 1;
        return;
      }

      const mainWin = s.windows.main;

      const prerender = Object.values(s.windows).find((w) => !w.reserved);

      if (
        mainWin.position != null &&
        mainWin.size != null &&
        a.payload.position == null
      )
        a.payload.position = positionInCenter(
          new Box(ZERO_XY, a.payload.size ?? ZERO_XY),
          new Box(mainWin.position, mainWin.size)
        ).topLeft;

      console.log(a.payload.position, mainWin.position, mainWin.size);

      const { prerenderLabel, ...payload } = a.payload;

      if (prerender != null) {
        s.windows[prerender.label] = {
          ...prerender,
          visible: true,
          reserved: true,
          ...payload,
          label: prerender.label,
        };
        s.labelKeys[prerender.label] = a.payload.key;
        s.keyLabels[a.payload.key] = prerender.label;
      } else {
        s.windows[a.payload.label] = {
          ...payload,
          label: a.payload.label,
          stage: "creating",
          reserved: true,
          processCount: 0,
          focusCount: 0,
          centerCount: 0,
        };
        s.labelKeys[a.payload.label] = a.payload.key;
        s.keyLabels[a.payload.key] = a.payload.label;
      }
      s.windows[prerenderLabel as string] = {
        key: "__prerender__",
        label: prerenderLabel as string,
        stage: "creating",
        visible: false,
        reserved: false,
        processCount: 0,
        focusCount: 0,
        centerCount: 0,
      };
    },
    setWindowStage: assertLabel<SetWindowStatePayload>((s, a) => {
      s.windows[a.payload.label].stage = a.payload.stage;
    }),
    closeWindow: assertLabel<CloseWindowPayload>((s, a) => {
      const win = s.windows[a.payload.label];
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete s.windows[a.payload.label];
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete s.labelKeys[win.label];
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete s.keyLabels[win.key];
    }),
    registerProcess: assertLabel<MaybeKeyPayload>(incrementCounter("processCount")),
    completeProcess: assertLabel<MaybeKeyPayload>(
      incrementCounter("processCount", true)
    ),
    setWindowError: (s: DriftState, a: PayloadAction<SetWindowErrorPaylod>) => {
      s.windows[a.payload.key].error = a.payload.message;
    },
    focusWindow: assertLabel<FocusWindowPayload>(incrementCounter("focusCount")),
    setWindowMinimized: assignBool("minimized"),
    setWindowMaximized: assignBool("maximized"),
    setWindowVisible: assignBool("visible", true),
    setWindowFullscreen: assignBool("fullscreen", true),
    centerWindow: assertLabel<CenterWindowPayload>(incrementCounter("centerCount")),
    setWindowPosition: assertLabel<SetWindowPositionPayload>((s, a) => {
      s.windows[a.payload.label].position = a.payload.position;
    }),
    setWindowSize: assertLabel<SetWindowSizePayload>((s, a) => {
      s.windows[a.payload.label].size = a.payload.size;
    }),
    setWindowMinSize: assertLabel<SetWindowMinSizePayload>((s, a) => {
      s.windows[a.payload.label].minSize = a.payload.size;
    }),
    setWindowMaxSize: assertLabel<SetWindowMaxSizePayload>((s, a) => {
      s.windows[a.payload.label].maxSize = a.payload.size;
    }),
    setWindowResizable: assignBool("resizable"),
    setWindowSkipTaskbar: assignBool("skipTaskbar"),
    setWindowAlwaysOnTop: assignBool("alwaysOnTop"),
    setWindowTitle: assertLabel<SetWindowTitlePayload>((s, a) => {
      s.windows[a.payload.label].title = a.payload.title;
    }),
    setWindowProps: (s: DriftState, a: PayloadAction<SetWindowPropsPayload>) => {
      const win = s.windows[a.payload.label];
      // @ts-expect-error
      const deepPartialEqual = Deep.partialEqual(win, a.payload);
      if (!deepPartialEqual) s.windows[a.payload.label] = { ...win, ...a.payload };
    },
  },
});

export const {
  reducer,
  actions: {
    setWindowProps,
    setWindowLabel,
    createWindow,
    setWindowStage,
    closeWindow,
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
export const isDriftAction = (type: string): boolean =>
  type.startsWith(DRIFT_SLICE_NAME);

/** A list of actions that shouldn't be emitted to other windows. */
const EXCLUDED_ACTIONS: readonly string[] = [setWindowLabel.type];

/**
 * @returns true if the action with the given type should be emitted to other
 * windows.
 * @param emitted - Boolean indicating if the action was emitted by another window.
 * @param type - The action type to check.
 *
 */
export const shouldEmit = (emitted: boolean, type: string): boolean =>
  !emitted && !EXCLUDED_ACTIONS.includes(type);

const purgeWinStateToProps = (
  window: WindowState & { prerenderLabel?: string }
): LabeledWindowProps => {
  const {
    centerCount,
    processCount,
    focusCount,
    stage,
    key,
    prerenderLabel,
    reserved,
    ...rest
  } = window;
  return rest;
};

export const sync = (
  prev: DriftState,
  next: DriftState,
  runtime: Manager & MainChecker & Properties,
  dispatch: Dispatch<DriftAction>,
  debug: boolean
): void => {
  log(debug, "sync", prev, next);

  const removed = Object.keys(prev.windows).filter((label) => !(label in next.windows));
  const added = Object.keys(next.windows).filter((label) => !(label in prev.windows));
  const isMain = runtime.isMain();
  if (isMain && removed.length > 0)
    removed.forEach((label) => {
      log(debug, "sync", "closing", label);
      if (label === MAIN_WINDOW)
        // close all other windows
        Object.keys(next.windows)
          .filter((l) => l !== MAIN_WINDOW)
          .forEach((l) => dispatch(closeWindow({ key: l })));
      void runtime.close(label);
    });
  if (isMain && added.length > 0)
    added.forEach((key) => {
      log(debug, "sync", "creating", key);
      runtime.create(purgeWinStateToProps(next.windows[key]));
    });

  const prevWin = prev.windows[runtime.label()];
  const nextWin = next.windows[runtime.label()];
  if (prevWin == null || nextWin == null) return;

  const changes: Array<[string, Promise<void>]> = [];

  if (nextWin.title != null && nextWin.title !== prevWin.title)
    changes.push(["title", runtime.setTitle(nextWin.title)]);

  if (nextWin.visible != null && nextWin.visible !== prevWin.visible)
    changes.push(["visible", runtime.setVisible(nextWin.visible)]);

  if (nextWin.skipTaskbar != null && nextWin.skipTaskbar !== prevWin.skipTaskbar)
    changes.push(["skipTaskbar", runtime.setSkipTaskbar(nextWin.skipTaskbar)]);

  if (nextWin.maximized != null && nextWin.maximized !== prevWin.maximized)
    changes.push(["maximized", runtime.setMaximized(nextWin.maximized)]);

  if (nextWin.fullscreen != null && nextWin.fullscreen !== prevWin.fullscreen)
    changes.push(["fullscreen", runtime.setFullscreen(nextWin.fullscreen)]);

  if (nextWin.centerCount !== prevWin.centerCount)
    changes.push(["center", runtime.center()]);

  if (nextWin.minimized != null && nextWin.minimized !== prevWin.minimized)
    changes.push(["minimized", runtime.setMinimized(nextWin.minimized)]);

  if (nextWin.minSize != null && !toXYEqual(nextWin.minSize, prevWin.minSize))
    changes.push(["minSize", runtime.setMinSize(nextWin.minSize)]);

  if (nextWin.maxSize != null && !toXYEqual(nextWin.maxSize, prevWin.maxSize))
    changes.push(["maxSize", runtime.setMinSize(nextWin.maxSize)]);

  if (nextWin.size != null && !toXYEqual(nextWin.size, prevWin.size))
    changes.push(["size", runtime.setSize(nextWin.size)]);

  if (nextWin.position != null && !toXYEqual(nextWin.position, prevWin.position))
    changes.push(["position", runtime.setPosition(nextWin.position)]);

  if (nextWin.focusCount !== prevWin.focusCount)
    changes.push(["focus", runtime.focus()]);

  if (nextWin.resizable != null && nextWin.resizable !== prevWin.resizable)
    changes.push(["resizable", runtime.setResizable(nextWin.resizable)]);

  if (nextWin.alwaysOnTop != null && nextWin.alwaysOnTop !== prevWin.alwaysOnTop)
    changes.push(["alwaysOnTop", runtime.setAlwaysOnTop(nextWin.alwaysOnTop)]);

  changes.forEach(([name, change]) => {
    log(debug, "sync", "changing", name);
    void change.catch((e) => dispatch(setWindowError(e)));
  });
};
