// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentType, memo, useEffect, useMemo, useRef } from "react";

import uPlot from "uplot";
import "uplot/dist/uPlot.min.css";

import { Theme, Theming } from "../../theming";

import { Axis, LinePlotMeta, PlotData, Series } from "./types";

import "./LinePlotCore.css";

export interface LinePlotCoreProps extends LinePlotMeta {
  data: PlotData;
}

export type LinePlotCoreComponent = ComponentType<LinePlotCoreProps>;

const UPlotLinePlotCore = (props: LinePlotCoreProps): JSX.Element => {
  const ref = useRef<HTMLDivElement>(null);
  const { theme } = Theming.useContext();

  const [options, alignedData] = useMemo(() => buildPlot(props, theme), [props, theme]);

  let plot: uPlot;
  useEffect(() => {
    const plotContainer = ref.current;
    if (plotContainer == null) return;
    // eslint-disable-next-line new-cap
    plot = new uPlot(options, alignedData, plotContainer);
    return plot.destroy;
  }, [options, alignedData]);

  return <div ref={ref} className="pluto-line-plot__core--uplot"></div>;
};

const buildPlot = (
  { series, data, width, height, axes }: LinePlotCoreProps,
  theme: Theme
): [uPlot.Options, uPlot.AlignedData] => {
  const alignedData = alignData(data, series);
  return [
    {
      width,
      height,
      padding: [theme.sizes.base, 0, 0, 0],
      axes: buildAxes(theme, axes),
      series: buildSeries(theme, series),
      scales: buildScales(axes),
      legend: {
        show: false,
      },
      cursor: {
        drag: {
          x: true,
          y: true,
          uni: 50,
        },
      },
    },
    alignedData,
  ];
};

const locationSides = {
  right: 1,
  left: 3,
  bottom: 2,
  top: 0,
};

const alignData = (data?: PlotData, series?: Series[]): uPlot.AlignedData => {
  if (data == null || series == null) return [];
  return uPlot.join(
    series
      .filter(({ x, y }) => data[x] != null && data[y] != null)
      .map(({ x, y }) => [data[x], data[y]])
  );
};

const buildAxes = (theme: Theme, axes?: Axis[]): uPlot.Axis[] => {
  if (axes == null || axes.length === 0) return [];
  return axes.map(({ key, label, location = "bottom" }) => {
    return {
      label,
      grid: {
        stroke: theme.colors.gray.m2,
        width: theme.sizes.border.width as number,
      },
      stroke: theme.colors.text,
      side: locationSides[location],
      size: theme.sizes.base * 4,
      labelGap: theme.sizes.base * 2,
      scale: key,
    };
  });
};

const buildSeries = (theme: Theme, series?: Series[]): uPlot.Series[] => {
  if (series == null || series.length === 0) return [];
  const s = series.map(({ label, color, axis }, i) => {
    return {
      label,
      stroke: color ?? theme.colors.visualization.palettes.default[i],
      scale: axis,
    };
  });
  return [{}, ...s];
};

const buildScales = (axes?: Axis[]): uPlot.Scales => {
  if (axes == null || axes.length === 0) return {};
  const s = Object.fromEntries(
    axes.map(({ key, range }): [string, uPlot.Scale] => {
      return [
        key,
        {
          time: false,
          range,
          auto: range === undefined,
        },
      ];
    })
  );
  return s;
};

export const LinePlotCore = {
  UPlot: memo(UPlotLinePlotCore),
};
