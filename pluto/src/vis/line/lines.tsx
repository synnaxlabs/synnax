// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ChannelKey } from "@synnaxlabs/client";
import { Box, Deep } from "@synnaxlabs/x";

import { VisContext } from "../context";
import { LineRenderRequest } from "../gl/line";

import { hexToRGBA } from "@/color";
import { Theme } from "@/theming";
import { AxisKey, X_AXIS_KEYS, Y_AXIS_KEYS } from "@/vis/Axis";
import { GLLine } from "@/vis/gl";
import { Axes } from "@/vis/line/axes";
import { Scales } from "@/vis/line/scales";
import { Telem } from "@/vis/line/telem";
import { Viewport } from "@/vis/line/viewport";

export interface LineState {
  axis: AxisKey;
  channel: ChannelKey;
  range: string;
  color: string;
  width: number;
}

export const ZERO_LINES_STATE: LinesState = [];

export type LinesState = LineState[];

export class Lines {
  private state: LinesState;
  private lines: GLLine[];
  private box: Box;

  constructor() {
    this.state = Deep.copy(ZERO_LINES_STATE);
    this.lines = [];
    this.box = Box.ZERO;
  }

  static zeroState(): LinesState {
    return Deep.copy(ZERO_LINES_STATE);
  }

  update(state: LinesState): void {
    this.state = state;
  }

  build(
    viewport: Viewport,
    telem: Telem,
    scales: Scales,
    axes: Axes,
    theme: Theme
  ): void {
    this.lines = [];
    X_AXIS_KEYS.forEach((xKey) => {
      const xResponses = telem.axis(xKey);
      const xScale = scales.offset(xKey);
      if (xResponses.length === 0 || xScale == null) return;
      Y_AXIS_KEYS.forEach((yKey) => {
        const yResponses = telem.axis(yKey);
        const yScale = scales.offset(yKey);
        if (yResponses.length === 0 || yScale == null) return;
        const scale = { x: xScale.dim(1), y: yScale.dim(1) };
        const offset = { x: xScale.pos(0), y: yScale.pos(0) };
        const pallete = theme.colors.visualization.palettes.default;
        yResponses.forEach((yRes, i) => {
          const xRes = xResponses[0];
          const color = pallete[i % pallete.length];
          yRes.data.forEach((yArr, j) => {
            this.lines.push({
              color: hexToRGBA(color, 1, 255),
              scale,
              offset,
              y: yArr.gl,
              x: xRes.data[j].gl,
              strokeWidth: 3,
              length: yArr.length,
            });
          });
        });
      });
    });
    this.box = new Box(
      { x: viewport.box.x + axes.offsets.left, y: viewport.box.y + axes.offsets.top },
      {
        width: viewport.box.width - axes.offsets.left - axes.offsets.right,
        height: viewport.box.height - axes.offsets.top - axes.offsets.bottom,
      }
    );
  }

  render(ctx: VisContext): void {
    const base = ctx.gl.registry.get<LineRenderRequest>("line");
    const scissor = ctx.gl.scissor(base);
    scissor.render(ctx.gl, { box: this.box, lines: this.lines });
  }

  cleanup(ctx: VisContext): void {
    const scissor = ctx.gl.scissor(ctx.gl.registry.get<LineRenderRequest>("line"));
    scissor.clear(ctx.gl, this.box);
  }
}
