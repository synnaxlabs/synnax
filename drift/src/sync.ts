import { Dispatch } from "@reduxjs/toolkit";
import { toXYEqual, unique } from "@synnaxlabs/x";

import { log } from "./debug";

import { MainChecker, Manager, Properties } from "@/runtime";
import { DriftAction, DriftState, setWindowError } from "@/state";
import { WindowState, LabeledWindowProps, MAIN_WINDOW } from "@/window";

type RequiredRuntime = Manager & MainChecker & Properties;

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
    minimized,
    ...rest
  } = window;
  return rest;
};

export const syncInitial = (
  state: DriftState,
  dispatch: Dispatch<DriftAction>,
  runtime: RequiredRuntime,
  debug: boolean
): void => {
  const runtimeLabels = runtime.listLabels().filter((label) => label !== MAIN_WINDOW);
  const nonMain = Object.values(state.windows).filter(
    (win) => win.label !== MAIN_WINDOW
  );
  const nonMainLabels = nonMain.map((win) => win.label);
  log(debug, "syncInitial", state, runtime.listLabels(), nonMainLabels);
  // Create windows that are not in runtime, delete windows that are not in state
  unique([...runtimeLabels, ...nonMainLabels]).forEach((label) => {
    // Only the main runtime is allowed to create windows.
    if (!runtimeLabels.includes(label) && runtime.isMain())
      void createRuntimeWindow(runtime, state.windows[label], debug);
    else if (!nonMainLabels.includes(label))
      // We're safe to close the window even if we're not in the main runtime
      // because there's no state to maintain.
      void closeRuntimeWindow(runtime, label, debug);
  });
  const nextWin = state.windows[runtime.label()];
  if (nextWin == null) return;
  syncCurrent(
    {
      label: runtime.label(),
      key: "",
      stage: "created",
      focusCount: 0,
      processCount: 0,
      centerCount: 0,
      reserved: false,
    },
    { ...nextWin, focusCount: 0, processCount: 0, centerCount: 0, stage: "creating" },
    runtime,
    dispatch,
    debug
  );
};

export const sync = (
  prev: DriftState,
  next: DriftState,
  runtime: RequiredRuntime,
  dispatch: Dispatch<DriftAction>,
  debug: boolean
): void => {
  log(debug, "sync", prev, next);
  if (runtime.isMain()) syncMain(prev, next, runtime, debug);
  const prevWin = prev.windows[runtime.label()];
  const nextWin = next.windows[runtime.label()];
  if (prevWin == null || nextWin == null) return;
  syncCurrent(prevWin, nextWin, runtime, dispatch, debug);
};

export const syncCurrent = (
  prevWin: WindowState,
  nextWin: WindowState,
  runtime: RequiredRuntime,
  dispatch: Dispatch<DriftAction>,
  debug: boolean
): void => {
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
    changes.push(["focus", runtime.focus()], ["setVisible", runtime.setVisible(true)]);

  if (nextWin.resizable != null && nextWin.resizable !== prevWin.resizable)
    changes.push(["resizable", runtime.setResizable(nextWin.resizable)]);

  if (nextWin.alwaysOnTop != null && nextWin.alwaysOnTop !== prevWin.alwaysOnTop)
    changes.push(["alwaysOnTop", runtime.setAlwaysOnTop(nextWin.alwaysOnTop)]);

  changes.forEach(([name, change]) => {
    log(debug, "sync", "changing", name);
    void change.catch((e) => dispatch(setWindowError(e)));
  });
};

export const syncMain = (
  prev: DriftState,
  next: DriftState,
  runtime: RequiredRuntime,
  debug: boolean
): void => {
  const removed = Object.keys(prev.windows).filter((label) => !(label in next.windows));
  const added = Object.keys(next.windows).filter((label) => !(label in prev.windows));
  const isMain = runtime.isMain();
  if (isMain && removed.length > 0)
    removed.forEach((label) => {
      log(debug, "syncMain", "closing", label);
      void (async () => {
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
      })();
    });
  if (isMain && added.length > 0)
    added.forEach((key) => {
      void createRuntimeWindow(runtime, next.windows[key], debug);
    });
};

const createRuntimeWindow = async (
  runtime: Manager,
  window: WindowState & { prerenderLabel?: string },
  debug: boolean
): Promise<void> => {
  log(debug, "createWindow", window);
  return runtime.create(purgeWinStateToProps(window));
};

const closeRuntimeWindow = async (
  runtime: Manager,
  label: string,
  debug: boolean
): Promise<void> => {
  log(debug, "closeWindow", label);
  return await runtime.close(label);
};
