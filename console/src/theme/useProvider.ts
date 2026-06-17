// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnknownAction } from "@reduxjs/toolkit";
import { Theming, useAsyncEffect } from "@synnaxlabs/pluto";
import { type destructor } from "@synnaxlabs/x";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { type Dispatch } from "react";
import { useDispatch } from "react-redux";

import { Runtime } from "@/runtime";
import { useSelectTheme } from "@/theme/selectors";
import { setActive, toggle } from "@/theme/slice";

/**
 * useProvider returns the props to pass to the Pluto Provider's `theming` slot. It
 * reads the active theme from the theme slice and synchronizes it with the OS color
 * scheme.
 */
export const useProvider = (): Theming.ProviderProps => {
  const theme = useSelectTheme();
  const dispatch = useDispatch();
  useAsyncEffect(async (signal) => {
    if (!Runtime.isMainWindow()) return;
    await setInitialTheme(dispatch);
    if (signal.aborted) return;
    return await synchronizeWithOS(dispatch);
  }, [dispatch]);
  return {
    theme: Theming.themeZ.parse(theme),
    setTheme: (key: string) => dispatch(setActive(key)),
    toggleTheme: () => dispatch(toggle()),
  };
};

const matchThemeChange = (theme: string | null): keyof typeof Theming.SYNNAX_THEMES =>
  theme === "dark" ? "synnaxDark" : "synnaxLight";

const synchronizeWithOS = async (
  dispatch: Dispatch<UnknownAction>,
): Promise<destructor.Destructor> => {
  if (Runtime.ENGINE !== "tauri") {
    const query = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = (e: MediaQueryListEvent) =>
      dispatch(setActive(matchThemeChange(e.matches ? "dark" : "light")));
    query.addEventListener("change", handler);
    return () => query.removeEventListener("change", handler);
  }
  return await getCurrentWindow().onThemeChanged(({ payload }) =>
    dispatch(setActive(matchThemeChange(payload))),
  );
};

const setInitialTheme = async (dispatch: Dispatch<UnknownAction>): Promise<void> => {
  if (Runtime.ENGINE !== "tauri") {
    const theme = window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "dark"
      : "light";
    dispatch(setActive(matchThemeChange(theme)));
    return;
  }
  dispatch(setActive(matchThemeChange(await getCurrentWindow().theme())));
};
