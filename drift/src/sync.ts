import { Dispatch } from "@reduxjs/toolkit";

import { closeWindow, DriftAction, DriftState, setWindowError } from "@/state";
import { WindowState, LabeledWindowProps, MAIN_WINDOW } from "@/window";
import { MainChecker, Manager, Properties } from "@/runtime";
import { log } from "./debug";
import { toXYEqual } from "@synnaxlabs/x";

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
    ...rest
  } = window;
  return rest;
};

export const sync = (
  prev: DriftState,
  next: DriftState,
  runtime: RequiredRuntime,
  dispatch: Dispatch<DriftAction>,
  debug: boolean
): void => {
  log(debug, "sync", prev, next);

  if(runtime.isMain()) syncMain(prev, next, runtime, dispatch, debug);

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

export const syncMain = (
  prev: DriftState,
  next: DriftState,
  runtime: RequiredRuntime,
  dispatch: Dispatch<DriftAction>,
  debug: boolean
): void => {
  const removed = Object.keys(prev.windows).filter((label) => !(label in next.windows));
  const added = Object.keys(next.windows).filter((label) => !(label in prev.windows));
  const isMain = runtime.isMain();
  if (isMain && removed.length > 0)
    removed.forEach((label) => {
      log(debug, "syncMain", "closing", label);
      if (label === MAIN_WINDOW)
        // close all other windows
        Object.keys(next.windows)
          .filter((l) => l !== MAIN_WINDOW)
          .forEach((l) => dispatch(closeWindow({ key: l })));
      void runtime.close(label);
    });
  if (isMain && added.length > 0)
    added.forEach((key) => {
      log(debug, "syncMain", "creating", key);
      runtime.create(purgeWinStateToProps(next.windows[key]));
    });
};
