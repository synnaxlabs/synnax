// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction, type UnknownAction } from "@reduxjs/toolkit";
import { Drift, selectWindowKey } from "@synnaxlabs/drift";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  type AsyncDestructor,
  type Icon,
  type Nav,
  Text,
  Theming,
  Triggers,
  useAsyncEffect,
  useDebouncedCallback,
  useMemoCompare,
} from "@synnaxlabs/pluto";
import { compare } from "@synnaxlabs/x";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { type Dispatch, useCallback, useState } from "react";
import { useDispatch, useStore } from "react-redux";

import { useOpenInNewWindow } from "@/layout/Menu";
import {
  selectActiveMosaicTabKey,
  selectFocused,
  useSelectNavDrawer,
  useSelectTheme,
} from "@/layout/selectors";
import {
  type NavDrawerLocation,
  place,
  remove,
  resizeNavDrawer,
  setActiveTheme,
  setFocus,
  setNavDrawerVisible,
  type State,
  toggleActiveTheme,
} from "@/layout/slice";
import { type RootAction, type RootState, type RootStore } from "@/store";

export interface CreatorProps {
  windowKey: string;
  dispatch: Dispatch<PayloadAction<any>>;
  store: RootStore;
}

type StateWithoutWindowKey = Omit<State, "windowKey">;

/** A function that creates a layout given a set of utilities. */
export type Creator = (props: CreatorProps) => StateWithoutWindowKey;

export type PlacerProps = StateWithoutWindowKey | Creator;

/** A function that places a layout using the given properties or creation func. */
export type Placer = (layout: PlacerProps) => {
  windowKey: string;
  key: string;
};

/** A function that removes a layout. */
export type Remover = (...keys: string[]) => void;

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
export const usePlacer = (): Placer => {
  const dispatch = useDispatch();
  const store = useStore<RootState, RootAction>();
  const windowKey = useSelectWindowKey();
  return useCallback(
    (base) => {
      if (windowKey == null) throw new Error("windowKey is null");
      const layout =
        typeof base === "function" ? base({ dispatch, store, windowKey }) : base;
      const { key } = layout;
      dispatch(place({ ...layout, windowKey }));
      return { windowKey, key };
    },
    [dispatch, windowKey],
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
export const useRemover = (...baseKeys: string[]): Remover => {
  const dispatch = useDispatch();
  const memoKeys = useMemoCompare(
    () => baseKeys,
    ([a], [b]) => compare.primitiveArrays(a, b) === compare.EQUAL,
    [baseKeys],
  );
  return useCallback(
    (...keys) => dispatch(remove({ keys: [...keys, ...memoKeys] })),
    [memoKeys],
  );
};

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

export const useErrorThemeProvider = (): Theming.ProviderProps => {
  const [theme, setTheme] = useState<Theming.ThemeSpec | null>(Theming.SYNNAX_LIGHT);
  useAsyncEffect(async () => {
    const theme = matchThemeChange({ payload: await getCurrentWindow().theme() });
    setTheme(Theming.SYNNAX_THEMES[theme]);
  }, []);
  return {
    theme: Theming.themeZ.parse(theme),
    setTheme: (key: string) =>
      setTheme(Theming.SYNNAX_THEMES[key as keyof typeof Theming.SYNNAX_THEMES]),
    toggleTheme: () =>
      setTheme((t) =>
        t?.key === Theming.SYNNAX_LIGHT.key
          ? Theming.SYNNAX_DARK
          : Theming.SYNNAX_LIGHT,
      ),
  };
};

const matchThemeChange = ({
  payload: theme,
}: {
  payload: string | null;
}): keyof typeof Theming.SYNNAX_THEMES =>
  theme === "dark" ? "synnaxDark" : "synnaxLight";

const synchronizeWithOS = async (
  dispatch: Dispatch<UnknownAction>,
): Promise<AsyncDestructor> =>
  await getCurrentWindow().onThemeChanged((e) =>
    dispatch(setActiveTheme(matchThemeChange(e))),
  );

const setInitialTheme = async (dispatch: Dispatch<UnknownAction>): Promise<void> =>
  dispatch(
    setActiveTheme(matchThemeChange({ payload: await getCurrentWindow().theme() })),
  );

export interface NavMenuItem {
  key: string;
  icon: Icon.Element;
  tooltip: string;
}

export interface NavDrawerItem extends Nav.DrawerItem, NavMenuItem {}

export interface UseNavDrawerReturn {
  activeItem: NavDrawerItem | undefined;
  menuItems: NavMenuItem[];
  onSelect: (item: string) => void;
  onResize: (size: number) => void;
}

export const useNavDrawer = (
  location: NavDrawerLocation,
  items: NavDrawerItem[],
): UseNavDrawerReturn => {
  const windowKey = useSelectWindowKey() as string;
  const state = useSelectNavDrawer(location);
  const dispatch = useDispatch();
  const onResize = useDebouncedCallback(
    (size) => {
      dispatch(resizeNavDrawer({ windowKey, location, size }));
    },
    100,
    [dispatch, windowKey],
  );
  if (state == null)
    return {
      activeItem: undefined,
      menuItems: [],
      onSelect: () => {},
      onResize: () => {},
    };
  let activeItem: NavDrawerItem | undefined;
  if (state.activeItem != null)
    activeItem = items.find((item) => item.key === state.activeItem);
  const menuItems = items.filter((item) => state.menuItems.includes(item.key));

  if (activeItem != null) activeItem.initialSize = state.size;

  return {
    activeItem,
    menuItems,
    onSelect: (key: string) => dispatch(setNavDrawerVisible({ windowKey, key })),
    onResize,
  };
};

export const useTriggers = () => {
  const store = useStore<RootState>();
  const remove = useRemover();
  const openInNewWindow = useOpenInNewWindow();
  Triggers.use({
    triggers: [["Control", "L"]],
    loose: true,
    callback: ({ stage }) => {
      if (stage !== "start") return;
      const state = store.getState();
      const active = selectActiveMosaicTabKey(state);
      const windowKey = selectWindowKey(state);
      const { focused } = selectFocused(state);
      if (active == null || windowKey == null) return;
      if (focused != null) store.dispatch(setFocus({ key: null, windowKey }));
      else store.dispatch(setFocus({ key: active, windowKey }));
    },
  });
  Triggers.use({
    triggers: [["Control", "W"]],
    loose: true,
    callback: ({ stage }) => {
      if (stage !== "start") return;
      const state = store.getState();
      const active = selectActiveMosaicTabKey(state);
      if (active == null) return;
      remove(active);
    },
  });
  Triggers.use({
    triggers: [["Control", "O"]],
    loose: true,
    callback: ({ stage }) => {
      if (stage !== "start") return;
      const state = store.getState();
      const active = selectActiveMosaicTabKey(state);
      if (active == null) return;
      openInNewWindow(active);
    },
  });
  Triggers.use({
    triggers: [["Control", "E"]],
    loose: true,
    callback: ({ stage }) => {
      if (stage !== "start") return;
      const state = store.getState();
      const active = selectActiveMosaicTabKey(state);
      if (active == null) return;
      Text.edit(`pluto-tab-${active}`);
    },
  });
};
