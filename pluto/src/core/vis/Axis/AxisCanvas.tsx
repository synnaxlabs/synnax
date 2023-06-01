// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { textDimensions } from "@/core/std/Typography/textDimensions";
import {
  AxisContext,
  axisState,
  AxisState,
  ParsedAxisState,
} from "@/core/vis/Axis/core";
import { TickFactory, newTickFactory } from "@/core/vis/Axis/TickFactory";
import { RenderContext } from "@/core/vis/render";

const TICK_LINE_SIZE = 4;

export class AxisCanvas {
  ctx: RenderContext;
  state: ParsedAxisState;
  tickFactory: TickFactory;

  constructor(ctx: RenderContext, props: ParsedAxisState) {
    this.ctx = ctx;
    this.state = props;
    this.tickFactory = newTickFactory(this.state);
  }

  setState(state: AxisState): void {
    this.state = axisState.parse(state);
    this.tickFactory = newTickFactory(state);
  }

  render(ctx: AxisContext): void {
    const { canvas } = this.ctx;
    canvas.font = this.state.font;
    canvas.fillStyle = this.state.color.hex;
    canvas.strokeStyle = this.state.color.hex;

    switch (this.state.location) {
      case "left":
        this.drawLeft(ctx);
        break;
      case "right":
        this.drawRight(ctx);
        break;
      case "top":
        this.drawTop(ctx);
        break;
      case "bottom":
        this.drawBottom(ctx);
        break;
    }
  }

  drawBottom(ctx: AxisContext): void {
    const { canvas } = this.ctx;
    const { plottingRegion } = ctx;
    const { position: p, showGrid } = this.state;
    const size = plottingRegion.width;
    const gridSize = plottingRegion.height;

    canvas.beginPath();
    canvas.moveTo(p.x, p.y);
    canvas.lineTo(p.x + size, p.y);
    canvas.stroke();

    const ticks = this.tickFactory.generate({ ...ctx, size });

    console.log(canvas.strokeStyle, this.state.color.hex);

    ticks.forEach((tick) => {
      const { width, height } = textDimensions(
        tick.label,
        this.state.font,
        this.ctx.canvas
      );
      canvas.moveTo(p.x + tick.position, p.y);
      canvas.lineTo(p.x + tick.position, p.y + TICK_LINE_SIZE);
      canvas.stroke();
      canvas.fillText(tick.label, p.x + tick.position - width / 2, p.y + 5 + height);
    });
    if (showGrid) {
      canvas.strokeStyle = this.state.gridColor.hex;
      ticks.forEach((tick) => {
        canvas.beginPath();
        canvas.moveTo(p.x + tick.position, p.y);
        canvas.lineTo(p.x + tick.position, p.y - gridSize);
        canvas.stroke();
      });
    }
  }

  drawTop(ctx: AxisContext): void {
    const { canvas } = this.ctx;
    const { plottingRegion } = ctx;
    const { position: _p, showGrid } = this.state;
    const size = plottingRegion.width;
    const gridSize = plottingRegion.height;
    const p = { x: _p.x, y: _p.y + 20 };

    canvas.beginPath();

    canvas.moveTo(p.x, p.y);
    canvas.lineTo(p.x + size, p.y);
    canvas.stroke();

    const ticks = this.tickFactory.generate({ ...ctx, size });

    ticks.forEach((tick) => {
      const { width, height } = textDimensions(
        tick.label,
        this.state.font,
        this.ctx.canvas
      );
      canvas.moveTo(p.x + tick.position, p.y);
      canvas.lineTo(p.x + tick.position, p.y - TICK_LINE_SIZE);
      canvas.stroke();

      canvas.fillText(tick.label, p.x + tick.position - width / 2, p.y - 5 - height);
    });

    if (showGrid) {
      canvas.strokeStyle = this.state.gridColor.hex;
      ticks
        .filter((tick) => tick.position !== 0 && tick.position !== size)
        .forEach((tick) => {
          canvas.beginPath();
          canvas.moveTo(p.x + tick.position, p.y);
          canvas.lineTo(p.x + tick.position, p.y + gridSize);
          canvas.stroke();
        });
    }
  }

  drawLeft(ctx: AxisContext): void {
    const { canvas } = this.ctx;
    const { plottingRegion } = ctx;
    const { position: _p, showGrid } = this.state;
    const size = plottingRegion.height;
    const gridSize = plottingRegion.width;
    const p = { x: _p.x + 40, y: _p.y };

    canvas.beginPath();
    canvas.moveTo(p.x, p.y);
    canvas.lineTo(p.x, p.y + size);
    canvas.stroke();

    const ticks = this.tickFactory.generate({ ...ctx, size });

    ticks.forEach((tick) => {
      const { height, width } = textDimensions(
        tick.label,
        this.state.font,
        this.ctx.canvas
      );
      canvas.moveTo(p.x, p.y + tick.position);
      canvas.lineTo(p.x - TICK_LINE_SIZE, p.y + tick.position);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x - width - TICK_LINE_SIZE * 2,
        p.y + tick.position + height / 2
      );
    });
    if (showGrid) {
      canvas.strokeStyle = this.state.gridColor.hex;
      ticks
        .filter((tick) => tick.position !== 0 && tick.position !== size)
        .forEach((tick) => {
          canvas.beginPath();
          canvas.moveTo(p.x, p.y + tick.position);
          canvas.lineTo(p.x + gridSize, p.y + tick.position);
          canvas.stroke();
        });
    }
  }

  drawRight(ctx: AxisContext): void {
    const { canvas } = this.ctx;
    const { plottingRegion } = ctx;
    const { position: _p, showGrid } = this.state;
    const size = plottingRegion.height;
    const gridSize = plottingRegion.width;
    const p = { x: _p.x, y: _p.y };

    canvas.beginPath();
    canvas.moveTo(p.x, p.y);
    canvas.lineTo(p.x, p.y + size);
    canvas.stroke();

    const ticks = this.tickFactory.generate({ ...ctx, size });

    ticks.forEach((tick) => {
      canvas.moveTo(p.x, p.y + tick.position);
      canvas.lineTo(p.x + 5, p.y + tick.position);
      canvas.stroke();
      canvas.fillText(tick.label, p.x + 10, p.y + tick.position + 5);
    });

    if (showGrid) {
      canvas.strokeStyle = this.state.gridColor.hex;
      ticks
        .filter((tick) => tick.position !== 0 && tick.position !== size)
        .forEach((tick) => {
          canvas.beginPath();
          canvas.moveTo(p.x, p.y + tick.position);
          canvas.lineTo(p.x - gridSize, p.y + tick.position);
          canvas.stroke();
        });
    }
  }
}
