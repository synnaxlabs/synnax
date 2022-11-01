import { useSelectLayout } from "@/features/layout";
import memoize from "proxy-memoize";
import { useCallback } from "react";
import { useSelector } from "react-redux";
import { Visualization } from "../types";
import { VisualizationStoreState } from "./slice";

export const useSelectVisualization = (
  layoutKey: string
): Visualization | undefined => {
  const layout = useSelectLayout(layoutKey);
  return useSelector(
    useCallback(
      memoize(
        (state: VisualizationStoreState) =>
          state.visualization.visualizations[layoutKey]
      ),
      [layout]
    )
  );
};
