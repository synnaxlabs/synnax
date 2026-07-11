// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Drift, selectWindow, selectWindowKey } from "@synnaxlabs/drift";
import { Color, type Haul } from "@synnaxlabs/pluto";
import { type direction } from "@synnaxlabs/x";
import { useMemo, useSyncExternalStore } from "react";

import type * as Mosaic from "@/session/layout/mosaic";
import {
  SLICE_NAME,
  type SliceState,
  type State,
  type StoreState,
} from "@/session/layout/slice";
import { Modals } from "@/session/modals";
import { Select } from "@/session/select";

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const useSelectSliceState = (): SliceState =>
  Select.useMemo((state: StoreState) => selectSliceState(state), []);

export const select = (state: StoreState, key: string): State | undefined =>
  selectSliceState(state).layouts[key];

export const selectRequired = (state: StoreState, key: string): State => {
  const layout = select(state, key);
  if (layout == null) throw new Error(`Layout ${key} not found`);
  return layout;
};

export const selectType = (state: StoreState, key: string): string =>
  selectRequired(state, key).type;

export const useSelectType = (key: string): string =>
  Select.useMemo((state: StoreState) => selectType(state, key), [key]);

export const selectArgs = <A>(state: StoreState, key: string): A => {
  const layout = select(state, key);
  return layout?.args as A;
};

export const useSelectArgs = <A>(key: string): A =>
  Select.useMemo((state: StoreState) => selectArgs(state, key), [key]);

export const selectAltKey = (state: StoreState, key: string): string | undefined =>
  selectSliceState(state).keyToAltKey[key];

export const useSelectAltKey = (key: string): string | undefined =>
  Select.useMemo((state: StoreState) => selectAltKey(state, key), [key]);

const selectName = (state: StoreState, key: string): string | undefined =>
  select(state, key)?.name;

const selectRequiredName = (state: StoreState, key: string): string =>
  selectRequired(state, key).name;

export const useSelectName = (key: string): string | undefined =>
  Select.useMemo((state: StoreState) => selectName(state, key), [key]);

export const useSelectRequiredName = (key: string): string =>
  Select.useMemo((state: StoreState) => selectRequiredName(state, key), [key]);

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
  Select.useMemo((state: StoreState) => select(state, key), [key]);

export const useSelectRequired = (key: string): State =>
  Select.useMemo((state: StoreState) => selectRequired(state, key), [key]);

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
  Select.useMemo(selectFocused, []);

export const useSelectMosaic = (): [string, Mosaic.Node] | [null, null] =>
  Select.useMemo(selectMosaic, []);

/**
 * The data a single mosaic node needs to render, discriminated into a leaf (tab
 * strip) or a split (two child panes). Deriving one node at a time lets each node
 * component be its own re-render boundary.
 */
export type MosaicNodeData =
  | { variant: "leaf"; selected: string | null; tabs: Mosaic.Tab[] }
  | {
      variant: "split";
      direction: direction.Direction;
      size?: number;
      firstKey: number;
      lastKey: number;
    };

// descendToKey navigates from the root to the node at the given heap key (root = 1,
// children of k are 2k and 2k+1), reading only the nodes along that path. Unlike a
// depth-first search it never touches sibling subtrees, so a proxy-memoized selector
// built on it stays cached when an unrelated node changes.
const descendToKey = (root: Mosaic.Node, key: number): Mosaic.Node | undefined => {
  if (key <= 1) return root;
  const bits: number[] = [];
  for (let k = key; k > 1; k = Math.floor(k / 2)) bits.push(k % 2);
  let node: Mosaic.Node | undefined = root;
  for (let i = bits.length - 1; i >= 0 && node != null; i--)
    node = bits[i] === 0 ? node.first : node.last;
  return node;
};

export const selectMosaicNode = (
  state: StoreState & Drift.StoreState,
  key: number,
  windowKey?: string,
): MosaicNodeData | null => {
  const winKey = selectWindowKey(state, windowKey);
  if (winKey == null) return null;
  const mosaic = selectSliceState(state).mosaics[winKey];
  if (mosaic == null) return null;
  const node = descendToKey(mosaic.root, key);
  if (node == null) return null;
  if (node.tabs != null)
    return { variant: "leaf", selected: node.selected ?? null, tabs: node.tabs };
  if (node.first == null || node.last == null) return null;
  return {
    variant: "split",
    direction: node.direction ?? "x",
    size: node.size,
    firstKey: node.first.key,
    lastKey: node.last.key,
  };
};

