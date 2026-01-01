// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type PayloadAction } from "@reduxjs/toolkit";
import { dimensions, unique, xy } from "@synnaxlabs/x";

import { group, groupEnd, log } from "@/debug";
import { type MainChecker, type Manager, type Properties } from "@/runtime";
import {
  runtimeSetWindowProps,
  type RuntimeSetWindowProsPayload,
  type SliceState,
} from "@/state";
import {
  INITIAL_WINDOW_STATE,
  MAIN_WINDOW,
  type WindowProps,
  type WindowState,
} from "@/window";

type RequiredRuntime = Manager & MainChecker & Properties;

const purgeWinStateToProps = (
  window: WindowState & { prerenderLabel?: string },
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
  dispatch: Dispatch<PayloadAction<RuntimeSetWindowProsPayload>>,
  runtime: RequiredRuntime,
  debug: boolean,
): Promise<void> => {
  const runtimeLabels = (await runtime.listLabels()).filter(
    (label) => label !== MAIN_WINDOW,
  );
  const nonMain = Object.keys(state.windows).filter((label) => label !== MAIN_WINDOW);
  group(debug, "syncInitial");
  log(debug, "existing windows in runtime", runtimeLabels.sort());
  log(debug, "non-main windows in state", nonMain.sort());
  groupEnd(debug);
  // Create windows that are not in runtime, delete windows that are not in state
  const allLabels = unique.unique([...runtimeLabels, ...nonMain]);
  // Only the main runtime is allowed to create windows.
  for (const label of allLabels)
    if (!runtimeLabels.includes(label) && runtime.isMain()) {
      log(debug, "state window not in runtime, creating", label);
      await createRuntimeWindow(runtime, label, state.windows[label], debug);
    } else if (!nonMain.includes(label)) {
      log(debug, "runtime window not in state, closing", label);
      // We're safe to close the window even if we're not in the main runtime
      // because there's no state to maintain.
      await closeRuntimeWindow(runtime, label, debug);
    }
  const label = runtime.label();
  const next = state.windows[label];
  if (next == null) return;
  const initial: WindowState = { ...INITIAL_WINDOW_STATE, key: label };
  await syncCurrent(initial, next, runtime, debug);
  dispatch(
    runtimeSetWindowProps({ label: runtime.label(), ...(await runtime.getProps()) }),
  );
};

