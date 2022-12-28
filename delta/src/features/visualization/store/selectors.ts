import { LinePlotVisualization, Visualization } from "../types";

import { VisualizationStoreState } from "./slice";

import { LayoutStoreState, LAYOUT_SLICE_NAME, selectLayouts } from "@/features/layout";
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
export const selectVisualization = (
  state: VisualizationStoreState & LayoutStoreState,
  layoutKey?: string
): Visualization | undefined | null =>
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
export const useSelectVisualization = (
  layoutKey?: string
): Visualization | null | undefined =>
  useMemoSelect(
    (state: VisualizationStoreState & LayoutStoreState) =>
      selectVisualization(state, layoutKey),
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
export const useSelectSugaredVisualization = <V extends Visualization>(
  layoutKey?: string
): V | undefined =>
  useMemoSelect(
    (state: VisualizationStoreState & LayoutStoreState & WorkspaceStoreState) => {
      const vis = selectVisualization(state, layoutKey);
      if (vis == null) return undefined;
      switch (vis.variant) {
        case "linePlot": {
          const x1Ranges = selectRanges(
            state,
            (vis as LinePlotVisualization).ranges.x1
          );
          const x2Ranges = selectRanges(
            state,
            (vis as LinePlotVisualization).ranges.x2
          );
          return { ...vis, ranges: { x1: x1Ranges, x2: x2Ranges } };
        }
      }
      return undefined;
    },
    [layoutKey]
  ) as V | undefined;

export const selectVisualizations = (
  state: VisualizationStoreState & LayoutStoreState,
  layoutKeys?: string[]
): Visualization[] =>
  selectByKeys(
    state.visualization.visualizations,
    selectLayouts(state, layoutKeys).map((layout) => layout.key)
  );

export const selectWarpMode = (state: VisualizationStoreState): boolean =>
  state.visualization.warpMode;

export const useSelectWarpMode = (): boolean => useMemoSelect(selectWarpMode, []);
