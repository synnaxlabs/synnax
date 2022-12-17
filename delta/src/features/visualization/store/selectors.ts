import memoize from "proxy-memoize";
import { useSelector } from "react-redux";

import { LinePlotVisualization, Visualization } from "../types";

import { VisualizationStoreState } from "./slice";

import { LayoutStoreState, useSelectLayoutCore } from "@/features/layout";
import { useSelectRangeFilterCore, WorkspaceStoreState } from "@/features/workspace";

export const useSelectVisualizationCore = (
  state: VisualizationStoreState & LayoutStoreState,
  layoutKey: string
): Visualization | undefined => {
  const layout = useSelectLayoutCore(state, layoutKey);
  if (layout == null) return undefined;
  return state.visualization.visualizations[layout.key];
};

export const useSelectVisualization = (
  layoutKey: string
): Visualization | undefined => {
  return useSelector((state: VisualizationStoreState & LayoutStoreState) =>
    useSelectVisualizationCore(state, layoutKey)
  );
};

export const useSelectSugaredVisualization = (
  layoutKey: string
): Visualization | undefined =>
  useSelector(
    memoize(
      (state: VisualizationStoreState & LayoutStoreState & WorkspaceStoreState) => {
        const vis = useSelectVisualizationCore(state, layoutKey);
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
    )
  );
