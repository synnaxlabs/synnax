import {
	LayoutStoreState,
	useSelectLayout,
	useSelectLayoutCore,
} from "@/features/layout";
import { useSelectRangeFilterCore, WorkspaceStoreState } from "@/features/workspace";
import memoize from "proxy-memoize";
import { useCallback } from "react";
import { useSelector } from "react-redux";
import {
	LinePlotVisualization,
	SugaredLinePlotVisualization,
	Visualization,
} from "../types";
import { VisualizationStoreState } from "./slice";

export const useSelectVisualizationCore = (
	state: VisualizationStoreState & LayoutStoreState,
	layoutKey: string
) => {
	const layout = useSelectLayoutCore(state, layoutKey);
	if (!layout) return undefined;
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
				if (!vis) return undefined;
				switch (vis.variant) {
					case "linePlot":
						const ranges = useSelectRangeFilterCore(
							state,
							(vis as LinePlotVisualization).ranges
						);
						return {
							...vis,
							ranges,
						};
				}
				return undefined;
			}
		)
	);
