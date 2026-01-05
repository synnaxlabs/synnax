// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction, type Reducer } from "@reduxjs/toolkit";
import { box, deep, type dimensions, id, TimeSpan, xy } from "@synnaxlabs/x";

import { group, groupEnd, log } from "@/debug";
import {
  INITIAL_PRERENDER_WINDOW_STATE,
  INITIAL_WINDOW_STATE,
  MAIN_WINDOW,
  PRERENDER_WINDOW,
  type WindowProps,
  type WindowStage,
  type WindowState,
} from "@/window";

/** The Slice State */
export interface SliceState {
  label: string;
  config: Config;
  windows: Record<string, WindowState>;
  labelKeys: Record<string, string>;
  keyLabels: Record<string, string>;
}

export interface Config {
  enablePrerender: boolean;
  defaultWindowProps: Omit<WindowProps, "key">;
  debug: boolean;
}

/** State of a store with a drift slice */
export interface StoreState {
  drift: SliceState;
}

// Disabling consistent type definitions here because 'empty' interfaces can't be named,
// which raises an error on build.

export type MaybeKeyPayload = { key?: string };
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
  size: dimensions.Dimensions;
}

export type CreateWindowPayload = WindowProps & {
  label?: string;
  prerenderLabel?: string;
};
export type CloseWindowPayload = MaybeKeyPayload;
export type ReloadWindowPayload = MaybeKeyPayload;
export type SetWindowClosedPayload = MaybeKeyPayload;
export type FocusWindowPayload = MaybeKeyPayload;
export type SetWindowMinimizedPayload = MaybeKeyPayload & MaybeBooleanPayload;
export type SetWindowMaximizedPayload = MaybeKeyPayload & MaybeBooleanPayload;
export type SetWindowVisiblePayload = MaybeKeyPayload & MaybeBooleanPayload;
export type SetWindowFullScreenPayload = MaybeKeyPayload & MaybeBooleanPayload;
export type CenterWindowPayload = MaybeKeyPayload;
export type SetWindowPositionPayload = MaybeKeyPayload & { position: xy.XY };
export type SetWindowSizePayload = MaybeKeyPayload & SizePayload;
export type SetWindowMinSizePayload = MaybeKeyPayload & SizePayload;
export type SetWindowMaxSizePayload = MaybeKeyPayload & SizePayload;
export type SetWindowResizablePayload = MaybeKeyPayload & MaybeBooleanPayload;
export type SetWindowSkipTaskbarPayload = MaybeKeyPayload & MaybeBooleanPayload;
export type SetWindowAlwaysOnTopPayload = MaybeKeyPayload & MaybeBooleanPayload;
export type SetWindowTitlePayload = MaybeKeyPayload & { title: string };
export type SetWindowLabelPayload = LabelPayload;
export type SetWindowStagePayload = MaybeKeyPayload & { stage: WindowStage };
export type RuntimeSetWindowProsPayload = LabelPayload & Partial<WindowProps>;
export type SetWindowPropsPayload = MaybeKeyPayload & Partial<WindowProps>;
export type SetWindowErrorPayload = KeyPayload & { message: string };
export type SetWindowDecorationsPayload = KeyPayload & BooleanPayload;
export type SetConfigPayload = Partial<Config>;

/** Type representing all possible actions that are drift related. */
export type Payload =
  | LabelPayload
  | CreateWindowPayload
  | CloseWindowPayload
  | SetWindowStagePayload
  | MaybeKeyPayload
  | RuntimeSetWindowProsPayload
  | SetWindowErrorPayload
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
  | FocusWindowPayload
  | SetWindowDecorationsPayload
  | SetConfigPayload
  | ReloadWindowPayload;

/** Type representing all possible actions that are drift related. */
export type Action = PayloadAction<Payload>;

// For some reason, delaying the reload causes Tauri to crash less
const RELOAD_DELAY = TimeSpan.milliseconds(50);
const delayedReload = () =>
  setTimeout(() => window.location.reload(), RELOAD_DELAY.milliseconds);

