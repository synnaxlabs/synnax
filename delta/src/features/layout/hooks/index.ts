// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dispatch, useCallback } from "react";

import type { AnyAction } from "@reduxjs/toolkit";
import { closeWindow, createWindow, MAIN_WINDOW } from "@synnaxlabs/drift";
import type {
  NavDrawerItem,
  UseNavDrawerReturn,
  ThemeProviderProps,
  NavMenuItem,
  NavDrawerContent,
} from "@synnaxlabs/pluto";
import { appWindow } from "@tauri-apps/api/window";
import type { Theme as TauriTheme } from "@tauri-apps/api/window";
import { useDispatch } from "react-redux";

import {
  NavdrawerLocation,
  placeLayout,
  removeLayout,
  setActiveTheme,
  setNavdrawerEntryState,
  toggleActiveTheme,
  useSelectLayout,
  useSelectNavDrawer,
  useSelectTheme,
} from "../store";
import { Layout } from "../types";

import { useAsyncEffect, AsyncDestructor } from "@/hooks";

export interface LayoutCreatorProps {
  dispatch: Dispatch<AnyAction>;
}

/** A function that creates a layout given a set of utilities. */
export type LayoutCreator = (props: LayoutCreatorProps) => Layout;

/** A function that places a layout using the given properties or creation func. */
export type LayoutPlacer = (layout: Layout | LayoutCreator) => void;

/** A function that removes a layout. */
export type LayoutRemover = () => void;

/**
 * useLayoutPlacer is a hook that returns a function that allows the caller to place
 * a layout in the central mosaic or in a window.
 *
 * @returns A layout placer function that allows the caller to open a layout using one
 * of two methods. The first is to pass a layout object with the layout's key, type,
 * title, location, and window properties. The second is to pass a layout creator function
 * that accepts a few utilities and returns a layout object. Prefer the first method
 * when possible, but feel free to use the second method for more dynamic layout creation.
 */
export const useLayoutPlacer = (): LayoutPlacer => {
  const dispatch = useDispatch();
  return useCallback(
    (layout_: Layout | LayoutCreator) => {
      const layout = typeof layout_ === "function" ? layout_({ dispatch }) : layout_;
      const { key, location, window, title } = layout;
      dispatch(placeLayout(layout));
      if (location === "window")
        dispatch(
          createWindow({
            ...{ ...window, navTop: undefined },
            url: "/",
            key,
            title,
          })
        );
    },
    [dispatch]
  );
};

/**
 * useLayoutRemover is a hook that returns a function that allows the caller to remove
 * a layout.
 *
 * @param key - The key of the layout to remove.
 * @returns A layout remover function that allows the caller to remove a layout. If
 * the layout is in a window, the window will also be closed.
 */
export const useLayoutRemover = (key: string): LayoutRemover => {
  const dispatch = useDispatch();
  const layout = useSelectLayout(key);
  if (layout == null) throw new Error(`layout with key ${key} does not exist`);
  return () => {
    dispatch(removeLayout(key));
    if (layout.location === "window") dispatch(closeWindow(key));
  };
};

/**
 * useThemeProvider is a hook that returns the props to pass to a ThemeProvider from
 * @synnaxlabs/pluto. This hook allows theme management to be centralized in the layout
 * redux store, and be synchronized across several windows.
 *
 * @returns The props to pass to a ThemeProvider from @synnaxlabs/pluto.
 */
export const useThemeProvider = (): ThemeProviderProps => {
  const theme = useSelectTheme();
  const dispatch = useDispatch();

  useAsyncEffect(async (): AsyncDestructor => {
    if (appWindow.label !== MAIN_WINDOW) return;
    await setInitialTheme(dispatch);
    const cleanup = await synchronizeWithOS(dispatch);
    return cleanup;
  }, []);

  return {
    theme,
    setTheme: (key: string) => dispatch(setActiveTheme(key)),
    toggleTheme: () => dispatch(toggleActiveTheme()),
  };
};

const matchThemeChange = ({ payload: theme }: { payload: TauriTheme | null }): string =>
  theme === "dark" ? "synnaxDark" : "synnaxLight";

const synchronizeWithOS = async (dispatch: Dispatch<AnyAction>): AsyncDestructor => {
  return await appWindow.onThemeChanged((e) =>
    dispatch(setActiveTheme(matchThemeChange(e)))
  );
};

const setInitialTheme = async (dispatch: Dispatch<AnyAction>): Promise<void> => {
  const t = await appWindow.theme();
  dispatch(setActiveTheme(matchThemeChange({ payload: t })));
};

export const useNavDrawer = (
  loc: NavdrawerLocation,
  items: NavDrawerItem[]
): UseNavDrawerReturn => {
  const state = useSelectNavDrawer(loc);
  const dispatch = useDispatch();
  let activeItem: NavDrawerContent | undefined;
  let menuItems: NavMenuItem[] = [];
  if (state.activeItem != null)
    activeItem = items.find((item) => item.key === state.activeItem);
  menuItems = items.filter((item) => state.menuItems.includes(item.key));
  return {
    activeItem,
    menuItems,
    onSelect: (item: string) =>
      dispatch(
        setNavdrawerEntryState({
          location: loc,
          state: {
            activeItem: item === state.activeItem ? null : item,
          },
        })
      ),
  };
};
