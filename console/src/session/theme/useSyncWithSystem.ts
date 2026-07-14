// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { useAsyncEffect } from "@synnaxlabs/pluto";
import { type destructor } from "@synnaxlabs/x";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { useDispatch } from "react-redux";

import { Runtime } from "@/session/runtime";
import { useSelectSyncWithSystem } from "@/session/theme/selectors";
import { type Action, select } from "@/session/theme/slice";

const matchThemeChange = (theme: string | null): "synnaxDark" | "synnaxLight" =>
  theme === "dark" ? "synnaxDark" : "synnaxLight";

const syncWithOS = async (
  dispatch: Dispatch<Action>,
): Promise<destructor.Destructor> => {
  if (Runtime.ENGINE !== "tauri") {
    const query = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = (e: MediaQueryListEvent) =>
      dispatch(select(matchThemeChange(e.matches ? "dark" : "light")));
    query.addEventListener("change", handler);
    return () => query.removeEventListener("change", handler);
  }
  return await getCurrentWindow().onThemeChanged(({ payload }) =>
    dispatch(select(matchThemeChange(payload))),
  );
};

const setInitial = async (dispatch: Dispatch<Action>): Promise<void> => {
  if (Runtime.ENGINE !== "tauri") {
    const theme = window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "dark"
      : "light";
    dispatch(select(matchThemeChange(theme)));
    return;
  }
  dispatch(select(matchThemeChange(await getCurrentWindow().theme())));
};

/**
 * useSyncWithSystem keeps the selected theme in sync with the OS color scheme. It
 * sets the initial theme on mount and subscribes to subsequent OS theme changes,
 * dispatching into the shared theme slice so the change propagates to every window.
 *
 * Only the main window runs the sync; child windows receive the resulting action via
 * Drift's cross-window propagation. Sync is skipped entirely (and torn down if
 * already running) while syncWithSystem is disabled in the theme slice.
 */
export const useSyncWithSystem = (): void => {
  const dispatch = useDispatch<Dispatch<Action>>();
  const enabled = useSelectSyncWithSystem();
  useAsyncEffect(
    async (signal) => {
      if (!enabled || !Runtime.isMainWindow()) return;
      await setInitial(dispatch);
      if (signal.aborted) return;
      return await syncWithOS(dispatch);
    },
    [dispatch, enabled],
  );
};
