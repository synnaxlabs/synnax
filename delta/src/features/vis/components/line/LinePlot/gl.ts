import { GLLine, hexToRGBA, Theme } from "@synnaxlabs/pluto";
import { Box, BoxScale, ZERO_BOX } from "@synnaxlabs/x";

import { AxesState } from "./axes";
import { LineVisData } from "./data";
import { ScalesState } from "./scale";

import { X_AXIS_KEYS, Y_AXIS_KEYS } from "@/features/vis/types";

export interface GLState {
  lines: GLLine[];
  box: Box;
}

const ZERO_GL_STATE: GLState = {
  lines: [],
  box: ZERO_BOX,
};

const buildGL = (
  container: Box,
  data: LineVisData,
  scales: ScalesState,
  axes: AxesState,
  theme: Theme
): GLState => {
  const lines: GLLine[] = [];
  X_AXIS_KEYS.forEach((xKey) => {
    const xData = data[xKey];
    const xScale = scales.offset.forward[xKey];
    if (xData.length === 0 || xScale == null) return;
    Y_AXIS_KEYS.forEach((yKey) => {
      const yData = data[yKey];
      const yScale = scales.offset.forward[yKey];
      if (yData.length === 0 || yScale == null) return;
      const scale = { x: xScale.dim(1), y: yScale.dim(1) };
      const offset = { x: xScale.pos(0), y: yScale.pos(0) };
      yData.forEach((yRes, i) => {
        const xRes = xData[0];
        yRes.arrays.forEach((yArr, j) => {
          lines.push({
            color: hexToRGBA(theme.colors.visualization.palettes.default[i], 1, 255),
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
  return { lines, box: BoxScale.translate(container.topLeft).box(axes.innerBox) };
};

export const GL = {
  initial: () => ({ ...ZERO_GL_STATE }),
  build: buildGL,
};
