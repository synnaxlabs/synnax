import { LinePlotMetadata } from "@synnaxlabs/pluto";

type VisualizationVariant = "linePlot";

export interface Visualization {
  variant: string;
  layoutKey: string;
}

export interface LinePlotVisualization extends Visualization, LinePlotMetadata {
  channels: string[];
}
