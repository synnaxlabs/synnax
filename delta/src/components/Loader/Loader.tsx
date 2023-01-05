// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useRef } from "react";

import { Theming } from "@synnaxlabs/pluto";
import * as d3 from "d3";

type point = [number, number];

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

export const Loader = (): JSX.Element => {
  const ref = useRef<SVGSVGElement>(null);
  const strokeWidth = 3;
  const { theme } = Theming.useContext();

  useEffect(() => {
    if (ref.current == null) return;
    const svgEl = d3.select(ref.current);
    svgEl.selectAll("*").remove();
    svgEl.attr("viewBox", "0 0 100 100");
    const path = svgEl.append("path");
    const edgeLength = 45;
    path
      .attr("id", "path-1")
      .attr("stroke", theme.colors.primary.z)
      .attr("fill", "none")
      .attr("stroke-width", strokeWidth)
      .attr("stroke-linecap", "round")
      .attr("d", curveFunc(calculatePoints({ edgeLength, center: [50, 50] })));

    const totalLength = path?.node()?.getTotalLength() ?? 0;

    const animate = (): void => {
      void path
        .attr("stroke-dasharray", "16 12")
        .attr("stroke-dashoffset", totalLength)
        .transition()
        .duration(6000)
        .ease(d3.easeLinear)
        .attr("stroke-dashoffset", 0)
        .on("end", () => animate());
    };
    animate();
  }, []);

  return <svg ref={ref} style={{ height: "15%", width: "15%" }} />;
};
