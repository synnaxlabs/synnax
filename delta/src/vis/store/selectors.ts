// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { selectByKey, selectByKeys, useMemoSelect } from "@/hooks";
import { LayoutStoreState, selectLayouts } from "@/layout";
import { VisMeta } from "@/vis/core";
import { VisStoreState } from "@/vis/store/slice";

/**
 * Selects a visualization from the store by its key.
 *
 * @param state - The state of the visualization store.
 * @param layoutKey - The key of the visualization to select.
 * @returns The visualization with the given key, or undefined if the visualization
 * does not exist.
 */
export const selectVis = <V extends VisMeta>(
  state: VisStoreState & LayoutStoreState,
  layoutKey?: string,
  variant?: V["variant"]
): V | undefined | null => {
  const v = selectByKey<string, V>(
    state.visualization.visualizations as Record<string, V>,
    layoutKey,
    state.layout.mosaic.activeTab
  );
  if (v == null) return null;
  if (variant != null && v.variant !== variant)
    throw new Error(`[vis] - expected variant ${variant} but got ${v.variant}`);
  return v;
};

export const selectRequiredVis = <V extends VisMeta>(
  state: VisStoreState & LayoutStoreState,
  layoutKey?: string,
  variant?: V["variant"]
): V => {
  const v = selectVis<V>(state, layoutKey, variant);
  if (v == null)
    throw new Error(
      `[vis] - ${variant ?? ""}: ${layoutKey ?? ""} required but not found`
    );
  return v;
};

/**
 * Selects a visualization from the store by its key.
 *
 * @param layoutKey - The key of the visualization to select.
 * @returns The visualization with the given key, or undefined if the visualization
 * does not exist.
 */
export const useSelectVis = <V extends VisMeta = VisMeta>(
  layoutKey?: string,
  variant?: V["variant"]
): V | null | undefined =>
  useMemoSelect(
    (state: VisStoreState & LayoutStoreState) =>
      selectVis<V>(state, layoutKey, variant),
    [layoutKey]
  );

export const useSelectVisMeta = (layoutKey?: string): VisMeta | null | undefined =>
  useMemoSelect(
    (state: VisStoreState & LayoutStoreState) => {
      const vis = selectVis(state, layoutKey);
      if (vis == null) return null;
      return {
        key: vis.key,
        variant: vis.variant,
      };
    },
    [layoutKey]
  );

export const useSelectRequiredVisMeta = (layoutKey: string): VisMeta =>
  useMemoSelect(
    (state: VisStoreState & LayoutStoreState) => {
      const vis = selectRequiredVis(state, layoutKey);
      return {
        key: vis.key,
        variant: vis.variant,
      };
    },
    [layoutKey]
  );

export const selectMultipleVis = (
  state: VisStoreState & LayoutStoreState,
  layoutKeys?: string[]
): VisMeta[] =>
  selectByKeys(
    state.visualization.visualizations,
    selectLayouts(state, layoutKeys).map((layout) => layout.key)
  );
