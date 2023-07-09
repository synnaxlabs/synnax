// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bounds, Dimensions, XY } from "@synnaxlabs/x";

import { textDimensions } from "@/core/std/Typography/textDimensions";
import {
  Axis,
  AxisProps,
  AxisRenderResult,
  axisState,
  AxisState,
  ParsedAxisState,
} from "@/core/vis/Axis/core";
import { Tick, TickFactory, newTickFactory } from "@/core/vis/Axis/TickFactory";
import { RenderContext } from "@/core/vis/render";

const TICK_LINE_SIZE = 4;

export class AxisCanvas implements Axis {
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

  render(props: AxisProps): AxisRenderResult {
    const { lower2d: canvas } = this.ctx;
    canvas.font = this.state.font;
    canvas.fillStyle = this.state.color.hex;

    switch (this.state.location.crude) {
      case "left":
        return this.drawLeft(props);
      case "right":
        return this.drawRight(props);
      case "top":
        return this.drawTop(props);
      default:
        return this.drawBottom(props);
    }
  }

  drawBottom(ctx: AxisProps): AxisRenderResult {
    const { lower2d: canvas } = this.ctx;
    const { plottingRegion } = ctx;
    const size = plottingRegion.width;
    const gridSize = plottingRegion.height;
    const p = ctx.position;
    const ticks = this.tickFactory.generate({ ...ctx, size });
    this.maybeDrawGrid(size, ticks, (tick) => [
      p.translateX(tick.position),
      p.translate({ x: tick.position, y: -gridSize }),
    ]);
    canvas.strokeStyle = this.state.color.hex;
    this.drawLine(p, p.translateX(size));
    const maxTickDims = this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(p.x + tick.position, p.y);
      canvas.lineTo(p.x + tick.position, p.y + TICK_LINE_SIZE);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x + tick.position - d.width / 2,
        p.y + 5 + d.height
      );
    });
    return { size: maxTickDims.height + TICK_LINE_SIZE };
  }

  drawTop(ctx: AxisProps): AxisRenderResult {
    const { lower2d: canvas } = this.ctx;
    const { plottingRegion } = ctx;
    const size = plottingRegion.width;
    const gridSize = plottingRegion.height;
    const p = ctx.position.translateY(this.state.size);
    const ticks = this.tickFactory.generate({ ...ctx, size });
    this.maybeDrawGrid(size, ticks, (tick) => [
      p.translateX(tick.position),
      p.translate({ x: tick.position, y: gridSize }),
    ]);
    canvas.strokeStyle = this.state.color.hex;
    this.drawLine(p, p.translateX(size));
    const maxTickDims = this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(p.x + tick.position, p.y);
      canvas.lineTo(p.x + tick.position, p.y - TICK_LINE_SIZE);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x + tick.position - d.width / 2,
        p.y - 5 - d.height
      );
    });

    return { size: maxTickDims.height + TICK_LINE_SIZE };
  }

  drawLeft(ctx: AxisProps): AxisRenderResult {
    const { lower2d: canvas } = this.ctx;
    const { plottingRegion } = ctx;
    const size = plottingRegion.height;
    const gridSize = plottingRegion.width;
    const p = ctx.position.translateX(this.state.size);
    const ticks = this.tickFactory.generate({ ...ctx, size });
    this.maybeDrawGrid(size, ticks, (tick) => [
      p.translateY(tick.position),
      p.translate({ x: gridSize, y: tick.position }),
    ]);
    canvas.strokeStyle = this.state.color.hex;
    this.drawLine(p, p.translateY(size));
    const maxTickSize = this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(p.x, p.y + tick.position);
      canvas.lineTo(p.x - TICK_LINE_SIZE, p.y + tick.position);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x - d.width - TICK_LINE_SIZE * 2,
        p.y + tick.position + d.height / 2
      );
    });

    return { size: maxTickSize.width + TICK_LINE_SIZE };
  }

  drawRight(ctx: AxisProps): AxisRenderResult {
    const { lower2d: canvas } = this.ctx;
    const { plottingRegion } = ctx;
    const size = plottingRegion.height;
    const gridSize = plottingRegion.width;
    const p = ctx.position;
    const ticks = this.tickFactory.generate({ ...ctx, size });
    this.maybeDrawGrid(size, ticks, (tick) => [
      p.translateY(tick.position),
      p.translate({ x: -gridSize, y: tick.position }),
    ]);
    canvas.strokeStyle = this.state.color.hex;
    this.drawLine(p, p.translateY(size));
    const maxTickSize = this.drawTicks(ticks, (_, tick) => {
      canvas.moveTo(p.x, p.y + tick.position);
      canvas.lineTo(p.x + 5, p.y + tick.position);
      canvas.stroke();
      canvas.fillText(tick.label, p.x + 10, p.y + tick.position + 5);
    });
    return { size: maxTickSize.width + TICK_LINE_SIZE };
  }

  private drawLine(start: XY, end: XY): void {
    const { lower2d: canvas } = this.ctx;
    canvas.beginPath();
    canvas.moveTo(...start.couple);
    canvas.lineTo(...end.couple);
    canvas.stroke();
  }

  private drawTicks(
    ticks: Tick[],
    f: (textDimensions: Dimensions, tick: Tick) => void
  ): Dimensions {
    let maxDimensions = Dimensions.ZERO;
    ticks.forEach((tick) => {
      const d = textDimensions(tick.label, this.state.font, this.ctx.lower2d);
      maxDimensions = maxDimensions.pickGreatest(d);
      f(d, tick);
    });
    return maxDimensions;
  }

  private maybeDrawGrid(
    size: number,
    ticks: Tick[],
    f: (tick: Tick) => [XY, XY]
  ): void {
    const { showGrid, gridColor } = this.state;
    if (showGrid) {
      const startBound = new Bounds(-1, 1);
      const endBound = new Bounds(size - 1, size + 1);
      this.ctx.lower2d.strokeStyle = gridColor.hex;
      ticks
        .filter(
          ({ position }) =>
            !startBound.contains(position) && !endBound.contains(position)
        )
        .forEach((tick) => this.drawLine(...f(tick)));
    }
  }
}