export const ZERO_SLICE_STATE: SliceState = {
  label: MAIN_WINDOW,
  config: {
    enablePrerender: true,
    debug: false,
    defaultWindowProps: {},
  },
  windows: {
    main: {
      ...INITIAL_WINDOW_STATE,
      key: MAIN_WINDOW,
      reserved: true,
    },
  },
  labelKeys: {
    main: MAIN_WINDOW,
  },
  keyLabels: {
    main: MAIN_WINDOW,
  },
};

export const assignLabel = <T extends MaybeKeyPayload | LabelPayload>(
  a: PayloadAction<T>,
  s: SliceState,
): PayloadAction<T & LabelPayload> => {
  if (a.type === createWindow.type) {
    if (s.label !== MAIN_WINDOW) return a as PayloadAction<T & LabelPayload>;
    (a.payload as CreateWindowPayload).label = id.create();
    (a.payload as CreateWindowPayload).prerenderLabel = id.create();
    return a as PayloadAction<T & LabelPayload>;
  }
  if ("label" in a.payload) return a as PayloadAction<T & LabelPayload>;
  let label = s.label;

  // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
  const pld = a.payload as MaybeKeyPayload;
  if (pld.key != null)
    if (pld.key in s.windows) label = pld.key;
    else label = s.keyLabels[pld.key];
  a.payload = { ...a.payload, label };
  return a as PayloadAction<T & LabelPayload>;
};

const assertLabel =
  <T extends Payload>(
    f: (state: SliceState, action: PayloadAction<T & LabelPayload>) => void,
  ): ((s: SliceState, a: PayloadAction<T>) => void) =>
  (s, a) => {
    if (!("label" in a.payload)) throw new Error("Missing label");
    f(s, a as PayloadAction<T & LabelPayload>);
  };

const assignBool = <T extends MaybeKeyPayload & MaybeBooleanPayload>(
  prop: keyof WindowProps,
  def_: boolean = false,
): ((s: SliceState, a: PayloadAction<T>) => void) =>
  assertLabel<T>((s, a) => {
    let v = def_;
    const win = s.windows[a.payload.label];
    if (win == null) return;
    if (a.payload.value != null) v = a.payload.value;
    else {
      const existing = win[prop] as boolean | undefined;
      if (existing != null) v = !existing;
    }
    s.windows[a.payload.label] = { ...win, [prop]: v };
  });

const incrementCounter =
  (prop: keyof WindowState, decrement: boolean = false) =>
  (s: SliceState, a: PayloadAction<LabelPayload>) => {
    const win = s.windows[a.payload.label];
    if (win == null) return;
    s.windows[a.payload.label] = {
      ...win,
      [prop]: (win[prop] as number) + (decrement ? -1 : 1),
    };
  };

const alreadyHasPreRender = (s: SliceState): boolean =>
  Object.values(s.windows).some((w) => w.key === PRERENDER_WINDOW && !w.reserved);

export const SLICE_NAME = "drift";

const maybePositionInCenter = (
  mainWin: WindowState,
  position?: xy.XY,
  size?: dimensions.Dimensions,
): xy.XY | undefined => {
  if (mainWin.position != null && mainWin.size != null && position == null)
    return box.topLeft(
      box.positionInCenter(
        box.construct(xy.ZERO, size ?? xy.ZERO),
        box.construct(mainWin.position, mainWin.size),
      ),
    );
  return position;
};

