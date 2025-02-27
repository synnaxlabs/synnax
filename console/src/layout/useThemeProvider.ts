// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnknownAction } from "@reduxjs/toolkit";
import { Drift } from "@synnaxlabs/drift";
import { type AsyncDestructor, Theming, useAsyncEffect } from "@synnaxlabs/pluto";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { type Dispatch } from "react";
import { useDispatch } from "react-redux";

import { useSelectTheme } from "@/layout/selectors";
import { setActiveTheme, toggleActiveTheme } from "@/layout/slice";

/**
 * useThemeProvider is a hook that returns the props to pass to a ThemeProvider from
 * @synnaxlabs/pluto. This hook allows theme management to be centralized in the layout
 * redux store, and be synchronized across several windows.
 *
 * @returns The props to pass to a ThemeProvider from @synnaxlabs/pluto.
 */
export const useThemeProvider = (): Theming.ProviderProps => {
  const theme = useSelectTheme();
  const dispatch = useDispatch();

  useAsyncEffect(async () => {
    if (getCurrentWindow().label !== Drift.MAIN_WINDOW) return;
    await setInitialTheme(dispatch);
    const cleanup = await synchronizeWithOS(dispatch);
    return cleanup;
  }, []);

  return {
    theme: Theming.themeZ.parse(theme),
    setTheme: (key: string) => dispatch(setActiveTheme(key)),
    toggleTheme: () => dispatch(toggleActiveTheme()),
  };
};

const matchThemeChange = (theme: string | null): keyof typeof Theming.SYNNAX_THEMES =>
  theme === "dark" ? "synnaxDark" : "synnaxLight";

const synchronizeWithOS = async (
  dispatch: Dispatch<UnknownAction>,
): Promise<AsyncDestructor> =>
  await getCurrentWindow().onThemeChanged(({ payload }) =>
    dispatch(setActiveTheme(matchThemeChange(payload))),
  );

const setInitialTheme = async (dispatch: Dispatch<UnknownAction>): Promise<void> =>
  dispatch(setActiveTheme(matchThemeChange(await getCurrentWindow().theme())));
