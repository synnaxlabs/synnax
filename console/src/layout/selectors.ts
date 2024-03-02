// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { type Drift, selectWindow } from "@synnaxlabs/drift";
import { type Haul, type Mosaic, Theming } from "@synnaxlabs/pluto";

import { selectByKey, selectByKeys, useMemoSelect } from "@/hooks";
import { type LayoutState } from "@/layout/layout";
import {
  type SliceState,
  type StoreState,
  SLICE_NAME,
  type NavdrawerEntryState,
  type NavdrawerLocation,
} from "@/layout/slice";

/**
 * Selects the layout state.
 * @param state - The state of the layout store.
 * @returns The layout state.
 */
export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

/**
 * Selects a layout from the store by key.
 *
 * @param state - The store state.
 * @param key - The layout key.
 * @returns The layout. Undefined if not found.
 */
export const select = (state: StoreState, key: string): LayoutState | undefined =>
  selectSliceState(state).layouts[key];

export const selectRequired = (state: StoreState, key: string): LayoutState => {
  const layout = select(state, key);
  if (layout == null) throw new Error(`Layout ${key} not found`);
  return layout;
};

/**
 * Selects a layout from the store by key.
 *
 * @param key - The layout key.
 * @returns The layout. Undefined if not found.
 */
export const useSelect = (key: string): LayoutState | undefined =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const useSelectRequired = (key: string): LayoutState =>
  useMemoSelect((state: StoreState) => selectRequired(state, key), [key]);

/**
 * Selects the central layout mosaic from the store.
 *
 * @param state - The store state.
 * @returns The central layout mosaic.
 */
export const selectMosaic = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): [string, Mosaic.Node] => {
  const win = selectWindow(state, windowKey);
  if (win == null) throw new Error(`Window ${windowKey ?? ""} not found`);
  return [win.key, selectSliceState(state).mosaics[win.key].root];
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
export const selectActiveThemeKey = (state: StoreState): string =>
  selectSliceState(state).activeTheme;

/**
 * Selects the current theme from the store.
 *
 * @param state - The store state.
 * @returns  The current theme.
 */
export const selectTheme = (
  state: StoreState,
  key?: string,
): Theming.Theme | null | undefined => {
  const t = selectByKey(
    selectSliceState(state).themes,
    key,
    selectActiveThemeKey(state),
  );
  if (t == null) return t;
  return Theming.themeZ.parse(t);
};

export const selectRawTheme = (state: StoreState, key?: string): Theming.ThemeSpec => {
  const t = selectByKey(
    selectSliceState(state).themes,
    key,
    selectActiveThemeKey(state),
  );
  if (t == null) throw new UnexpectedError(`Theme ${key} not found`);
  return t;
};

/**
 * Selects the current theme from the store.
 *
 * @returns  The current theme.
 */
export const useSelectTheme = (key?: string): Theming.Theme | null | undefined =>
  useMemoSelect((state: StoreState) => selectTheme(state, key), [key]);

/**
 * Selects layouts from the store by a set of keys. If no keys are provided, all layouts
 * are selected.
 *
 * @param state - The store state.
 * @param keys - The keys of the layouts to select. If not provided, all layouts are
 * selected.
 * @returns The layouts with the given keys.
 */
export const selectMany = (state: StoreState, keys?: string[]): LayoutState[] =>
  selectByKeys<string, LayoutState>(selectSliceState(state).layouts, keys);

/**
 * Selects layouts from the store by a set of keys. If no keys are provided, all layouts
 * are selected.
 *
 * @param keys - The keys of the layouts to select. If not provided, all layouts are
 * @returns The layouts with the given keys.
 */
export const useSelectMany = (keys?: string[]): LayoutState[] =>
  useMemoSelect((state: StoreState) => selectMany(state, keys), [keys]);

export const selectNavDrawer = (
  state: StoreState,
  loc: NavdrawerLocation,
): NavdrawerEntryState => state.layout.nav.drawers[loc];

export const useSelectNavDrawer = (loc: NavdrawerLocation): NavdrawerEntryState =>
  useMemoSelect((state: StoreState) => selectNavDrawer(state, loc), [loc]);

export const selectActiveMosaicTabKey = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): string | null => {
  const win = selectWindow(state, windowKey);
  if (win == null) throw new Error(`Window ${windowKey ?? ""} not found`);
  return selectSliceState(state).mosaics[win.key].activeTab;
};

export const useSelectActiveMosaicTabKey = (): string | null =>
  useMemoSelect(selectActiveMosaicTabKey, []);

export const selectActiveMosaicTab = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): LayoutState | undefined => {
  const activeTabKey = selectActiveMosaicTabKey(state, windowKey);
  if (activeTabKey == null) return undefined;
  return select(state, activeTabKey);
};

export const useSelectActiveMosaicLayout = (): LayoutState | undefined => {
  return useMemoSelect(selectActiveMosaicTab, []);
};

export const selectHauling = (state: StoreState): Haul.DraggingState =>
  selectSliceState(state).hauling;

export const useSelectHauling = (): Haul.DraggingState =>
  useMemoSelect(selectHauling, []);
