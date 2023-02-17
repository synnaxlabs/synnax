// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { MosaicNode, Theme } from "@synnaxlabs/pluto";

import { Layout } from "../types";

import {
  LayoutState,
  LayoutStoreState,
  LAYOUT_SLICE_NAME,
  NavdrawerEntryState,
  NavdrawerLocation,
} from "./slice";

import { selectByKey, selectByKeys, useMemoSelect } from "@/hooks";

/**
 * Selects the layout state.
 * @param state - The state of the layout store.
 * @returns The layout state.
 */
export const selectLayoutState = (state: LayoutStoreState): LayoutState =>
  state[LAYOUT_SLICE_NAME];

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
): Layout | undefined => selectLayoutState(state).layouts[key];

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
export const selectMosaic = (state: LayoutStoreState): MosaicNode =>
  selectLayoutState(state).mosaic.root;

/**
 * Selects the central layout mosaic from the store.
 *
 * @returns The central layout mosaic.
 */
export const useSelectMosaic = (): MosaicNode => useMemoSelect(selectMosaic, []);

/**
 * Selects the active theme key from the store.
 *
 * @param state - The store state.
 */
export const selectActiveThemeKey = (state: LayoutStoreState): string =>
  selectLayoutState(state).activeTheme;

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
  selectByKey(selectLayoutState(state).themes, key, selectActiveThemeKey(state));

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
  selectByKeys(selectLayoutState(state).layouts, keys);

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