export const sync = async (
  prev: SliceState,
  next: SliceState,
  runtime: RequiredRuntime,
  debug: boolean,
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
  debug: boolean,
): Promise<void> => {
  const changes: Array<
    [string, { prev: unknown; next: unknown }, () => Promise<void>]
  > = [];

  if (nextWin.title != null && nextWin.title !== prevWin.title)
    changes.push([
      "title",
      { prev: prevWin.title, next: nextWin.title },
      async () => runtime.setTitle(nextWin.title as string),
    ]);

  const changeVisibility =
    nextWin.visible != null && nextWin.visible !== prevWin.visible;
  const changeVisibilityNow = nextWin.visible === false;
  const changeVisibilityF = (): number =>
    changes.push([
      "visible",
      { prev: prevWin.visible, next: nextWin.visible },
      async () => {
        await runtime.setVisible(nextWin.visible as boolean);
        if (nextWin.visible === false) return;
        let position = nextWin.position;
        position ??= (await runtime.getProps()).position;
        if (position == null) return;
        // This is very much a hack - some runtimes (tauri) won't emit window-created
        // events, so we move the window a bit to emit events in order to do things like
        // hide traffic lights
        await runtime.setPosition(xy.translate(position, { x: 1, y: 1 }));
        await runtime.setPosition(position);
      },
    ]);

  // If we're making the window invisible, we should make other changes AFTER
  // we make it invisible.
  if (changeVisibility && changeVisibilityNow) changeVisibilityF();

  if (nextWin.skipTaskbar != null && nextWin.skipTaskbar !== prevWin.skipTaskbar)
    changes.push([
      "skipTaskbar",
      { prev: prevWin.skipTaskbar, next: nextWin.skipTaskbar },
      async () => await runtime.setSkipTaskbar(nextWin.skipTaskbar as boolean),
    ]);

  if (nextWin.maximized != null && nextWin.maximized !== prevWin.maximized)
    changes.push([
      "maximized",
      { prev: prevWin.maximized, next: nextWin.maximized },
      async () => await runtime.setMaximized(nextWin.maximized as boolean),
    ]);

  if (nextWin.fullscreen != null && nextWin.fullscreen !== prevWin.fullscreen)
    changes.push([
      "fullscreen",
      { prev: prevWin.fullscreen, next: nextWin.fullscreen },
      async () => await runtime.setFullscreen(nextWin.fullscreen as boolean),
    ]);

  if (nextWin.centerCount !== prevWin.centerCount)
    changes.push([
      "center",
      { prev: prevWin.centerCount, next: nextWin.centerCount },
      async () => runtime.center(),
    ]);

  if (nextWin.minimized != null && nextWin.minimized !== prevWin.minimized)
    changes.push([
      "minimized",
      { prev: prevWin.minimized, next: nextWin.minimized },
      async () => await runtime.setMinimized(nextWin.minimized as boolean),
    ]);

  if (nextWin.resizable != null && nextWin.resizable !== prevWin.resizable)
    changes.push([
      "resizable",
      { prev: prevWin.resizable, next: nextWin.resizable },
      async () => await runtime.setResizable(nextWin.resizable as boolean),
    ]);

  if (nextWin.minSize != null && !dimensions.equals(nextWin.minSize, prevWin.minSize))
    changes.push([
      "minSize",
      { prev: prevWin.minSize, next: nextWin.minSize },
      async () => await runtime.setMinSize(nextWin.minSize as dimensions.Dimensions),
    ]);

  if (nextWin.maxSize != null && !dimensions.equals(nextWin.maxSize, prevWin.maxSize))
    changes.push([
      "maxSize",
      { prev: prevWin.maxSize, next: nextWin.maxSize },
      async () => await runtime.setMaxSize(nextWin.maxSize as dimensions.Dimensions),
    ]);

  if (nextWin.size != null && !dimensions.equals(nextWin.size, prevWin.size))
    changes.push([
      "size",
      { prev: prevWin.size, next: nextWin.size },
      async () => await runtime.setSize(nextWin.size as dimensions.Dimensions),
    ]);

  if (
    nextWin.position != null &&
    !dimensions.equals(nextWin.position, prevWin.position)
  )
    changes.push([
      "position",
      { prev: prevWin.position, next: nextWin.position },
      async () => await runtime.setPosition(nextWin.position as xy.XY),
    ]);

  if (nextWin.focusCount !== prevWin.focusCount)
    changes.push(
      [
        "setVisible",
        { prev: prevWin.visible, next: nextWin.visible },
        async () => await runtime.setVisible(true),
      ],
      [
        "focus",
        { prev: prevWin.focusCount, next: nextWin.focusCount },
        async () => await runtime.focus(),
      ],
    );

  if (nextWin.decorations != null && nextWin.decorations !== prevWin.decorations)
    changes.push([
      "decorations",
      { prev: prevWin.decorations, next: nextWin.decorations },
      async () => await runtime.setDecorations(nextWin.decorations as boolean),
    ]);

  if (nextWin.alwaysOnTop != null && nextWin.alwaysOnTop !== prevWin.alwaysOnTop)
    changes.push([
      "alwaysOnTop",
      { prev: prevWin.alwaysOnTop, next: nextWin.alwaysOnTop },
      async () => await runtime.setAlwaysOnTop(nextWin.alwaysOnTop as boolean),
    ]);

  // If we're going from invisible to visible, we should make other changes BEFORE
  // we make it visible.
  if (changeVisibility && !changeVisibilityNow) changeVisibilityF();

  if (changes.length === 0) return;
  group(debug, `syncCurrent, label: ${runtime.label()}, key: ${nextWin.key}`);
  for (const [name, { prev, next }] of changes) log(debug, name, prev, "->", next);
  groupEnd(debug);
  for (const [, , change] of changes) await change();
};

export const syncMain = async (
  prev: SliceState,
  next: SliceState,
  runtime: RequiredRuntime,
  debug: boolean,
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
            .map(async (l) => await closeRuntimeWindow(runtime, l, debug)),
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
  debug: boolean,
): Promise<void> => {
  log(debug, "createWindow", window);
  return await runtime.create(label, purgeWinStateToProps(window));
};

const closeRuntimeWindow = async (
  runtime: Manager,
  label: string,
  debug: boolean,
): Promise<void> => {
  log(debug, "closeWindow", label);
  return await runtime.close(label);
};