/** Selects the render data for a single node of the active window's mosaic. */
export const useSelectMosaicNode = (key: number): MosaicNodeData | null =>
  Select.useMemo(
    (state: StoreState & Drift.StoreState) => selectMosaicNode(state, key),
    [key],
  );

/**
 * Reports whether the given tab is the window's active mosaic tab. Returns a
 * boolean so consumers re-render only when their own active state flips, not on
 * every active-tab change.
 */
export const useSelectIsActiveMosaicTab = (tabKey: string | null): boolean =>
  Select.useMemo(
    (state: StoreState & Drift.StoreState) =>
      tabKey != null && selectActiveMosaicTabState(state).layoutKey === tabKey,
    [tabKey],
  );

export const selectMany = (state: StoreState, keys?: string[]): State[] =>
  Select.byKeys<string, State>(selectSliceState(state).layouts, keys);

export const useSelectMany = (keys?: string[]): State[] =>
  Select.useMemo((state: StoreState) => selectMany(state, keys), [keys]);

export interface SelectActiveMosaicTabState {
  layoutKey: string | null;
  blurred: boolean;
}

export const selectActiveMosaicTabState = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): Pick<SelectActiveMosaicTabState, "layoutKey"> => {
  const winKey = selectWindowKey(state, windowKey);
  if (winKey == null) return { layoutKey: null };
  const sliceState = selectSliceState(state);
  return { layoutKey: sliceState.mosaics[winKey].activeTab };
};

export const selectActiveMosaicTabKeyAndNotBlurred = (
  state: StoreState & Drift.StoreState,
  modals: Modals.Store,
  windowKey?: string,
): string | null => {
  if (modals.isAnyOpen()) return null;
  return selectActiveMosaicTabState(state, windowKey).layoutKey;
};

const useIsAnyModalOpen = (): boolean => {
  const store = Modals.useStore("useIsAnyModalOpen");
  return useSyncExternalStore(store.subscribe, store.isAnyOpen);
};

export const useSelectActiveMosaicTabKeyAndNotBlurred = (): string | null => {
  const blurred = useIsAnyModalOpen();
  const layoutKey = Select.useMemo(
    (state: StoreState & Drift.StoreState) =>
      selectActiveMosaicTabState(state).layoutKey,
    [],
  );
  return blurred ? null : layoutKey;
};

export const selectActiveMosaicTabName = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): string | null => {
  const active = selectActiveMosaicTabState(state, windowKey);
  if (active.layoutKey == null) return null;
  return select(state, active.layoutKey)?.name ?? null;
};

export const useSelectActiveMosaicTabState = (): SelectActiveMosaicTabState => {
  const blurred = useIsAnyModalOpen();
  const layoutKey = Select.useMemo(
    (state: StoreState & Drift.StoreState) =>
      selectActiveMosaicTabState(state).layoutKey,
    [],
  );
  return useMemo(() => ({ layoutKey, blurred }), [layoutKey, blurred]);
};

export const useSelectActiveMosaicTabName = (): string | null =>
  Select.useMemo(selectActiveMosaicTabName, []);

export const selectActiveMosaicLayout = (
  state: StoreState & Drift.StoreState,
  windowKey?: string,
): State | undefined => {
  const activeTabKey = selectActiveMosaicTabState(state, windowKey);
  if (activeTabKey.layoutKey == null) return undefined;
  return select(state, activeTabKey.layoutKey);
};

export const useSelectActiveMosaicLayout = (): State | undefined =>
  Select.useMemo(selectActiveMosaicLayout, []);

export const selectHauling = (state: StoreState): Haul.DraggingState =>
  selectSliceState(state).hauling;

export const useSelectHauling = (): Haul.DraggingState =>
  Select.useMemo(selectHauling, []);

export const selectColorContext = (state: StoreState): Color.ContextState => {
  const rawContext = selectSliceState(state).colorContext;
  return Color.contextStateZ.parse(rawContext);
};

export const useSelectColorContext = (): Color.ContextState =>
  Select.useMemo(selectColorContext, []);
