import { LinePlotMetadata } from "@synnaxlabs/pluto";
import { Range } from "@/features/workspace";

type VisualizationVariant = "linePlot";

export interface Visualization {
	variant: string;
	layoutKey: string;
}

export interface LinePlotVisualization extends Visualization, LinePlotMetadata {
	channels: string[];
	ranges: string[];
}

export interface SugaredLinePlotVisualization
	extends Omit<LinePlotVisualization, "ranges"> {
	ranges: Range[];
}