const reduceCreateWindow = (
  s: SliceState,
  { payload }: PayloadAction<CreateWindowPayload>,
): void => {
  if (payload.key === PRERENDER_WINDOW) return;
  const { key, label, prerenderLabel } = payload;
  if (label == null || prerenderLabel == null)
    throw new Error("[drift] - bug - missing label and prerender label");

  group(s.config.debug, "reducer create window");

  const mainWin = s.windows.main;
  payload.position = maybePositionInCenter(mainWin, payload.position, payload.size);

  // If the window already exists, un-minimize and focus it
  if (key in s.keyLabels) {
    log(s.config.debug, "window already exists, un-minimize and focus it");
    const existingLabel = s.keyLabels[payload.key];
    s.windows[existingLabel].visible = true;
    s.windows[existingLabel].focusCount += 1;
    s.windows[existingLabel].minimized = false;
    s.windows[existingLabel].position = payload.position;
    groupEnd(s.config.debug);
    return;
  }

  const [availableLabel, available] = Object.entries(s.windows).find(
    ([, w]) => !w.reserved,
  ) ?? [null, null];

  // If we have an available pre-rendered window, use it.
  if (availableLabel != null) {
    log(s.config.debug, "using available pre-rendered window");
    s.windows[availableLabel] = {
      ...available,
      visible: true,
      reserved: true,
      focusCount: 1,
      focus: true,
      ...payload,
    };
    s.labelKeys[availableLabel] = payload.key;
    s.keyLabels[payload.key] = availableLabel;
  } else {
    // If we don't, just create the window directly.
    log(s.config.debug, "creating new window");
    s.windows[label] = {
      ...s.config.defaultWindowProps,
      ...INITIAL_WINDOW_STATE,
      ...payload,
      reserved: true,
    };
    s.labelKeys[label] = key;
    s.keyLabels[key] = label;
  }

  if (s.config.enablePrerender && !alreadyHasPreRender(s)) {
    log(s.config.debug, "creating pre-render window");
    s.windows[prerenderLabel] = deep.copy({
      ...s.config.defaultWindowProps,
      ...INITIAL_PRERENDER_WINDOW_STATE,
    });
  }
  groupEnd(s.config.debug);
};

const reduceSetWindowStage = assertLabel<SetWindowStagePayload>((s, a) => {
  const win = s.windows[a.payload.label];
  if (win == null) return;
  win.stage = a.payload.stage;
});

const reduceCloseWindow = assertLabel<CloseWindowPayload>(
  (s, { payload: { label } }) => {
    const win = s.windows[label];
    if (win == null || win.processCount > 0) return;
    win.stage = "closing";
    delete s.windows[label];
    delete s.labelKeys[label];
    delete s.keyLabels[win.key];
  },
);

const reduceReloadWindow = assertLabel<ReloadWindowPayload>((s, a) => {
  const win = s.windows[a.payload.label];
  if (win == null || win.processCount > 0) return;
  win.stage = "reloading";
  delayedReload();
});

const reduceRegisterProcess = assertLabel<MaybeKeyPayload>(
  incrementCounter("processCount"),
);

const reduceCompleteProcess = assertLabel<MaybeKeyPayload>((s, a) => {
  incrementCounter("processCount", true)(s, a);
  const win = s.windows[a.payload.label];
  if (win == null) return;
  if (win.processCount === 0)
    if (win.stage === "reloading") delayedReload();
    else {
      s.windows[a.payload.label].visible = false;
      delete s.windows[a.payload.label];
      delete s.labelKeys[a.payload.label];
      delete s.keyLabels[win.key];
    }
});

const reduceSetWindowError = (
  s: SliceState,
  a: PayloadAction<SetWindowErrorPayload>,
): void => {
  const win = s.windows[a.payload.key];
  if (win == null) return;
  win.error = a.payload.message;
};

const reduceFocusWindow = assertLabel<FocusWindowPayload>((s, a) => {
  const win = s.windows[a.payload.label];
  if (win == null) return;
  if (win.visible !== true) win.visible = true;
  incrementCounter("focusCount")(s, a);
});

const reduceSetWindowMinimized = assignBool("minimized");
const reduceSetWindowMaximized = assignBool("maximized");
const reduceSetWindowVisible = assignBool("visible", true);
const reduceSetWindowFullscreen = assignBool("fullscreen", true);
const reduceCenterWindow = assertLabel<CenterWindowPayload>(
  incrementCounter("centerCount"),
);

const reduceSetWindowPosition = assertLabel<SetWindowPositionPayload>((s, a) => {
  s.windows[a.payload.label].position = a.payload.position;
});

const reduceSetWindowSize = assertLabel<SetWindowSizePayload>((s, a) => {
  s.windows[a.payload.label].size = a.payload.size;
});

const reduceSetWindowMinSize = assertLabel<SetWindowMinSizePayload>((s, a) => {
  s.windows[a.payload.label].minSize = a.payload.size;
});

