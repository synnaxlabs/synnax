// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SVGProps, useEffect, useRef, useState } from "react";

import * as d3 from "d3";

import { TypographyLevel } from "../../atoms";
import { Theming } from "../../theming";

export interface Metric {
  name: string;
  value: number;
  max: number;
  units: string;
}

export interface Title {
  text: string;
  textLevel: TypographyLevel;
}

export interface HexagonBarProps extends SVGProps<unknown> {
  title?: Title;
  strokeWidth: number;
  metrics: Metric[];
}

type point = [number, number];

const calculateDashArray = ({
  edgeLength,
  metric,
}: {
  edgeLength: number;
  metric: Metric;
}): string => {
  const proportion = metric.value / metric.max;
  // 6 is the number of sides in a hexagon!
  return `${edgeLength * proportion * 6} ${edgeLength * (1 - proportion) * 6}`;
};

const HEX_COS = Math.abs(Math.cos((2 * Math.PI) / 3));
const HEX_SIN = Math.abs(Math.sin((2 * Math.PI) / 3));

const calculatePoints = ({
  edgeLength,
  center = [0, 0],
}: {
  edgeLength: number;
  center: point;
}): point[] => {
  const cw = HEX_SIN * edgeLength;
  const sw = HEX_COS * edgeLength;
  return [
    [0, edgeLength / 2 + sw],
    [-cw, edgeLength / 2],
    [-cw, -edgeLength / 2],
    [0, -(edgeLength / 2 + sw)],
    [cw, -edgeLength / 2],
    [cw, edgeLength / 2],
    [0, edgeLength / 2 + sw],
  ].map(([x, y]) => [x + center[0], y + center[1]]);
};

const curveFunc = d3
  .line()
  .curve(d3.curveLinear)
  .x((d) => d[0])
  .y((d) => d[1]);

export const HexagonBar = ({
  strokeWidth = 5,
  metrics = [],
  ...props
}: HexagonBarProps): JSX.Element => {
  const ref = useRef<SVGSVGElement>(null);
  const [numPaths, setNumPaths] = useState<number>(0);
  const { theme } = Theming.useContext();

  useEffect(() => {
    if (ref.current == null) return;
    const svgEl = d3.select(ref.current);
    svgEl.selectAll("*").remove();
    svgEl.attr("viewBox", "0 0 100 100");
  }, []);

  useEffect(() => {
    if (ref.current == null) return;
    const svgEl = d3.select(ref.current);
    metrics.forEach((metric, i) => {
      const pathID = `path-${i}`;
      const path: d3.Selection<SVGPathElement, unknown, null, undefined> =
        i >= numPaths ? svgEl.append("path") : svgEl.select(`#${pathID}`);
      const edgeLength = 25 + i * strokeWidth * 1.25;
      path
        .attr("id", pathID)
        .attr("stroke", theme.colors.visualization.palettes.default[i])
        .attr("fill", "none")
        .attr("stroke-width", strokeWidth)
        .attr("d", curveFunc(calculatePoints({ edgeLength, center: [50, 50] })))
        .transition()
        .duration(1000)
        .ease(d3.easeLinear)
        .attr(`stroke-dasharray`, calculateDashArray({ edgeLength, metric }));
    });
    svgEl
      .selectAll(`path`)
      .filter((d, i) => i > metrics.length - 1)
      .remove();
    setNumPaths(metrics.length);
  }, [metrics, strokeWidth]);

  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-expect-error
  return <svg ref={ref} {...props} />;
};
