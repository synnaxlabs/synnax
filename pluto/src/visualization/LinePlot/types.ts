export interface Series {
  label: string;
  x: string;
  y: string;
  color?: string;
  axis?: string;
}

type AxisLocation = "top" | "bottom" | "left" | "right";

export interface Axis {
  key: string;
  location?: AxisLocation;
  range?: [number, number];
  label: string;
}

export type Array = uPlot.TypedArray | number[];

export type PlotData = Record<string, any[]>;

export interface LinePlotMeta {
  width: number;
  height: number;
  series: Series[];
  axes: Axis[];
}
