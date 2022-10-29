import { memo, useMemo, useRef, useEffect, ComponentType } from "react";
import { Axis, PlotData, LinePlotMetadata, Series } from "./types";
import uPlot from "uplot";
import { Theming, Theme } from "../../theming";
import "uplot/dist/uPlot.min.css";
import "./LinePlotCore.css";

export interface LinePlotCoreProps extends LinePlotMetadata {
  data: PlotData;
}

export type LinePlotCore = ComponentType<LinePlotCoreProps>;

function UPlotLinePlotCore(props: LinePlotCoreProps) {
  const ref = useRef<HTMLDivElement>(null);
  const { theme } = Theming.useContext();

  const [options, alignedData] = useMemo(
    () => buildPlot(props, theme),
    [props, theme]
  );

  let plot: uPlot;
  useEffect(() => {
    const plotContainer = ref.current;
    if (!plotContainer) return;
    plot = new uPlot(options, alignedData, plotContainer);
    return plot.destroy;
  }, [options, alignedData]);

  return <div ref={ref} className="pluto-line-plot__core--uplot"></div>;
}

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
      axes: buildAxes(axes, theme),
      series: buildSeries(series, theme),
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

const alignData = (data: PlotData, series: Series[]): uPlot.AlignedData => {
  if (!data || !series) return [];
  return uPlot.join(series.map(({ x, y }) => [data[x], data[y]]));
};

const buildAxes = (axes: Axis[], theme: Theme): uPlot.Axis[] => {
  if (!axes) return [];
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

const buildSeries = (series: Series[], theme: Theme): uPlot.Series[] => {
  if (!series) return [];
  const s = series.map(({ label, color, axis }, i) => {
    return {
      label,
      stroke: color || theme.colors.visualization.palettes.default[i],
      scale: axis,
    };
  });
  return [{}, ...s];
};

const buildScales = (axes: Axis[]): uPlot.Scales => {
  if (!axes) return {};
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
