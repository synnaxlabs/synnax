// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { type Drift, selectWindow, selectWindowKey } from "@synnaxlabs/drift";
import { Color, type Haul, type Mosaic, Theming } from "@synnaxlabs/pluto";

import { selectByKey, selectByKeys, useMemoSelect } from "@/hooks";
import {
  type NavDrawerEntryState,
  type NavDrawerLocation,
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
} from "@/layout/slice";

/**
 * Selects the layout state.
 * @param state - The state of the layout store.
 * @returns The layout state.
 */
export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const useSelectSliceState = (): SliceState =>
  useMemoSelect((state: StoreState) => selectSliceState(state), []);

/**
 * Selects a layout from the store by key.
 *
 * @param state - The store state.
 * @param key - The layout key.
 * @returns The layout. Undefined if not found.
 */
export const select = (state: StoreState, key: string): State | undefined =>
  selectSliceState(state).layouts[key];

export const selectRequired = (state: StoreState, key: string): State => {
  const layout = select(state, key);
  if (layout == null) throw new Error(`Layout ${key} not found`);
  return layout;
};

export const selectType = (state: StoreState, key: string): string | undefined =>
  select(state, key)?.type;

export const useSelectType = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectType(state, key), [key]);

export const selectArgs = <A>(state: StoreState, key: string): A => {
  const layout = select(state, key);
  return layout?.args as A;
};

export const useSelectArgs = <A>(key: string): A =>
  useMemoSelect((state: StoreState) => selectArgs(state, key), [key]);

export const selectAltKey = (state: StoreState, key: string): string | undefined =>
  selectSliceState(state).keyToAltKey[key];

export const useSelectAltKey = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectAltKey(state, key), [key]);

const selectName = (state: StoreState, key: string): string | undefined =>
  select(state, key)?.name;

const selectRequiredName = (state: StoreState, key: string): string =>
  selectRequired(state, key).name;

export const useSelectName = (key: string): string | undefined =>
  useMemoSelect((state: StoreState) => selectName(state, key), [key]);

export const useSelectRequiredName = (key: string): string =>
  useMemoSelect((state: StoreState) => selectRequiredName(state, key), [key]);

export const selectByFilter = (
  state: StoreState,
  filter: (layout: State) => boolean,
): State | undefined => Object.values(selectSliceState(state).layouts).find(filter);

/**
 * Selects a layout from the store by key.
 *
 * @param key - The layout key.
 * @returns The layout. Undefined if not found.
 */
export const useSelect = (key: string): State | undefined =>
  useMemoSelect((state: StoreState) => select(state, key), [key]);

export const useSelectRequired = (key: string): State =>
  useMemoSelect((state: StoreState) => selectRequired(state, key), [key]);

export const selectModals = (state: StoreState): State[] =>
  Object.values(state[SLICE_NAME].layouts).filter(
    ({ location }) => location === "modal",
  );

export const useSelectModals = (): State[] => useMemoSelect(selectModals, []);

export const selectWindowModals = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): State[] => {
  const winKey = selectWindowKey(state, windowKey);
  if (winKey == null) return [];
  return selectModals(state).filter(({ windowKey }) => windowKey === winKey);
};

export const useSelectWindowModals = (): State[] =>
  useMemoSelect(selectWindowModals, []);

/**
 * Selects the central layout mosaic from the store.
 *
 * @param state - The store state.
 * @returns The central layout mosaic.
 */
export const selectMosaic = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): [string, Mosaic.Node] | [null, null] => {
  const winKey = selectWindowKey(state, windowKey);
  if (winKey == null) return [null, null];
  return [winKey, selectSliceState(state).mosaics[winKey].root];
};

export interface UseSelectFocusedReturn {
  windowKey: string | null;
  focused: string | null;
}

export const selectFocused = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): UseSelectFocusedReturn => {
  const win = selectWindow(state, windowKey);
  if (win == null) return { windowKey: null, focused: null };
  return {
    windowKey: win.key,
    focused: selectSliceState(state).mosaics[win.key]?.focused ?? null,
  };
};

