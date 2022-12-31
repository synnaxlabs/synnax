// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
