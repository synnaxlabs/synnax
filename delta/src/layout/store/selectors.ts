// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DriftStoreState, selectWindow } from "@synnaxlabs/drift";
import { Haul, Mosaic, Theming } from "@synnaxlabs/pluto";

import { selectByKey, selectByKeys, useMemoSelect } from "@/hooks";
import { LayoutState } from "@/layout/types";

import {
  LayoutSliceState,
  LayoutStoreState,
  LAYOUT_SLICE_NAME,
  NavdrawerEntryState,
  NavdrawerLocation,
} from "./slice";

/**
 * Selects the layout state.
 * @param state - The state of the layout store.
 * @returns The layout state.
 */
export const selectLayoutState = (state: LayoutStoreState): LayoutSliceState =>
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
): LayoutState | undefined => {
  return selectLayoutState(state).layouts[key];
};

/**
 * Selects a layout from the store by key.
 *
 * @param key - The layout key.
 * @returns The layout. Undefined if not found.
 */
export const useSelectLayout = (key: string): LayoutState | undefined =>
  useMemoSelect((state: LayoutStoreState) => selectLayout(state, key), [key]);

export const useSelectRequiredLayout = (key: string): LayoutState => {
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
export const selectMosaic = (
  state: LayoutStoreState & DriftStoreState,
  windowKey?: string
): [string, Mosaic.Node] => {
  const win = selectWindow(state, windowKey);
  if (win == null) throw new Error(`Window ${windowKey ?? ""} not found`);
  return [win.key, selectLayoutState(state).mosaics[win.key].root];
};

/**
 * Selects the central layout mosaic from the store.
 *
 * @returns The central layout mosaic.
 */
export const useSelectMosaic = (): [string, Mosaic.Node] =>
  useMemoSelect(selectMosaic, []);

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
): Theming.Theme | null | undefined => {
  const t = selectByKey(
    selectLayoutState(state).themes,
    key,
    selectActiveThemeKey(state)
  );
  if (t == null) return t;
  return Theming.themeZ.parse(t);
};

/**
 * Selects the current theme from the store.
 *
 * @returns  The current theme.
 */
export const useSelectTheme = (key?: string): Theming.Theme | null | undefined =>
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
export const selectLayouts = (
  state: LayoutStoreState,
  keys?: string[]
): LayoutState[] =>
  selectByKeys<string, LayoutState>(selectLayoutState(state).layouts, keys);

/**
 * Selects layouts from the store by a set of keys. If no keys are provided, all layouts
 * are selected.
 *
 * @param keys - The keys of the layouts to select. If not provided, all layouts are
 * @returns The layouts with the given keys.
 */
export const useSelectLayouts = (keys?: string[]): LayoutState[] =>
  useMemoSelect((state: LayoutStoreState) => selectLayouts(state, keys), [keys]);

export const selectNavDrawer = (
  state: LayoutStoreState,
  loc: NavdrawerLocation
): NavdrawerEntryState => state.layout.nav.drawers[loc];

export const useSelectNavDrawer = (loc: NavdrawerLocation): NavdrawerEntryState =>
  useMemoSelect((state: LayoutStoreState) => selectNavDrawer(state, loc), [loc]);

export const selectActiveMosaicTabKey = (
  state: LayoutStoreState & DriftStoreState,
  windowKey?: string
): string | null => {
  const win = selectWindow(state, windowKey);
  if (win == null) throw new Error(`Window ${windowKey ?? ""} not found`);
  return selectLayoutState(state).mosaics[win.key].activeTab;
};

export const useSelectActiveMosaicTabKey = (): string | null =>
  useMemoSelect(selectActiveMosaicTabKey, []);

export const selectActiveMosaicLayout = (
  state: LayoutStoreState & DriftStoreState,
  windowKey?: string
): LayoutState | undefined => {
  const activeTabKey = selectActiveMosaicTabKey(state, windowKey);
  if (activeTabKey == null) return undefined;
  return selectLayout(state, activeTabKey);
};

export const useSelectActiveMosaicLayout = (): LayoutState | undefined => {
  return useMemoSelect(selectActiveMosaicLayout, []);
};

export const selectHauling = (state: LayoutStoreState): Haul.Item[] =>
  selectLayoutState(state).hauling;

export const useSelectHauling = (): Haul.Item[] => useMemoSelect(selectHauling, []);
