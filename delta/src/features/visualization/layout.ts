import { Layout, LayoutCreator, LayoutCreatorProps } from "@/features/layout";
import { Visualization } from "./types";
import { nanoid } from "nanoid";
import { Optional } from "@/util/types";
import { setVisualization as storeCreateVizualization } from "./store";

export const createVisualization =
	<V extends Visualization>(initial: Partial<V>): LayoutCreator =>
	({ dispatch }: LayoutCreatorProps): Layout => {
		const key = initial.layoutKey ?? nanoid();
		dispatch(
			storeCreateVizualization({
				...initial,
				layoutKey: key,
				variant: "linePlot",
			})
		);
		return {
			key,
			location: "mosaic",
			type: "visualization",
			title: initial.layoutKey || "Plot",
		};
	};
