// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dispatch, PayloadAction } from "@reduxjs/toolkit";
import { dimensions, unique } from "@synnaxlabs/x";

import { log } from "@/debug";
import { MainChecker, Manager, Properties } from "@/runtime";
import { SliceState, setWindowProps, SetWindowPropsPayload } from "@/state";
import { WindowState, MAIN_WINDOW, INITIAL_WINDOW_STATE, WindowProps } from "@/window";

type RequiredRuntime = Manager & MainChecker & Properties;

const purgeWinStateToProps = (
  window: WindowState & { prerenderLabel?: string }
): Omit<WindowProps, "key"> => {
  const {
    centerCount,
    processCount,
    focusCount,
    stage,
    key,
    prerenderLabel,
    reserved,
    minimized,
    ...rest
  } = window;
  return rest;
};

export const syncInitial = async (
  state: SliceState,
  dispatch: Dispatch<PayloadAction<SetWindowPropsPayload>>,
  runtime: RequiredRuntime,
  debug: boolean
): Promise<void> => {
  const runtimeLabels = runtime.listLabels().filter((label) => label !== MAIN_WINDOW);
  const nonMain = Object.keys(state.windows).filter((label) => label !== MAIN_WINDOW);
  log(debug, "syncInitial", state, runtime.listLabels(), nonMain);
  // Create windows that are not in runtime, delete windows that are not in state
  const allLabels = unique([...runtimeLabels, ...nonMain]);
  for (const label of allLabels) {
    // Only the main runtime is allowed to create windows.
    if (!runtimeLabels.includes(label) && runtime.isMain())
      await createRuntimeWindow(runtime, label, state.windows[label], debug);
    else if (!nonMain.includes(label))
      // We're safe to close the window even if we're not in the main runtime
      // because there's no state to maintain.
      await closeRuntimeWindow(runtime, label, debug);
  }
  const label = runtime.label();
  const next = state.windows[label];
  if (next == null) return;
  const initial: WindowState = { ...INITIAL_WINDOW_STATE, key: label };
  await syncCurrent(initial, next, runtime, debug);
  // Make sure our redux store as up to date.
  dispatch(setWindowProps({ label: runtime.label(), ...(await runtime.getProps()) }));
  // startcheckingFullScreen(dispatch, runtime);
};

export const sync = async (
  prev: SliceState,
  next: SliceState,
  runtime: RequiredRuntime,
  debug: boolean
): Promise<void> => {
  log(debug, "sync", prev, next);
  if (runtime.isMain()) await syncMain(prev, next, runtime, debug);
  const prevWin = prev.windows[runtime.label()];
  const nextWin = next.windows[runtime.label()];
  if (prevWin == null || nextWin == null) return;
  await syncCurrent(prevWin, nextWin, runtime, debug);
};

export const syncCurrent = async (
  prevWin: WindowState,
  nextWin: WindowState,
  runtime: RequiredRuntime,
  debug: boolean
): Promise<void> => {
  const changes: Array<[string, Promise<void>]> = [];

  if (nextWin.title != null && nextWin.title !== prevWin.title)
    changes.push(["title", runtime.setTitle(nextWin.title)]);

  const changeVisibility =
    nextWin.visible != null && nextWin.visible !== prevWin.visible;
  const changeVisibilityNow = nextWin.visible === false;
  const changeVisibilityF = (): number =>
    changes.push(["visible", runtime.setVisible(nextWin.visible as boolean)]);

  // If we're making the window invisible, we should make other changes AFTER
  // we make it invisible.
  if (changeVisibility && changeVisibilityNow) changeVisibilityF();

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

  if (
    nextWin.minSize != null &&
    !dimensions.equals(nextWin.minSize, prevWin.minSize)
  )
    changes.push(["minSize", runtime.setMinSize(nextWin.minSize)]);

  if (
    nextWin.maxSize != null &&
    !dimensions.equals(nextWin.maxSize, prevWin.maxSize)
  )
    changes.push(["maxSize", runtime.setMaxSize(nextWin.maxSize)]);

  if (nextWin.size != null && !dimensions.equals(nextWin.size, prevWin.size))
    changes.push(["size", runtime.setSize(nextWin.size)]);

  if (
    nextWin.position != null &&
    !dimensions.equals(nextWin.position, prevWin.position)
  )
    changes.push(["position", runtime.setPosition(nextWin.position)]);

  if (nextWin.focusCount !== prevWin.focusCount)
    changes.push(["setVisible", runtime.setVisible(true)], ["focus", runtime.focus()]);

  if (nextWin.resizable != null && nextWin.resizable !== prevWin.resizable)
    changes.push(["resizable", runtime.setResizable(nextWin.resizable)]);

  if (nextWin.decorations != null && nextWin.decorations !== prevWin.decorations)
    changes.push(["decorations", runtime.setDecorations(nextWin.decorations)]);

  if (nextWin.alwaysOnTop != null && nextWin.alwaysOnTop !== prevWin.alwaysOnTop)
    changes.push(["alwaysOnTop", runtime.setAlwaysOnTop(nextWin.alwaysOnTop)]);

  // If we're going from invisible to visible, we should make other changes BEFORE
  // we make it visible.
  if (changeVisibility && !changeVisibilityNow) changeVisibilityF();

  await Promise.all(
    changes.map(async ([name, change]) => {
      log(debug, "sync", "changing", name);
      return await change;
    })
  );
};

export const syncMain = async (
  prev: SliceState,
  next: SliceState,
  runtime: RequiredRuntime,
  debug: boolean
): Promise<void> => {
  const removed = Object.keys(prev.windows).filter((label) => !(label in next.windows));
  const added = Object.keys(next.windows).filter((label) => !(label in prev.windows));
  const isMain = runtime.isMain();
  if (isMain && removed.length > 0)
    for (const label of removed) {
      log(debug, "syncMain", "closing", label);
      // Close all other windows. It's important to note that we aren't
      // actually removing these windows from state. This is because we
      // may persist old window state when we restart the main window.
      if (label === MAIN_WINDOW)
        await Promise.all(
          Object.keys(next.windows)
            .filter((l) => l !== MAIN_WINDOW)
            .map(async (l) => await closeRuntimeWindow(runtime, l, debug))
        );
      await closeRuntimeWindow(runtime, label, debug);
    }
  if (isMain && added.length > 0)
    for (const label of added)
      await createRuntimeWindow(runtime, label, next.windows[label], debug);
};

const createRuntimeWindow = async (
  runtime: Manager,
  label: string,
  window: WindowState & { prerenderLabel?: string },
  debug: boolean
): Promise<void> => {
  log(debug, "createWindow", window);
  return await runtime.create(label, purgeWinStateToProps(window));
};

const closeRuntimeWindow = async (
  runtime: Manager,
  label: string,
  debug: boolean
): Promise<void> => {
  log(debug, "closeWindow", label);
  return await runtime.close(label);
};
