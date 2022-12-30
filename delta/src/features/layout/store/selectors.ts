import type { MosaicLeaf, Theme } from "@synnaxlabs/pluto";

import { Layout } from "../types";

import { LayoutStoreState, NavdrawerEntryState, NavdrawerLocation } from "./slice";

import { selectByKey, selectByKeys, useMemoSelect } from "@/hooks";

/**
 * Selects a layout from the store by key.
 *
 * @param state - The store state.
 * @param key - The layout key.
 * @returns The layout. Undefined if not found.
 */
export const selectLayout = (
  state: LayoutStoreState,
  key: string
): Layout | undefined => {
  return state.layout.layouts[key];
};

/**
 * Selects a layout from the store by key.
 *
 * @param key - The layout key.
 * @returns The layout. Undefined if not found.
 */
export const useSelectLayout = (key: string): Layout | undefined =>
  useMemoSelect((state: LayoutStoreState) => selectLayout(state, key), [key]);

export const useSelectRequiredLayout = (key: string): Layout => {
  const layout = useSelectLayout(key);
  if (layout == null) throw new Error(`Layout ${key} not found`);
  return layout;
};

/**
 * Selects the central layout mosaic from the store.
 *
 * @param state - The store state.
 * @returns The central layout mosaic.
 */
export const selectMosaic = (state: LayoutStoreState): MosaicLeaf =>
  state.layout.mosaic.root;

/**
 * Selects the central layout mosaic from the store.
 *
 * @returns The central layout mosaic.
 */
export const useSelectMosaic = (): MosaicLeaf => useMemoSelect(selectMosaic, []);

/**
 * Selects the active theme key from the store.
 *
 * @param state - The store state.
 */
export const selectActiveThemeKey = (state: LayoutStoreState): string =>
  state.layout.activeTheme;

/**
 * Selects the current theme from the store.
 *
 * @param state - The store state.
 * @returns  The current theme.
 */
export const selectTheme = (
  state: LayoutStoreState,
  key?: string
): Theme | null | undefined =>
  selectByKey(state.layout.themes, key, selectActiveThemeKey(state));

/**
 * Selects the current theme from the store.
 *
 * @returns  The current theme.
 */
export const useSelectTheme = (key?: string): Theme | null | undefined =>
  useMemoSelect((state: LayoutStoreState) => selectTheme(state, key), [key]);

/**
 * Selects layouts from the store by a set of keys. If no keys are provided, all layouts
 * are selected.
 *
 * @param state - The store state.
 * @param keys - The keys of the layouts to select. If not provided, all layouts are
 * selected.
 * @returns The layouts with the given keys.
 */
export const selectLayouts = (state: LayoutStoreState, keys?: string[]): Layout[] =>
  selectByKeys(state.layout.layouts, keys);

/**
 * Selects layouts from the store by a set of keys. If no keys are provided, all layouts
 * are selected.
 *
 * @param keys - The keys of the layouts to select. If not provided, all layouts are
 * @returns The layouts with the given keys.
 */
export const useSelectLayouts = (keys?: string[]): Layout[] =>
  useMemoSelect((state: LayoutStoreState) => selectLayouts(state, keys), [keys]);

export const selectNavDrawer = (
  state: LayoutStoreState,
  loc: NavdrawerLocation
): NavdrawerEntryState => state.layout.nav.drawer[loc];

export const useSelectNavDrawer = (loc: NavdrawerLocation): NavdrawerEntryState =>
  useMemoSelect((state: LayoutStoreState) => selectNavDrawer(state, loc), [loc]);
