// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LineVis } from "../components/line/types";
import { Vis } from "../types";

import { LayoutStoreState, selectLayouts } from "@/features/layout";

import { VisualizationStoreState } from "./slice";

import { selectRanges, WorkspaceStoreState } from "@/features/workspace";
import { selectByKey, selectByKeys, useMemoSelect } from "@/hooks";

/**
 * Selects a visualization from the store by its key.
 *
 * @param state - The state of the visualization store.
 * @param layoutKey - The key of the visualization to select.
 * @returns The visualization with the given key, or undefined if the visualization
 * does not exist.
 */
export const selectVis = (
  state: VisualizationStoreState & LayoutStoreState,
  layoutKey?: string
): Vis | undefined | null =>
  selectByKey(
    state.visualization.visualizations,
    layoutKey,
    state.layout.mosaic.activeTab
  );

/**
 * Selects a visualization from the store by its key.
 *
 * @param layoutKey - The key of the visualization to select.
 * @returns The visualization with the given key, or undefined if the visualization
 * does not exist.
 */
export const useSelectVis = (layoutKey?: string): Vis | null | undefined =>
  useMemoSelect(
    (state: VisualizationStoreState & LayoutStoreState) => selectVis(state, layoutKey),
    [layoutKey]
  );

/**
 * Selects a sugared visualization from the store by its key. Adds any additional
 * properties to the visualization that are not stored in the core visualization.
 *
 * @param layoutKey - The key of the visualization to select.
 * @returns The visualization with the given key, or undefined if the visualization
 * does not exist.
 */
export const useSelectSVis = <V extends Vis>(layoutKey?: string): V | undefined =>
  useMemoSelect(
    (state: VisualizationStoreState & LayoutStoreState & WorkspaceStoreState) => {
      const vis = selectVis(state, layoutKey);
      if (vis == null) return undefined;
      switch (vis.variant) {
        case "linePlot": {
          const x1Ranges = selectRanges(state, (vis as LineVis).ranges.x1);
          return { ...vis, ranges: { x1: x1Ranges } };
        }
      }
      return undefined;
    },
    [layoutKey]
  ) as V | undefined;

export const selectMultipleVis = (
  state: VisualizationStoreState & LayoutStoreState,
  layoutKeys?: string[]
): Vis[] =>
  selectByKeys(
    state.visualization.visualizations,
    selectLayouts(state, layoutKeys).map((layout) => layout.key)
  );
