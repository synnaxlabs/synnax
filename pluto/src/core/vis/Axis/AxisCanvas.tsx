// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dimensions, XY } from "@synnaxlabs/x";

import { textDimensions } from "@/core/std/Typography/textDimensions";
import {
  AxisContext,
  axisState,
  AxisState,
  ParsedAxisState,
  X_AXIS_SIZE,
  Y_AXIS_SIZE,
} from "@/core/vis/Axis/core";
import { Tick, TickFactory, newTickFactory } from "@/core/vis/Axis/TickFactory";
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

    switch (this.state.location.v) {
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
    const size = plottingRegion.width;
    const gridSize = plottingRegion.height;
    const p = this.state.position;
    this.drawLine(p, p.translateX(size));
    const ticks = this.tickFactory.generate({ ...ctx, size });
    this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(p.x + tick.position, p.y);
      canvas.lineTo(p.x + tick.position, p.y + TICK_LINE_SIZE);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x + tick.position - d.width / 2,
        p.y + 5 + d.height
      );
    });
    this.maybeDrawGrid(size, ticks, (tick) => [
      p.translateX(tick.position),
      p.translate({ x: tick.position, y: -gridSize }),
    ]);
  }

  drawTop(ctx: AxisContext): void {
    const { canvas } = this.ctx;
    const { plottingRegion } = ctx;
    const size = plottingRegion.width;
    const gridSize = plottingRegion.height;
    const p = this.state.position.translateY(Y_AXIS_SIZE);
    this.drawLine(p, p.translateX(size));
    const ticks = this.tickFactory.generate({ ...ctx, size });
    this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(p.x + tick.position, p.y);
      canvas.lineTo(p.x + tick.position, p.y - TICK_LINE_SIZE);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x + tick.position - d.width / 2,
        p.y - 5 - d.height
      );
    });
    this.maybeDrawGrid(size, ticks, (tick) => [
      p.translateX(tick.position),
      p.translate({ x: tick.position, y: gridSize }),
    ]);
  }

  drawLeft(ctx: AxisContext): void {
    const { canvas } = this.ctx;
    const { plottingRegion } = ctx;
    const size = plottingRegion.height;
    const gridSize = plottingRegion.width;
    const p = this.state.position.translateX(X_AXIS_SIZE);
    this.drawLine(p, p.translateY(size));
    const ticks = this.tickFactory.generate({ ...ctx, size });
    this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(p.x, p.y + tick.position);
      canvas.lineTo(p.x - TICK_LINE_SIZE, p.y + tick.position);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x - d.width - TICK_LINE_SIZE * 2,
        p.y + tick.position + d.height / 2
      );
    });
    this.maybeDrawGrid(size, ticks, (tick) => [
      p.translateY(tick.position),
      p.translate({ x: gridSize, y: tick.position }),
    ]);
  }

  drawRight(ctx: AxisContext): void {
    const { canvas } = this.ctx;
    const { plottingRegion } = ctx;
    const size = plottingRegion.height;
    const gridSize = plottingRegion.width;
    const p = this.state.position;
    canvas.beginPath();
    canvas.moveTo(p.x, p.y);
    canvas.lineTo(p.x, p.y + size);
    canvas.stroke();
    const ticks = this.tickFactory.generate({ ...ctx, size });
    this.drawTicks(ticks, (_, tick) => {
      canvas.moveTo(p.x, p.y + tick.position);
      canvas.lineTo(p.x + 5, p.y + tick.position);
      canvas.stroke();
      canvas.fillText(tick.label, p.x + 10, p.y + tick.position + 5);
    });
    this.maybeDrawGrid(size, ticks, (tick) => [
      p.translateY(tick.position),
      p.translate({ x: -gridSize, y: tick.position }),
    ]);
  }

  private drawLine(start: XY, end: XY): void {
    this.ctx.canvas.beginPath();
    this.ctx.canvas.moveTo(...start.couple);
    this.ctx.canvas.lineTo(...end.couple);
    this.ctx.canvas.stroke();
  }

  private drawTicks(
    ticks: Tick[],
    f: (textDimensions: Dimensions, tick: Tick) => void
  ): void {
    ticks.forEach((tick) =>
      f(textDimensions(tick.label, this.state.font, this.ctx.canvas), tick)
    );
  }

  private maybeDrawGrid(
    size: number,
    ticks: Tick[],
    f: (tick: Tick) => [XY, XY]
  ): void {
    const { showGrid, gridColor } = this.state;
    if (showGrid) {
      this.ctx.canvas.strokeStyle = gridColor.hex;
      ticks
        .filter((tick) => tick.position !== 0 && tick.position !== size)
        .forEach((tick) => this.drawLine(...f(tick)));
    }
  }
}
