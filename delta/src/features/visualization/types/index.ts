import type { LinePlotMetadata } from "@synnaxlabs/pluto";

import { Range } from "@/features/workspace";

export interface Visualization {
  variant: string;
  key: string;
}

export interface LinePlotVisualization extends Visualization, LinePlotMetadata {
  channels: {
    y1: readonly string[];
    y2: readonly string[];
    y3: readonly string[];
    y4: readonly string[];
    x1: string;
    x2: string;
  };
  ranges: {
    x1: readonly string[];
    x2: readonly string[];
  };
}

export interface SugaredLinePlotVisualization
  extends Omit<LinePlotVisualization, "ranges"> {
  ranges: {
    x1: Range[];
    x2: Range[];
  };
}
