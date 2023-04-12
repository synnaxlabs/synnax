// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemo } from "react";

import { GLLine, hexToRGBA, Theme } from "@synnaxlabs/pluto";
import { Box } from "@synnaxlabs/x";

import { X_AXIS_KEYS, Y_AXIS_KEYS } from "@/vis/axis";
import { Axes } from "@/vis/line/axes";
import { Data } from "@/vis/line/data";
import { Scales } from "@/vis/line/scales";

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
          const pallete = theme.colors.visualization.palettes.default;
          yData.forEach((yRes, i) => {
            const xRes = xData[0];
            const color = pallete[i % pallete.length];
            yRes.arrays.forEach((yArr, j) => {
              lines.push({
                color: hexToRGBA(color, 1, 255),
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
      return new Lines(
        new Box(
          { x: container.x + axes.offsets.left, y: container.y + axes.offsets.top },
          {
            width: container.width - axes.offsets.left - axes.offsets.right,
            height: container.height - axes.offsets.top - axes.offsets.bottom,
          }
        ),
        lines
      );
    }, [container, data, scales, axes, theme]);
  }

  get gl(): GLLine[] {
    return this.lines;
  }

  valid(): boolean {
    return this.lines.length > 0;
  }
}
