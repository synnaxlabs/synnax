import { useMemo } from "react";

import { GLLine, hexToRGBA, Theme } from "@synnaxlabs/pluto";
import { Box } from "@synnaxlabs/x";

import { Axes } from "@/vis/line/axes";
import { Data } from "@/vis/line/data";
import { Scales } from "@/vis/line/scales";
import { X_AXIS_KEYS, Y_AXIS_KEYS } from "@/vis/types";

export class Lines {
  readonly box: Box;
  readonly lines: GLLine[];

  constructor(box: Box, lines: GLLine[]) {
    this.box = box;
    this.lines = lines;
  }

  static use(
    container: Box,
    data: Data,
    scales: Scales,
    axes: Axes,
    theme: Theme | null | undefined
  ): Lines {
    return useMemo(() => {
      if (theme == null) return new Lines(container, []);
      const lines: GLLine[] = [];
      X_AXIS_KEYS.forEach((xKey) => {
        const xData = data.axis(xKey);
        const xScale = scales.offset(xKey);
        if (xData.length === 0 || xScale == null) return;
        Y_AXIS_KEYS.forEach((yKey) => {
          const yData = data.axis(yKey);
          const yScale = scales.offset(yKey);
          if (yData.length === 0 || yScale == null) return;
          const scale = { x: xScale.dim(1), y: yScale.dim(1) };
          const offset = { x: xScale.pos(0), y: yScale.pos(0) };
          yData.forEach((yRes, i) => {
            const xRes = xData[0];
            yRes.arrays.forEach((yArr, j) => {
              lines.push({
                color: hexToRGBA(
                  theme.colors.visualization.palettes.default[i],
                  1,
                  255
                ),
                scale,
                offset,
                y: yRes.buffers.value[j].buf,
                x: xRes.buffers.value[j].buf,
                strokeWidth: 3,
                length: yArr.length,
              });
            });
          });
        });
      });
      return new Lines(container, lines);
    }, [container, data, scales, axes, theme]);
  }

  get gl(): GLLine[] {
    return this.lines;
  }

  valid(): boolean {
    return this.lines.length > 0;
  }
}