export const useSelectFocused = (): UseSelectFocusedReturn =>
  useMemoSelect(selectFocused, []);

/**
 * Selects the central layout mosaic from the store.
 *
 * @returns The central layout mosaic.
 */
export const useSelectMosaic = (): [string, Mosaic.Node] | [null, null] =>
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
export const selectMany = (state: StoreState, keys?: string[]): State[] =>
  selectByKeys<string, State>(selectSliceState(state).layouts, keys);

/**
 * Selects layouts from the store by a set of keys. If no keys are provided, all layouts
 * are selected.
 *
 * @param keys - The keys of the layouts to select. If not provided, all layouts are
 * @returns The layouts with the given keys.
 */
export const useSelectMany = (keys?: string[]): State[] =>
  useMemoSelect((state: StoreState) => selectMany(state, keys), [keys]);

export const selectNavDrawer = (
  state: StoreState & Drift.StoreState,
  loc: NavDrawerLocation,
): NavDrawerEntryState | null => {
  const winKey = selectWindowKey(state) as string;
  const navState = selectSliceState(state).nav[winKey];
  if (navState == null) return null;
  return navState.drawers[loc] ?? null;
};

export const useSelectNavDrawer = (
  loc: NavDrawerLocation,
): NavDrawerEntryState | null =>
  useMemoSelect(
    (state: StoreState & Drift.StoreState) => selectNavDrawer(state, loc),
    [loc],
  );

export interface SelectActiveMosaicTabState {
  layoutKey: string | null;
  blurred: boolean;
}

export const selectActiveMosaicTabState = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): SelectActiveMosaicTabState => {
  const winKey = selectWindowKey(state, windowKey);
  if (winKey == null) return { layoutKey: null, blurred: false };
  const sliceState = selectSliceState(state);
  const hasModals = Object.values(sliceState.layouts).some(
    (l) => l.location === "modal" && l.windowKey === winKey,
  );
  return {
    layoutKey: sliceState.mosaics[winKey].activeTab,
    blurred: hasModals,
  };
};

export const selectActiveMosaicTabKeyAndNotBlurred = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): string | null => {
  const active = selectActiveMosaicTabState(state, windowKey);
  if (active.layoutKey == null) return null;
  if (active.blurred) return null;
  return active.layoutKey;
};

export const useSelectActiveMosaicTabKeyAndNotBlurred = (): string | null =>
  useMemoSelect(selectActiveMosaicTabKeyAndNotBlurred, []);

export const selectActiveMosaicTabName = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): string | null => {
  const active = selectActiveMosaicTabState(state, windowKey);
  if (active.layoutKey == null) return null;
  return select(state, active.layoutKey)?.name ?? null;
};

export const useSelectActiveMosaicTabState = (): SelectActiveMosaicTabState =>
  useMemoSelect(selectActiveMosaicTabState, []);

export const useSelectActiveMosaicTabName = (): string | null =>
  useMemoSelect(selectActiveMosaicTabName, []);

export const selectActiveMosaicLayout = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): State | undefined => {
  const activeTabKey = selectActiveMosaicTabState(state, windowKey);
  if (activeTabKey.layoutKey == null) return undefined;
  return select(state, activeTabKey.layoutKey);
};

export const useSelectActiveMosaicLayout = (): State | undefined =>
  useMemoSelect(selectActiveMosaicLayout, []);

export const selectHauling = (state: StoreState): Haul.DraggingState =>
  selectSliceState(state).hauling;

export const useSelectHauling = (): Haul.DraggingState =>
  useMemoSelect(selectHauling, []);

export const selectColorContext = (state: StoreState): Color.ContextState => {
  const rawContext = selectSliceState(state).colorContext;
  return Color.contextStateZ.parse(rawContext);
};

export const useSelectColorContext = (): Color.ContextState =>
  useMemoSelect(selectColorContext, []);
