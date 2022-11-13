import { useClusterClient } from "@/features/cluster";
import { LayoutRendererProps } from "@/features/layout";
import { memo } from "react";
import { useDispatch } from "react-redux";
import { setVisualization, useSelectSugaredVisualization } from "../store";
import { SugaredLinePlotVisualization, Visualization } from "../types";
import { LinePlot } from "./LinePlot";

export const VisualizationLayoutRenderer = memo(
	({ layoutKey }: LayoutRendererProps) => {
		const vis = useSelectSugaredVisualization(layoutKey);
		const dispatch = useDispatch();
		const client = useClusterClient();
		if (!vis || !client) {
			return <h1>No Client</h1>;
		}

		const onChange = (vis: Visualization) => {
			dispatch(setVisualization(vis));
		};

		switch (vis.variant) {
			case "linePlot":
				return (
					<LinePlot
						visualization={vis as SugaredLinePlotVisualization}
						client={client}
						onChange={onChange}
						resizeDebounce={100}
					/>
				);
		}
		return <h1>No Visualization Found</h1>;
	}
);