const reduceSetWindowMaxSize = assertLabel<SetWindowMaxSizePayload>((s, a) => {
  s.windows[a.payload.label].maxSize = a.payload.size;
});

const reduceSetWindowResizable = assignBool("resizable");
const reduceSetWindowSkipTaskbar = assignBool("skipTaskbar");
const reduceSetWindowAlwaysOnTop = assignBool("alwaysOnTop");

const reduceSetWindowTitle = assertLabel<SetWindowTitlePayload>((s, a) => {
  s.windows[a.payload.label].title = a.payload.title;
});

const reduceSetWindowDecorations = assignBool("decorations");

const reduceSetWindowProps = (
  s: SliceState,
  a: PayloadAction<RuntimeSetWindowProsPayload>,
): void => {
  const prev = s.windows[a.payload.label];
  const deepPartialEqual = deep.partialEqual(prev, a.payload);
  if (!deepPartialEqual) s.windows[a.payload.label] = { ...prev, ...a.payload };
};

interface InternalSetInitialPayload extends SetConfigPayload, SetWindowLabelPayload {}

export const reduceInternalSetInitial = (
  s: SliceState,
  a: PayloadAction<InternalSetInitialPayload>,
): void => {
  s.config = { ...s.config, ...a.payload };
  s.label = a.payload.label;
  if (s.label === MAIN_WINDOW && s.config.enablePrerender) {
    const prerenderLabel = id.create();
    s.windows[prerenderLabel] = {
      ...s.config.defaultWindowProps,
      ...INITIAL_PRERENDER_WINDOW_STATE,
    };
  }
};

/**
 * The slice definition now references the extracted reducer functions.
 */
const slice = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    internalSetInitial: reduceInternalSetInitial,
    createWindow: reduceCreateWindow,
    setWindowStage: reduceSetWindowStage,
    closeWindow: reduceCloseWindow,
    registerProcess: reduceRegisterProcess,
    completeProcess: reduceCompleteProcess,
    setWindowError: reduceSetWindowError,
    focusWindow: reduceFocusWindow,
    reloadWindow: reduceReloadWindow,
    setWindowMinimized: reduceSetWindowMinimized,
    setWindowMaximized: reduceSetWindowMaximized,
    setWindowVisible: reduceSetWindowVisible,
    setWindowFullscreen: reduceSetWindowFullscreen,
    centerWindow: reduceCenterWindow,
    setWindowPosition: reduceSetWindowPosition,
    setWindowSize: reduceSetWindowSize,
    setWindowMinSize: reduceSetWindowMinSize,
    setWindowMaxSize: reduceSetWindowMaxSize,
    setWindowResizable: reduceSetWindowResizable,
    setWindowSkipTaskbar: reduceSetWindowSkipTaskbar,
    setWindowAlwaysOnTop: reduceSetWindowAlwaysOnTop,
    setWindowTitle: reduceSetWindowTitle,
    setWindowDecorations: reduceSetWindowDecorations,
    runtimeSetWindowProps: reduceSetWindowProps,
    setWindowProps: reduceSetWindowProps as (
      s: SliceState,
      a: PayloadAction<SetWindowPropsPayload>,
    ) => void,
  },
});

export const {
  actions: {
    runtimeSetWindowProps,
    setWindowProps,
    createWindow,
    internalSetInitial,
    setWindowStage,
    closeWindow,
    registerProcess,
    completeProcess,
    setWindowError,
    focusWindow,
    reloadWindow,
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
    setWindowDecorations,
  },
} = slice;

export const reducer: Reducer<SliceState, Action> = slice.reducer;

/**
 * @returns true if the given action type is a drift action.
 * @param type - The action type to check.
 */
export const isDriftAction = (type: string): boolean => type.startsWith(SLICE_NAME);

/** A list of actions that shouldn't be emitted to other windows. */
const EXCLUDED_ACTIONS: string[] = [internalSetInitial.type];

/**
 * @returns true if the action with the given type should be emitted to other
 * windows.
 * @param emitted - Boolean indicating if the action was emitted by another window.
 * @param type - The action type to check.
 */
export const shouldEmit = (emitted: boolean, type: string): boolean =>
  !emitted && !EXCLUDED_ACTIONS.includes(type);
