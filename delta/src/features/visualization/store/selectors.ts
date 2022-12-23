import { LinePlotVisualization, Visualization } from "../types";

import { VisualizationStoreState } from "./slice";

import { LayoutStoreState, selectLayout } from "@/features/layout";
import { useSelectRangeFilterCore, WorkspaceStoreState } from "@/features/workspace";
import { useMemoSelect } from "@/hooks";

export const selectVisualization = (
  state: VisualizationStoreState & LayoutStoreState,
  layoutKey: string
): Visualization | undefined => {
  const layout = selectLayout(state, layoutKey);
  if (layout == null) return undefined;
  return state.visualization.visualizations[layout.key];
};

export const useSelectVisualization = (layoutKey: string): Visualization | undefined =>
  useMemoSelect(
    (state: VisualizationStoreState & LayoutStoreState) =>
      selectVisualization(state, layoutKey),
    [layoutKey]
  );

export const useSelectSugaredVisualization = (
  layoutKey: string
): Visualization | undefined =>
  useMemoSelect(
    (state: VisualizationStoreState & LayoutStoreState & WorkspaceStoreState) => {
      const vis = selectVisualization(state, layoutKey);
      if (vis == null) return undefined;
      switch (vis.variant) {
        case "linePlot": {
          const ranges = useSelectRangeFilterCore(
            state,
            (vis as LinePlotVisualization).ranges
          );
          return {
            ...vis,
            ranges,
          };
        }
      }
      return undefined;
    }
  );
